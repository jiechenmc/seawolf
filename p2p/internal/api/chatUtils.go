package api

import (
    "fmt"
    "log"
    "sync"
    "time"
    "context"
    "strconv"
    "strings"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/core/network"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    cid "github.com/ipfs/go-cid"
)

const chatProtocol = "/orcanet/p2p/seawolf/chat"
const chatRequestTimeout = time.Second * 120
const chatIdleTimeout = time.Second * 60 * 10

//Chat room statuses
const (
    ONGOING = "ongoing"
    FINISHED = "finished"
    TIMEOUT = "timed out"
    ERROR = "error"
)

//Chat request statuses
const (
    PENDING = "pending"
    DECLINED = "declined"
    ACCEPTED = "accepted"
)

type Message struct {
    Timestamp time.Time `json:"timestamp"`
    From peer.ID        `json:"from"`
    Text string         `json:"text"`
}

type ChatRequest struct {
    RequestID int       `json:"request_id"`
    PeerID peer.ID      `json:"peer_id"`
    FileCidStr string   `json:"file_cid"`
    Status string       `json:"status"`
    fileCid cid.Cid
    stream *P2PStream
}

type ChatRoom struct {
    ChatID int          `json:"chat_id"`
    Buyer peer.ID       `json:"buyer"`
    Seller peer.ID      `json:"seller"`
    FileCidStr string   `json:"file_cid"`
    Messages []Message  `json:"messages"`
    Status string       `json:"status"`
    fileCid cid.Cid
    chatLock sync.Mutex
    stream *P2PStream
}

type ChatNode struct {
    host host.Host
    kadDHT *dht.IpfsDHT
    fsNode *FileShareNode
    chats map[peer.ID]map[int]*ChatRoom
    chatsLock sync.Mutex
    outgoingRequests map[int]*ChatRequest
    outgoingRequestsLock sync.Mutex
    incomingRequests map[peer.ID]map[int]*ChatRequest
    incomingRequestsLock sync.Mutex
    currChatID int
}

func ChatNodeCreate(hostNode host.Host, kadDHT *dht.IpfsDHT, fsNode *FileShareNode) *ChatNode {
    cn := &ChatNode {
        host: hostNode,
        kadDHT: kadDHT,
        fsNode: fsNode,
        chats: make(map[peer.ID]map[int]*ChatRoom),
        outgoingRequests: make(map[int]*ChatRequest),
        incomingRequests: make(map[peer.ID]map[int]*ChatRequest),
        currChatID: 0,
    }
    hostNode.SetStreamHandler(chatProtocol, cn.ChatStreamHandler)
    return cn
}

func (cn *ChatNode) ChatStreamHandler(s network.Stream) {
    stream := p2pWrapStream(&s)
    req, err := stream.ReadString('\n', chatRequestTimeout)
    if err != nil {
        return
    }

    switch req {
        case "REQUEST\n":
            err = cn.handleChatRequest(context.Background(), stream)
            if err != nil {
                stream.Close()
            }
        default:
            stream.Close()
    }
}


/*
    Protocol
                   REQUEST
    Buyer -------------------------> Seller
               FileCid, ChatID

                ACCEPT or DECLINE
    Buyer <------------------------- Seller
                ChatID(If ACCEPT)

                 (IF ACCEPT)
            (Choose larger ChatID)
    Buyer <------------------------> Seller   
*/
func (cn *ChatNode) handleChatRequest(ctx context.Context, stream *P2PStream) error {
    fileCidStr, err := stream.ReadString('\n', chatRequestTimeout)
    if err != nil {
        return err
    }
    fileCid, err := cid.Decode(fileCidStr[:len(fileCidStr) - 1])
    if err != nil {
        return err
    }
    reqChatIDStr, err := stream.ReadString('\n', chatRequestTimeout)
    if err != nil {
        return err
    }
    reqChatID, err := strconv.Atoi(reqChatIDStr[:len(reqChatIDStr) - 1])
    if err != nil {
        return err
    }

    if (cn.fsNode.HasFile(fileCid)) {
        respChatID := 0
        if (reqChatID >= cn.currChatID) {
            respChatID = reqChatID
        } else {
            respChatID = cn.currChatID
            cn.currChatID++
        }
        cn.CreateIncomingRequest(respChatID, fileCid, stream)
        return nil
    } else {
        return contentNotFound
    }
}

func (cn *ChatNode) DeclineRequest(peerIDStr string, reqID int) error {
    peerID, err := peer.Decode(peerIDStr)
    if err != nil {
        log.Printf("Failed to decode peer ID string '%v'. %v\n", peerIDStr, err)
        return invalidParams
    }

    cn.incomingRequestsLock.Lock()
    peerRequests, ok := cn.incomingRequests[peerID]
    if ok {
        request, ok := peerRequests[reqID]
        if ok {
            if request.Status == PENDING {
                request.Status = DECLINED
                cn.incomingRequestsLock.Unlock()

                var builder strings.Builder
                builder.WriteString(fmt.Sprintf("DECLINE\n%d\n", reqID))
                err = request.stream.SendString(builder.String())
                if err != nil {
                    cn.ResolveIncomingRequest(reqID, peerID, DECLINED)
                    return err
                }
                cn.ResolveIncomingRequest(reqID, peerID, DECLINED)
                return nil
            }
        }
    }
    cn.incomingRequestsLock.Unlock()
    return requestNotFound
}

func (cn *ChatNode) AcceptRequest(peerIDStr string, reqID int) (*ChatRoom, error) {
    peerID, err := peer.Decode(peerIDStr)
    if err != nil {
        log.Printf("Failed to decode peer ID string '%v'. %v\n", peerIDStr, err)
        return nil, invalidParams
    }

    cn.incomingRequestsLock.Lock()
    peerRequests, ok := cn.incomingRequests[peerID]
    if ok {
        request, ok := peerRequests[reqID]
        if ok {
            if request.Status == PENDING {
                request.Status = ACCEPTED
                cn.incomingRequestsLock.Unlock()

                var builder strings.Builder
                builder.WriteString(fmt.Sprintf("ACCEPT\n%d\n", reqID))
                err = request.stream.SendString(builder.String())
                if err != nil {
                    cn.ResolveIncomingRequest(reqID, peerID, DECLINED)
                    return nil, err
                }
                cn.ResolveIncomingRequest(reqID, peerID, ACCEPTED)
                chatRoom := cn.CreateChatRoom(reqID, peerID, cn.host.ID(), request.fileCid, request.stream)
                return chatRoom, nil
            }
        }
    }
    cn.incomingRequestsLock.Unlock()
    return nil, requestNotFound
}

func (cn *ChatNode) SendRequest(ctx context.Context, providerIDStr string, fileCidStr string) (*ChatRequest, error) {
    fileCid, err := cid.Decode(fileCidStr)
    if err != nil {
        log.Printf("Failed to decode cid %v. %v\n", fileCidStr, err)
        return nil, invalidParams
    }

    providerID, err := peer.Decode(providerIDStr)
    if err != nil {
        log.Printf("Failed to decode provider ID string '%v'. %v\n", providerIDStr, err)
        return nil, invalidParams
    }

    timeoutCtx, cancel := context.WithTimeout(ctx, chatRequestTimeout)
    stream, err := p2pOpenStream(timeoutCtx, chatProtocol, cn.host, cn.kadDHT, providerIDStr)
    cancel()
    if err != nil {
        return nil, err
    }
    var builder strings.Builder
    builder.WriteString(fmt.Sprintf("REQUEST\n%s\n%d\n", fileCidStr, cn.currChatID))

    err = stream.SendString(builder.String())
    if err != nil {
        return nil, err
    }

    request := cn.CreateOutgoingRequest(cn.currChatID, fileCid, stream)

    //Wait for response
    go func(currChatID int) {
        resp, err := stream.ReadString('\n', chatRequestTimeout)
        if err != nil {
            goto declined
        }

        if resp == "ACCEPT\n" {
            respChatIDStr, err := stream.ReadString('\n', chatRequestTimeout)
            if err != nil {
                goto declined
            }
            respChatID, err := strconv.Atoi(respChatIDStr[:len(respChatIDStr) - 1])
            if err != nil {
                goto declined
            }
            cn.ResolveOutgoingRequest(currChatID, ACCEPTED)
            cn.CreateChatRoom(respChatID, cn.host.ID(), providerID, fileCid, stream)
            return
        } else {
            goto declined
        }
declined:
        cn.ResolveOutgoingRequest(currChatID, DECLINED)
    }(cn.currChatID)
    cn.currChatID++
    return request, nil
}

/*
    Protocol
                   MESSAGE
    Sender -------------------------> Receiver
                    Text

*/
func (chatRoom *ChatRoom) handleMessage() error {
    // No need to lock for read because only one thread should be calling read on stream
    text, err := chatRoom.stream.ReadString('\n', chatRequestTimeout)
    if err != nil {
        return err
    }
    message := Message{
        Timestamp: time.Now().UTC(),
        From: chatRoom.stream.RemotePeerID,
        Text: text[:len(text) - 1],
    }
    chatRoom.chatLock.Lock()
    chatRoom.Messages = append(chatRoom.Messages, message)
    chatRoom.chatLock.Unlock()
    return nil
}

func (cn *ChatNode) SendMessage(remotePeerIDStr string, chatID int, text string) (*Message, error) {
    remotePeerID, err := peer.Decode(remotePeerIDStr)
    if err != nil {
        log.Printf("Failed to decode remote peer ID string '%v'. %v\n", remotePeerIDStr, err)
        return nil, invalidParams
    }
    cn.chatsLock.Lock()
    peerChats, ok := cn.chats[remotePeerID]
    if !ok {
        cn.chatsLock.Unlock()
        return nil, chatNotFound
    }
    chat, ok := peerChats[chatID]
    cn.chatsLock.Unlock()
    if !ok {
        return nil, chatNotFound
    }

    chat.chatLock.Lock()
    defer chat.chatLock.Unlock()
    if chat.Status != ONGOING {
        return nil, chatNotOngoing
    }

    var builder strings.Builder
    builder.WriteString(fmt.Sprintf("MESSAGE\n%s\n", text))

    err = chat.stream.SendString(builder.String())
    if err != nil {
        return nil, failedToSendMessage
    }
    message := Message{
        Timestamp: time.Now().UTC(),
        From: remotePeerID,
        Text: text,
    }
    chat.Messages = append(chat.Messages, message)

    return &message, nil
}

func (cn *ChatNode) CloseChat(remotePeerIDStr string, chatID int) (*ChatRoom, error) {
    chat, err := cn.GetChat(remotePeerIDStr, chatID, false)
    if err != nil {
        return nil, err
    }
    chat.chatLock.Lock()
    defer chat.chatLock.Unlock()
    if chat.Status == ONGOING {
        err = chat.stream.SendString("CLOSE\n")
        if err != nil {
            chat.Status = ERROR
            chat.Close()
            return nil, err
        }
        chat.Status = FINISHED
        return chat, nil
    } else {
        return nil, chatNotOngoing
    }
}

func (cn *ChatNode) GetMessages(remotePeerIDStr string, chatID int) ([]Message, error) {
    chat, err := cn.GetChat(remotePeerIDStr, chatID, false)
    if err != nil {
        return nil, err
    }
    return chat.Messages, nil
}

func (cn *ChatNode) GetChat(remotePeerIDStr string, chatID int, makeCopy bool) (*ChatRoom, error) {
    remotePeerID, err := peer.Decode(remotePeerIDStr)
    if err != nil {
        log.Printf("Failed to decode remote peer ID string '%v'. %v\n", remotePeerIDStr, err)
        return nil, invalidParams
    }
    cn.chatsLock.Lock()
    peerChats, ok := cn.chats[remotePeerID]
    if !ok {
        cn.chatsLock.Unlock()
        return nil, chatNotFound
    }
    chat, ok := peerChats[chatID]
    cn.chatsLock.Unlock()
    if !ok {
        return nil, chatNotFound
    }
    if makeCopy {
        chat.chatLock.Lock()
        chatCpy := *chat
        chat.chatLock.Unlock()
        chat = &chatCpy
    }
    return chat, nil
}

func (cn *ChatNode) GetChats() []ChatRoom {
    chats := []ChatRoom{}
    for _, peerChats := range cn.chats {
        for _, chat := range peerChats {
            chat.chatLock.Lock()
            chats = append(chats, *chat)
            chat.chatLock.Unlock()
        }
    }
    return chats
}

func (cn *ChatNode) CreateChatRoom(id int, buyer peer.ID, seller peer.ID, fileCid cid.Cid, p2pStream *P2PStream) *ChatRoom {
    chatRoom := &ChatRoom{
        ChatID: id,
        Buyer: buyer,
        Seller: seller,
        fileCid: fileCid,
        FileCidStr: fileCid.String(),
        Messages: []Message{},
        Status: ONGOING,
        stream: p2pStream,
    }
    cn.chatsLock.Lock()
    peerChats, ok := cn.chats[p2pStream.RemotePeerID]
    if !ok {
        peerChats = make(map[int]*ChatRoom)
        cn.chats[p2pStream.RemotePeerID] = peerChats
    }
    peerChats[id] = chatRoom
    cn.chatsLock.Unlock()

    go chatRoom.StreamHandler()

    return chatRoom
}

func (cn *ChatNode) CreateOutgoingRequest(id int, fileCid cid.Cid, p2pStream *P2PStream) *ChatRequest {
    request := &ChatRequest{
        RequestID: id,
        PeerID: p2pStream.RemotePeerID,
        fileCid: fileCid,
        FileCidStr: fileCid.String(),
        Status: PENDING,
        stream: p2pStream,
    }
    cn.outgoingRequestsLock.Lock()
    defer cn.outgoingRequestsLock.Unlock()
    cn.outgoingRequests[id] = request
    return request
}

func (cn *ChatNode) ResolveOutgoingRequest(id int, status string) {
    cn.outgoingRequestsLock.Lock()
    defer cn.outgoingRequestsLock.Unlock()
    request, ok := cn.outgoingRequests[id]
    if ok {
        request.Status = status
        if status == DECLINED {
            request.stream.Close()
        }
    }
}

func (cn *ChatNode) GetOutgoingRequests() []*ChatRequest {
    requests := []*ChatRequest{}
    cn.outgoingRequestsLock.Lock()
    defer cn.outgoingRequestsLock.Unlock()
    for _, request := range cn.outgoingRequests {
        requests = append(requests, request)
    }
    return requests
}

func (cn *ChatNode) CreateIncomingRequest(id int, fileCid cid.Cid, p2pStream *P2PStream) *ChatRequest {
    request := &ChatRequest{
        RequestID: id,
        PeerID: p2pStream.RemotePeerID,
        fileCid: fileCid,
        FileCidStr: fileCid.String(),
        stream: p2pStream,
        Status: PENDING,
    }
    cn.incomingRequestsLock.Lock()
    defer cn.incomingRequestsLock.Unlock()
    peerRequests, ok := cn.incomingRequests[p2pStream.RemotePeerID]
    if !ok {
        peerRequests = make(map[int]*ChatRequest)
        cn.incomingRequests[p2pStream.RemotePeerID] = peerRequests
    }
    peerRequests[id] = request
    return request
}

func (cn *ChatNode) ResolveIncomingRequest(id int, peerID peer.ID, status string) {
    cn.outgoingRequestsLock.Lock()
    defer cn.outgoingRequestsLock.Unlock()
    peerRequests, ok := cn.incomingRequests[peerID]
    if ok {
        request, ok := peerRequests[id]
        if ok {
            request.Status = status
            if status == DECLINED {
                request.stream.Close()
            }
        }
    }
}

func (cn *ChatNode) GetIncomingRequests() []*ChatRequest {
    requests := []*ChatRequest{}
    cn.incomingRequestsLock.Lock()
    defer cn.incomingRequestsLock.Unlock()
    for _, peerRequests := range cn.incomingRequests {
        for _, request := range peerRequests {
            requests = append(requests, request)
        }
    }
    return requests
}

func (chatRoom *ChatRoom) StreamHandler() {
    for {
        req, err := chatRoom.stream.ReadString('\n', chatIdleTimeout)
        if err != nil {
            goto close
        }

        switch req {
            case "MESSAGE\n":
                err = chatRoom.handleMessage()
                if err != nil {
                    goto close
                }
            case "CLOSE\n":
                chatRoom.chatLock.Lock()
                chatRoom.Status = FINISHED
                chatRoom.chatLock.Unlock()
                goto close
        }
    }
close:
    chatRoom.chatLock.Lock()
    defer chatRoom.chatLock.Unlock()
    if chatRoom.Status != FINISHED {
        chatRoom.Status = ERROR
    }
    chatRoom.Close()
}

func (chatRoom *ChatRoom) Close() {
    chatRoom.stream.Close()
}
