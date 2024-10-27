package api

import (
    "time"
    "os"
    "io"
    "fmt"
    "log"
    "context"
    "sync"
    "strconv"
    "strings"
    "encoding/binary"
    "path/filepath"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/core/network"
    "github.com/ipfs/boxo/bitswap"
    "github.com/ipfs/boxo/blockservice"
    "github.com/ipfs/go-datastore"
    libbytes "bytes"
    bsnetwork "github.com/ipfs/boxo/bitswap/network"
    dssync "github.com/ipfs/go-datastore/sync"
    pb "github.com/ipfs/boxo/ipld/merkledag/pb"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    ipld "github.com/ipfs/go-ipld-format"
    dag "github.com/ipfs/boxo/ipld/merkledag"
    cid "github.com/ipfs/go-cid"
    blockstore "github.com/ipfs/boxo/blockstore"
)

const fileShareProtocol = "/orcanet/p2p/seawolf/fileshare"
const fileShareWantHaveTimeout = time.Second * 5
const fileShareWantTimeout = time.Second * 10
const fileShareFindProvidersTimeout = time.Second * 1
const fileShareIdleTimeout = time.Second * 60

var nextSessionIDLock sync.Mutex
var nextSessionID = 0
var chunkSize = 256 * 1024
var dagMaxChildren = 10
var comboService *dag.ComboService = nil

type FileShareNode struct {
    Host host.Host
    DHT *dht.IpfsDHT
    bstore blockstore.Blockstore
    fstore map[cid.Cid]dag.ProtoNode
    sessionStore map[int]*FileShareSession
    rSessionStore map[peer.ID]map[int]*FileShareRemoteSession
    fstoreLock sync.Mutex
    sessionStoreLock sync.Mutex
    rSessionStoreLock sync.Mutex
}

type Pausable struct {
    pauseLock sync.Mutex
    paused int
    resumeChannel chan bool
}

type FileShareSession struct {
    SessionID int                   `json:"session_id"`
    ReqCid string                   `json:"req_cid"`
    RxBytes uint64                  `json:"rx_bytes"`
    Complete bool                   `json:"is_complete"`
    Result int                      `json:"result"`
    node *FileShareNode
    streamMap map[peer.ID]*P2PStream
    streamLock sync.Mutex
    statsLock sync.Mutex
    pausable *Pausable
    sessionContext context.Context
}

type FileShareRemoteSession struct {
    remoteSessionID int
    remotePeerID peer.ID
    txBytesLock sync.Mutex
    txBytes uint64
    pausable *Pausable
}

type FileShareFileInfo struct {
    Size uint64                   `json:"size"`
    Name string                   `json:"name"`
    Providers []FileShareProvider `json:"providers"`
}

type FileShareProvider struct {
    PeerID peer.ID          `json:"peer_id"`
    Price float64           `json:"price"`
}

type RootBlock struct {
    Size uint64             `json:"size"`
    Price float64           `json:"price"`
    Name string             `json:"name"`
}

func (r *RootBlock) Marshal() ([]byte, error) {
    var nameByteLen uint8
    if len(r.Name) > 255 {
        return nil, invalidParams
    } else {
        nameByteLen = uint8(len(r.Name))
    }
    bytes := make([]byte, 0, 8 + 8 + 1 + len(r.Name))
    bytes, _ = binary.Append(bytes, binary.BigEndian, r.Size)
    bytes, _ = binary.Append(bytes, binary.BigEndian, r.Price)
    bytes, _ = binary.Append(bytes, binary.BigEndian, nameByteLen)
    bytes, _ = binary.Append(bytes, binary.BigEndian, []byte(r.Name))
    return bytes, nil
}

func (r *RootBlock) Unmarshal(bytes []byte) error {
    var nameByteLen uint8

    buf := libbytes.NewReader(bytes)
    err := binary.Read(buf, binary.BigEndian, &r.Size)
    if err != nil {
        return invalidParams
    }
    err = binary.Read(buf, binary.BigEndian, &r.Price)
    if err != nil {
        return invalidParams
    }
    err = binary.Read(buf, binary.BigEndian, &nameByteLen)
    if err != nil {
        return invalidParams
    }
    if nameByteLen == 0 || len(bytes) < 8 + 8 + 1 + int(nameByteLen) {
        return invalidParams
    }
    r.Name = string(bytes[17:17 + nameByteLen])
    return nil
}

func NewPausable() *Pausable {
    return &Pausable{
        pauseLock: sync.Mutex{},
        paused: 0,
        resumeChannel: make(chan bool, 0),
    }
}

func (p *Pausable) Pause() {
    p.pauseLock.Lock()
    if p.paused == 0 {
        p.paused = 1
    }
    p.pauseLock.Unlock()
}

func (p *Pausable) Resume() {
    p.pauseLock.Lock()
    if p.paused != 0 {
        for ; p.paused > 1; {
            p.resumeChannel <- true
            p.paused --
        }
        p.paused = 0
    }
    p.pauseLock.Unlock()
}

func (p *Pausable) Wait() {
    p.pauseLock.Lock()
    if p.paused != 0 {
        p.paused ++
        p.pauseLock.Unlock()
        //Wait for resume
        <- p.resumeChannel
    } else {
        p.pauseLock.Unlock()
    }
}

func FileShareNodeCreate(node host.Host, kadDHT *dht.IpfsDHT) *FileShareNode {
    //Create datastore
    ds := datastore.NewMapDatastore()
    mds := dssync.MutexWrap(ds)

    //Create a blockstore
    blkStore := blockstore.NewBlockstore(mds)

    fsNode := &FileShareNode{
        Host: node,
        DHT: kadDHT,
        bstore: blkStore,
        fstore: make(map[cid.Cid]dag.ProtoNode),
        sessionStore: make(map[int]*FileShareSession),
        rSessionStore: make(map[peer.ID]map[int]*FileShareRemoteSession),
        fstoreLock: sync.Mutex{},
        sessionStoreLock: sync.Mutex{},
        rSessionStoreLock: sync.Mutex{},
    }

    node.SetStreamHandler(fileShareProtocol, func(s network.Stream) {
        stream := p2pWrapStream(&s)
        defer stream.Close()
        for {
            req, err := stream.ReadString('\n', fileShareIdleTimeout)
            if err != nil {
                return
            }

            switch req {
                case "WANT HAVE\n":
                    err = fsNode.handleWantHave(context.Background(), stream)
                    if err != nil {
                        return
                    }
                case "WANT\n":
                    err = fsNode.handleWant(context.Background(), stream)
                    if err != nil {
                        return
                    }
                case "WANT DATA\n":
                    err = fsNode.handleWantData(context.Background(), stream)
                    if err != nil {
                        return
                    }
                case "PAUSE\n":
                    err = fsNode.handlePause(stream)
                    if err != nil {
                        return
                    }
                case "RESUME\n":
                    err = fsNode.handleResume(stream)
                    if err != nil {
                        return
                    }
                case "DISCOVER\n":
                    err = fsNode.handleDiscover(stream)
                    if err != nil {
                        return
                    }
                case "CLOSE\n":
                    return
                default:
                    return
            }
        }
    })

    return fsNode
}

//Request:  "WANT HAVE\n<count>\n<cid1>\n<cid2>\n..."
//Response: "HAVE\n<count>\n<cid1>\n<cid2>\n..."
func (f *FileShareNode) handleWantHave(ctx context.Context, stream *P2PStream) error {
    countStr, err := stream.ReadString('\n', fileShareWantHaveTimeout)
    if err != nil {
        return err
    }
    count, err := strconv.Atoi(countStr[:len(countStr) - 1])
    if err != nil {
        return err
    }
    haveCids := []cid.Cid{}
    for i := 0; i < count; i ++ {
        cidStr, err := stream.ReadString('\n', fileShareWantHaveTimeout)
        if err != nil {
            return err
        }
        cid, err := cid.Decode(cidStr[:len(cidStr) - 1])
        if err != nil {
            return err
        }
        //Query local blockstore for cid
        has, err := f.bstore.Has(ctx, cid)
        if err != nil {
            return err
        }
        if has {
            haveCids = append(haveCids, cid)
        }
    }
    //Create HAVE response
    var builder strings.Builder
    builder.WriteString(fmt.Sprintf("HAVE\n%d\n", len(haveCids)))
    for _, c := range haveCids {
	    builder.WriteString(c.String())
	    builder.WriteString("\n")
    }

    err = stream.SendString(builder.String())
    return err
}

//Request:  "WANT\n<cid>\n"
//Response: "HERE\n<size>\n<byte1><byte2>..."
func (f *FileShareNode) handleWant(ctx context.Context, stream *P2PStream) error {
    //Get requested CID
    cidStr, err := stream.ReadString('\n', fileShareWantTimeout)
    if err != nil {
        return err
    }
    cid, err := cid.Decode(cidStr[:len(cidStr) - 1])
    if err != nil {
        return err
    }

    //Query local blockstore for CID
    has, err := f.bstore.Has(ctx, cid)
    if err == nil && has {
        block, err := f.bstore.Get(ctx, cid)
        if err != nil {
            goto Failed
        }
        node, err := dag.DecodeProtobufBlock(block)
        if err != nil {
            goto Failed
        }
        rawData := node.RawData()

        err = stream.SendString(fmt.Sprintf("HERE\n%d\n", len(rawData)))
        if err != nil {
            return err
        }
        err = stream.Send(rawData)
        if err != nil {
            return err
        }
    }

    return nil
Failed:
    stream.SendString("DON'T HAVE\n")
    return nil
}

//Request:  "WANT DATA\n<remote_session_id>\n<cid>\n"
//Response: "HERE\n<size>\n<byte1><byte2>..."
func (f *FileShareNode) handleWantData(ctx context.Context, stream *P2PStream) error {
    remoteSessionIDStr, err := stream.ReadString('\n', fileShareWantTimeout)
    if err != nil {
        return err
    }

    //Get remote session ID
    remoteSessionID, err := strconv.Atoi(remoteSessionIDStr[:len(remoteSessionIDStr) - 1])
    if err != nil {
        return err
    }

    //Get requested CID
    cidStr, err := stream.ReadString('\n', fileShareWantTimeout)
    if err != nil {
        return err
    }
    cid, err := cid.Decode(cidStr[:len(cidStr) - 1])
    if err != nil {
        return err
    }

    //Query local blockstore for CID
    has, err := f.bstore.Has(ctx, cid)
    if err == nil && has {
        block, err := f.bstore.Get(ctx, cid)
        if err != nil {
            goto Failed
        }
        node, err := dag.DecodeProtobuf(block.RawData())
        if err != nil {
            goto Failed
        }
        data := node.Data()

        rSession := f.RemoteSessionCreate(stream.RemotePeerID, remoteSessionID)
        defer f.RemoteSessionCleanup(rSession)

        err = stream.SendString(fmt.Sprintf("HERE\n%d\n", len(data)))
        if err != nil {
            return err
        }
        //Send the data chunk by chunk
        for byteOffset := 0; byteOffset < len(data); byteOffset += chunkSize {
            //If paused, wait till resumed
            rSession.pausable.Wait()

            txBytes := 0
            if (byteOffset + chunkSize) > len(data) {
                err = stream.Send(data[byteOffset:])
                txBytes += len(data) - byteOffset
            } else {
                err = stream.Send(data[byteOffset:byteOffset + chunkSize])
                txBytes += chunkSize
            }
            if err != nil {
                return err
            }
            rSession.txBytesLock.Lock()
            rSession.txBytes += uint64(txBytes)
            rSession.txBytesLock.Unlock()
        }
    }

    return nil
Failed:
    stream.SendString("DON'T HAVE\n")
    return nil
}

//Request: "RESUME\n<remote_session_id>\n"
func (f *FileShareNode) handleResume(stream *P2PStream) error {
    remoteSessionIDStr, err := stream.ReadString('\n', fileShareWantHaveTimeout)
    if err != nil {
        return err
    }
    //Get remote session ID
    remoteSessionID, err := strconv.Atoi(remoteSessionIDStr[:len(remoteSessionIDStr) - 1])
    if err != nil {
        return err
    }

    //Query for remote session
    f.rSessionStoreLock.Lock()
    _, ok := f.rSessionStore[stream.RemotePeerID]
    if !ok {
        return remoteSessionNotFound
    }
    rSession, ok := f.rSessionStore[stream.RemotePeerID][remoteSessionID]
    if !ok {
        return remoteSessionNotFound
    }
    f.rSessionStoreLock.Unlock()

    rSession.pausable.Resume()
    return nil
}

//Request: "PAUSE\n<remote_session_id>\n"
func (f *FileShareNode) handlePause(stream *P2PStream) error {
    remoteSessionIDStr, err := stream.ReadString('\n', fileShareWantHaveTimeout)
    if err != nil {
        return err
    }
    //Get remote session ID
    remoteSessionID, err := strconv.Atoi(remoteSessionIDStr[:len(remoteSessionIDStr) - 1])
    if err != nil {
        return err
    }

    //Query for remote session
    f.rSessionStoreLock.Lock()
    _, ok := f.rSessionStore[stream.RemotePeerID]
    if !ok {
        return remoteSessionNotFound
    }
    rSession, ok := f.rSessionStore[stream.RemotePeerID][remoteSessionID]
    if !ok {
        return remoteSessionNotFound
    }
    f.rSessionStoreLock.Unlock()

    rSession.pausable.Pause()
    return nil
}

//Request:  "DISCOVER\n<max_count>\n"
//Response: "KNOW\n<count>\n<cid1>\n<cid2>\n..."
func (f *FileShareNode) handleDiscover(stream *P2PStream) error {
    const myMaxCount = 1000

    maxCountStr, err := stream.ReadString('\n', fileShareWantHaveTimeout)
    if err != nil {
        return err
    }
    maxCount, err := strconv.Atoi(maxCountStr[:len(maxCountStr) - 1])
    if err != nil {
        return err
    }

    if maxCount > myMaxCount {
        maxCount = myMaxCount
    }

    knownCids := make([]cid.Cid, 0, maxCount)
    i := 0
    f.fstoreLock.Lock()
    for cid, _ := range f.fstore {
        knownCids = append(knownCids, cid)
        i ++
        if i == maxCount {
            break
        }
    }
    f.fstoreLock.Unlock()
    //Create KNOW response
    var builder strings.Builder
    builder.WriteString(fmt.Sprintf("KNOW\n%d\n", len(knownCids)))
    for _, c := range knownCids {
	    builder.WriteString(c.String())
	    builder.WriteString("\n")
    }

    err = stream.SendString(builder.String())
    return err
}


func (f *FileShareNode) SessionCreate(ctx context.Context, reqCid cid.Cid) *FileShareSession {
    nextSessionIDLock.Lock()
    sessionID := nextSessionID
    nextSessionID++
    nextSessionIDLock.Unlock()

    fileShareSession := &FileShareSession {
        SessionID: sessionID,
        node: f,
        streamMap: make(map[peer.ID]*P2PStream),
        streamLock: sync.Mutex{},
        sessionContext: ctx,
        pausable: NewPausable(),
        statsLock: sync.Mutex{},
        ReqCid: reqCid.String(),
        RxBytes: uint64(0),
        Complete: false,
        Result: 0,
    }

    f.sessionStoreLock.Lock()
    f.sessionStore[sessionID] = fileShareSession
    f.sessionStoreLock.Unlock()

    return fileShareSession
}

func (f *FileShareNode) SessionCleanup(session *FileShareSession, result int) {
    session.statsLock.Lock()
    session.Complete = true
    session.Result = result
    session.statsLock.Unlock()
}

func (f *FileShareNode) RemoteSessionCreate(remotePeerID peer.ID, remoteSessionID int) *FileShareRemoteSession {
    //If a remote session already exists, use it
    f.rSessionStoreLock.Lock()
    _, ok := f.rSessionStore[remotePeerID]
    if !ok {
        f.rSessionStore[remotePeerID] = make(map[int]*FileShareRemoteSession)
    }
    rSession, ok := f.rSessionStore[remotePeerID][remoteSessionID]
    if !ok {
        rSession = &FileShareRemoteSession{
            remoteSessionID: remoteSessionID,
            remotePeerID: remotePeerID,
            pausable: NewPausable(),
            txBytesLock: sync.Mutex{},
            txBytes: uint64(0),
        }
        f.rSessionStore[remotePeerID][remoteSessionID] = rSession
    }
    f.rSessionStoreLock.Unlock()
    return rSession
}

func (f *FileShareNode) RemoteSessionCleanup(remoteSession *FileShareRemoteSession) {
    f.rSessionStoreLock.Lock()
    defer f.rSessionStoreLock.Unlock()
    delete(f.rSessionStore[remoteSession.remotePeerID], remoteSession.remoteSessionID)
}

func (f *FileShareNode) GetSession(sessionID int) (*FileShareSession, error) {
    f.sessionStoreLock.Lock()
    session, ok := f.sessionStore[sessionID]
    f.sessionStoreLock.Unlock()
    if !ok {
        return nil, sessionNotFound
    }
    //Ensure we don't get any corrupted stats
    session.statsLock.Lock()
    sessionCpy := *session
    session.statsLock.Unlock()

    return &sessionCpy, nil
}

func (f *FileShareNode) PauseSession(ctx context.Context, sessionID int) error {
    f.sessionStoreLock.Lock()
    session, ok := f.sessionStore[sessionID]
    f.sessionStoreLock.Unlock()
    if !ok {
        return sessionNotFound
    }

    session.Pause(ctx)
    return nil
}

func (f *FileShareNode) ResumeSession(ctx context.Context, sessionID int) error {
    f.sessionStoreLock.Lock()
    session, ok := f.sessionStore[sessionID]
    f.sessionStoreLock.Unlock()
    if !ok {
        return sessionNotFound
    }

    session.Resume(ctx)
    return nil
}

func (s *FileShareSession) GetStream(peerID peer.ID) (*P2PStream, error) {
    s.streamLock.Lock()
    stream, ok := s.streamMap[peerID]
    s.streamLock.Unlock()
    if !ok {
        newStream, err := p2pOpenStream(s.sessionContext, fileShareProtocol, s.node.Host, peerID.String())
        if err == nil {
            s.streamLock.Lock()
            //If stream was created while we were attempting to create a new one, discard new stream
            stream, ok := s.streamMap[peerID]
            if !ok {
                s.streamMap[peerID] = newStream
                stream = newStream
            } else {
                newStream.Close()
            }
            s.streamLock.Unlock()
            return stream, err
        }
        return newStream, err
    }
    return stream, nil
}


func (s *FileShareSession) DeleteStream(peerID peer.ID) {
    s.streamLock.Lock()
    stream, ok := s.streamMap[peerID]
    if ok {
        stream.Close()
        delete(s.streamMap, peerID)
    }
    s.streamLock.Unlock()
}

func (s *FileShareSession) sendString(peerID peer.ID, str string) error {
    stream, err := s.GetStream(peerID)
    if err != nil {
        return err
    }
    err = stream.SendString(str)
    if err != nil {
        if err == network.ErrReset {
            s.DeleteStream(peerID)
        }
        return err
    }
    return nil
}

func (s *FileShareSession) readString(peerID peer.ID, delim byte, timeout time.Duration) (string, error) {
    stream, err := s.GetStream(peerID)
    if err != nil {
        return "", err
    }
    resp, err := stream.ReadString(delim, timeout)
    if err != nil {
        if err == network.ErrReset {
            s.DeleteStream(peerID)
        }
        return "", err
    }
    return resp, nil
}

func (s *FileShareSession) read(peerID peer.ID, n int, timeout time.Duration) ([]byte, error) {
    stream, err := s.GetStream(peerID)
    if err != nil {
        return nil, err
    }
    resp, err := stream.Read(n, timeout)
    if err != nil {
        if err == network.ErrReset {
            s.DeleteStream(peerID)
        }
        return nil, err
    }
    return resp, nil
}

func (s *FileShareSession) SendWantHave(peerID peer.ID, cids []cid.Cid) []cid.Cid {
    if len(cids) == 0 {
        return nil
    }

    //Create WANT HAVE request
    var builder strings.Builder
    builder.WriteString(fmt.Sprintf("WANT HAVE\n%d\n", len(cids)))
    for _, c := range cids {
	    builder.WriteString(c.String())
	    builder.WriteString("\n")
    }

    err := s.sendString(peerID, builder.String())
    if err != nil {
        return nil
    }

    //Wait for response
    resp, err := s.readString(peerID, '\n', fileShareWantHaveTimeout)
    if err != nil {
        return nil
    }

    //We only care about HAVE responses for now
    if resp == "HAVE\n" {
        countStr, err := s.readString(peerID, '\n', fileShareWantHaveTimeout)
        if err != nil {
            return nil
        }
        count, err := strconv.Atoi(countStr[:len(countStr) - 1])
        if err != nil {
            return nil
        }
        haveCIDs := make([]cid.Cid, count)
        for i := 0; i < count; i ++ {
            cidStr, err := s.readString(peerID, '\n', 0)
            if err != nil {
                return nil
            }
            haveCIDs[i], err = cid.Decode(cidStr[:len(cidStr) - 1])
            if err != nil {
                return nil
            }
        }
        return haveCIDs
    }

    return nil
}

func (s *FileShareSession) SendWant(peerID peer.ID, c cid.Cid) []byte {
    //Send WANT request
    err := s.sendString(peerID, fmt.Sprintf("WANT\n%s\n", c.String()))
    if err != nil {
        return nil
    }

    //Wait for response
    resp, err := s.readString(peerID, '\n', fileShareWantTimeout)
    if err != nil {
        return nil
    }

    //Response of the form HERE\n<size>\n<byte><byte>...
    if resp == "HERE\n" {
        sizeStr, err := s.readString(peerID, '\n', fileShareWantHaveTimeout)
        if err != nil {
            return nil
        }
        size, err := strconv.Atoi(sizeStr[:len(sizeStr) - 1])
        if err != nil {
            return nil
        }
        data, err := s.read(peerID, size, fileShareWantHaveTimeout)
        if err != nil {
            return nil
        }
        return data
    }

    return nil
}

func (s *FileShareSession) SendWantData(peerID peer.ID, c cid.Cid) chan []byte {
    //Send WANT DATA request
    err := s.sendString(peerID, fmt.Sprintf("WANT DATA\n%d\n%s\n", s.SessionID, c.String()))
    if err != nil {
        return nil
    }

    //Wait for response
    resp, err := s.readString(peerID, '\n', fileShareWantTimeout)
    if err != nil {
        return nil
    }

    //Response of the form HERE\n<size>\n<byte><byte>...
    if resp == "HERE\n" {
        sizeStr, err := s.readString(peerID, '\n', fileShareWantHaveTimeout)
        if err != nil {
            return nil
        }
        size, err := strconv.Atoi(sizeStr[:len(sizeStr) - 1])
        if err != nil {
            return nil
        }
        dataChannel := make(chan []byte)
        var chunkData []byte
        go func() {
            for byteOffset := 0; byteOffset < size; byteOffset += chunkSize {
                //If paused wait till resumed
                s.pausable.Wait()

                if (size - byteOffset) < chunkSize {
                    chunkData, err = s.read(peerID, size - byteOffset, fileShareWantHaveTimeout)
                } else {
                    chunkData, err = s.read(peerID, chunkSize, fileShareWantHaveTimeout)
                }
                if err != nil {
                    close(dataChannel)
                    return
                }
                dataChannel <- chunkData
                s.statsLock.Lock()
                s.RxBytes+= uint64(len(chunkData))
                s.statsLock.Unlock()
            }
            close(dataChannel)
        }()
        return dataChannel
    }
    return nil
}

func (s *FileShareSession) Pause(ctx context.Context) error {
    s.pausable.Pause()
    for peerID, _ := range s.streamMap {
        stream, err := p2pOpenStream(ctx, fileShareProtocol, s.node.Host, peerID.String())
        if err != nil {
            return err
        }
        err = stream.SendString(fmt.Sprintf("PAUSE\n%d\n", s.SessionID))
        err = stream.SendString("CLOSE\n")
        stream.Close()
    }
    return nil
}


func (s *FileShareSession) Resume(ctx context.Context) error {
    s.pausable.Resume()
    for peerID, _ := range s.streamMap {
        stream, err := p2pOpenStream(ctx, fileShareProtocol, s.node.Host, peerID.String())
        if err != nil {
            return err
        }
        err = stream.SendString(fmt.Sprintf("RESUME\n%d\n", s.SessionID))
        if err != nil {
            stream.Close()
            return err
        }
        err = stream.SendString("CLOSE\n")
        if err != nil {
            stream.Close()
            return err
        }
    }
    return nil
}

func (s *FileShareSession) SendDiscover(peerID peer.ID, maxCount int) []cid.Cid {
    //Create DISCOVER request
    err := s.sendString(peerID, fmt.Sprintf("DISCOVER\n%d\n", maxCount))
    if err != nil {
        return nil
    }

    //Wait for response
    resp, err := s.readString(peerID, '\n', fileShareWantHaveTimeout)
    if err != nil {
        return nil
    }

    //We only care about KNOW responses
    if resp == "KNOW\n" {
        countStr, err := s.readString(peerID, '\n', fileShareWantHaveTimeout)
        if err != nil {
            return nil
        }
        count, err := strconv.Atoi(countStr[:len(countStr) - 1])
        if err != nil {
            return nil
        }
        knownCIDs := make([]cid.Cid, count)
        for i := 0; i < count; i ++ {
            cidStr, err := s.readString(peerID, '\n', 0)
            if err != nil {
                return nil
            }
            knownCIDs[i], err = cid.Decode(cidStr[:len(cidStr) - 1])
            if err != nil {
                return nil
            }
        }
        return knownCIDs
    }

    return nil
}


func (f *FileShareNode) GetFile(ctx context.Context, providerIDStr string, rootCidStr string, outputFile string) (int, error) {
    rootCid, err := cid.Decode(rootCidStr)
    if err != nil {
        log.Printf("Failed to decode cid %v. %v", rootCid, err)
        return -1, invalidParams
    }

    providerID, err := peer.Decode(providerIDStr)
    if err != nil {
        log.Printf("Failed to decode provider ID string '%v'. %v\n", providerIDStr, err)
        return -1, invalidParams
    }

    tmpOutputFile := outputFile + ".tmp"

    //Open temporary file
    file, err := os.Create(tmpOutputFile)
    if err != nil {
        log.Printf("Error opening file: %v. %v\n", tmpOutputFile, err)
        return -1, failedToOpenFile
    }
    deferCleanup := true
    //Create a fileshare session
    session := f.SessionCreate(ctx, rootCid)
    defer func() {
        if deferCleanup {
            file.Close()
            f.SessionCleanup(session, 1)
        }
    }()

    var bytes []byte
    var dataChannel chan []byte

    isRoot := true
    reqCid := rootCid
    rootBlock := &RootBlock{}
    for {
        //Check local blockstore before asking peers
        has, err := f.bstore.Has(ctx, reqCid)
        if err != nil {
            return -1, internalError
        }
        if has {
            block, err := f.bstore.Get(ctx, reqCid)
            if err != nil {
                return -1, internalError
            }
            bytes = block.RawData()
            if !isRoot {
                dataChannel <- bytes
                close(dataChannel)
            }
        } else {
            if isRoot { 
                bytes = session.SendWant(providerID, reqCid)
                if bytes == nil {
                    log.Printf("Failed to get file metadata.\n")
                    return -1, internalError
                }
            } else {
                dataChannel = session.SendWantData(providerID, reqCid)
                if dataChannel == nil {
                    log.Printf("Failed to get file.\n")
                    return -1, internalError
                }
            }
        }

        if isRoot {
            protoNode, err := dag.DecodeProtobuf(bytes)
            if err != nil {
                log.Printf("Failed to parse bytes from provider.\n")
                return -1, internalError
            }
            //Get metadata and price
            err = rootBlock.Unmarshal(protoNode.Data())
            if err != nil {
                log.Printf("Failed to unmarshal file metadata.\n")
                return -1, internalError
            }

            log.Printf("Downloading file %v, size: %v bytes, price: %v\n", rootBlock.Name, rootBlock.Size, rootBlock.Price)

            isRoot = false
            links := protoNode.Links()
            if len(links) == 1 {
                reqCid = links[0].Cid
            } else {
                log.Printf("Unexpected links from node.\n")
                return -1, internalError
            }

            //Add root node to fstore
            f.fstoreLock.Lock()
            f.fstore[reqCid] = *protoNode
            f.fstoreLock.Unlock()
        } else {
            deferCleanup = false
            go func() {
                bytesWritten := uint64(0)
                for data := range dataChannel {
                    file.Write(data)
                    bytesWritten += uint64(len(data))
                }
                file.Close()
                if bytesWritten != rootBlock.Size {
                    f.SessionCleanup(session, 1)
                    log.Printf("Wrong number of bytes received\n")
                    return
                }
                err = os.Rename(outputFile + ".tmp", outputFile)
                if err != nil {
                    f.SessionCleanup(session, 1)
                    log.Printf("Failed to move temporary file to output file. %v\n")
                    return
                }
                f.SessionCleanup(session, 0)
            }()
            break
        }
    }
    return session.SessionID, nil
}

func (f *FileShareNode) PutFile(ctx context.Context, inputFile string, price float64) (cid.Cid, error) {
    //Open input file for reading
    file, err := os.OpenFile(inputFile, os.O_RDONLY, 0644)
    if err != nil {
        log.Printf("Error opening file: %v. %v\n", inputFile, err)
        return cid.Cid{}, failedToOpenFile
    }
    defer file.Close()
    buffer := []byte{}
    bytesRead := 0

    for {
        tempBuffer := make([]byte, chunkSize)
        n, err := file.Read(tempBuffer)
        if err != nil && err != io.EOF {
            log.Printf("Error reading file: %v. %v\n", inputFile, err)
            return cid.Cid{}, internalError
        } else {
            if n == 0 && err == io.EOF {
                break
            }
        }
        for i := 0; i < n; i ++ {
            buffer = append(buffer, tempBuffer[i])
        }
        bytesRead += n
    }

    //Create root node for metadata
    rootBlock := &RootBlock{ Size: uint64(bytesRead), Price: price, Name: filepath.Base(inputFile) }
    rootBlockBytes, err := rootBlock.Marshal()
    if err != nil {
        log.Printf("Failed to marshal root block. %v \n", err)
        return cid.Cid{}, nil
    }
    rootNode := dag.NodeWithData(rootBlockBytes)

    //Create data node and link it with root node
    node := dag.NodeWithData(buffer).Copy()
    rootNode.AddNodeLink("data", node)

    f.bstore.Put(ctx, node)
    f.bstore.Put(ctx, rootNode.Copy())

    f.fstoreLock.Lock()
    f.fstore[rootNode.Cid()] = *rootNode
    f.fstoreLock.Unlock()

    err = f.DHT.Provide(ctx, node.Cid(), true)
    if err != nil {
        log.Printf("Failed to provide cid. %v\n", err)
        return rootNode.Cid(), nil
    }
    err = f.DHT.Provide(ctx, rootNode.Cid(), true)
    if err != nil {
        log.Printf("Failed to provide cid. %v\n", err)
        return rootNode.Cid(), nil
    }

    return rootNode.Cid(), nil
}

func (f *FileShareNode) Discover(ctx context.Context) []FileShareFileInfo {
    session := f.SessionCreate(ctx, cid.Cid{})

    cidMapLock := sync.Mutex{}
    rootCidMap := make(map[cid.Cid]bool)

    p2pHost := f.Host
    peerIDs := p2pHost.Peerstore().Peers()
    wg := sync.WaitGroup{}
    wg.Add(len(peerIDs))

    //Iterate through node's known peers and send discover requests
    for _, peerID := range peerIDs {
        go func(peerID peer.ID) {
            cids := session.SendDiscover(peerID, 1000)
            if cids != nil {
                cidMapLock.Lock()
                for _, c := range cids {
                    rootCidMap[c] = true
                }
                cidMapLock.Unlock()
            }
            wg.Done()
        }(peerID)
    }
    wg.Wait()

    fileCidMap := make(map[string]*FileShareFileInfo)

    wg.Add(len(rootCidMap))
    //Iterate through all unique root Cids, get metadata from their providers, and compile a set of unique files(same data cid and file name)
    for rootCid, _ := range rootCidMap {
        go func() {
            ctxTimeout, cancel := context.WithTimeout(ctx, fileShareFindProvidersTimeout)
            providerChannel := f.DHT.FindProvidersAsync(ctxTimeout, rootCid, 10)
            defer cancel()
            for provider := range providerChannel {
                bytes := session.SendWant(provider.ID, rootCid)
                if bytes == nil {
                    continue
                }
                protoNode, err := dag.DecodeProtobuf(bytes)
                if err != nil {
                    //Ignore this provider
                    continue
                }
                links := protoNode.Links()
                //Get metadata and price
                rootBlock := RootBlock{}
                err = rootBlock.Unmarshal(protoNode.Data())
                if err != nil || len(links) != 1 {
                    //Ignore this provider
                    continue
                }
                dataCid := links[0].Cid
                dataCidNameStr := dataCid.String() + rootBlock.Name
                provider := FileShareProvider{
                    PeerID: provider.ID,
                    Price: rootBlock.Price,
                }

                //Add unique files into map and accumulate providers for each file
                cidMapLock.Lock()
                fileShareFileInfo, ok := fileCidMap[dataCidNameStr]
                if !ok {
                    fileShareFileInfo = &FileShareFileInfo{
                        Name: rootBlock.Name,
                        Size: rootBlock.Size,
                        Providers: []FileShareProvider{provider},
                    }
                    fileCidMap[dataCidNameStr] = fileShareFileInfo
                } else {
                    fileShareFileInfo.Providers = append(fileShareFileInfo.Providers, provider)
                }
                cidMapLock.Unlock()
            }
            wg.Done()
       }()
    }
    wg.Wait()

    fileInfos := make([]FileShareFileInfo, 0, len(fileCidMap))
    for _, fileInfo := range fileCidMap {
        fileInfos = append(fileInfos, *fileInfo)
    }

    return fileInfos
}

func bitswapCreate(ctx context.Context, node host.Host, kadDHT *dht.IpfsDHT) (*bitswap.Bitswap, *blockstore.Blockstore) {
    //Create datastore
    ds := datastore.NewMapDatastore()
    mds := dssync.MutexWrap(ds)

    //Create a blockstore
    bstore := blockstore.NewBlockstore(mds)

    //Create a bitswap network
    bsNetwork := bsnetwork.NewFromIpfsHost(node, kadDHT, bsnetwork.Prefix("/orcanet/p2p/seawolf"))

    //Create and return bitswap instance
    exchange := bitswap.New(network.WithAllowLimitedConn(ctx, "/orcanet/p2p/seawolf/bitswap"), bsNetwork, bstore)

    log.Printf("Successfully created bitswap instance\n")

    return exchange, &bstore
}

func bitswapGetFile(ctx context.Context, exchange *bitswap.Bitswap, bstore *blockstore.Blockstore, rootCid string, outputFile string) error {
    if comboService == nil {
        blockService := blockservice.New(*bstore, exchange)
        dagService := dag.NewDAGService(blockService)
        comboService = &dag.ComboService{ Read: dagService, Write: dagService }
    }

    root, err := cid.Decode(rootCid)
    if err != nil {
        log.Printf("Failed to decode cid %v. %v", rootCid, err)
        return invalidParams
    }

    tmpOutputFile := outputFile + ".tmp"

    //Open temporary file
    file, err := os.Create(tmpOutputFile)
    if err != nil {
        log.Printf("Error opening file: %v. %v\n", tmpOutputFile, err)
        return internalError
    }

    //Create new session
    session := dag.NewSession(ctx, comboService)

    //Keep a stack of node chans
    stack := make([]<- chan *ipld.NodeOption, 256)
    ctxCancelStack := make([] context.CancelFunc, 256)
    ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*10)
    stack[0] = session.GetMany(ctxTimeout, []cid.Cid{root})
    ctxCancelStack[0] = cancel
    stackPointer := 1

    //Chan used for async writes to disk
    dataNodeChan := make(chan ipld.Node, 128)
    errChan := make(chan error, 1)
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        pbNode := pb.PBNode{}
        for node := range dataNodeChan {
            err := pbNode.Unmarshal(node.RawData())
            if err != nil {
                log.Printf("Failed to unmarshal raw data. %v", err)
                errChan <- internalError
            }
            //We've reached a data node/block
            _, err = file.Write(pbNode.Data)
            if err != nil {
                log.Printf("Error writing to file: %v. %v\n", tmpOutputFile, err)
                errChan <- internalError
            }
        }
        errChan <- nil
    }()

    //Iterate the Merkle DAG in depth first search fashion
    for stackPointer != 0 {
        stackPointer --
        nodeChannel := stack[stackPointer]
        nodeOption := <- nodeChannel
        ctxCancelStack[stackPointer]()
        err = nodeOption.Err
        if err != nil {
            log.Printf("Failed to fetch node in Merkle DAG. %v\n", err)
            file.Close()
            if err == context.DeadlineExceeded {
                return timeoutError
            }
            return internalError
        }
        node := nodeOption.Node
        links := node.Links()
        if len(links) == 0 {
            dataNodeChan <- node
        } else {
            //Push the link Cid()s onto the stack in reverse order
            for i := len(links) - 1; i >= 0; i -- {
                ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*10)
                stack[stackPointer] = session.GetMany(ctxTimeout, []cid.Cid{links[i].Cid})
                ctxCancelStack[stackPointer] = cancel
                stackPointer ++
            }
        }
    }
    close(dataNodeChan)
    //Wait until writing thread is done writing
    wg.Wait()
    err = <- errChan
    if err != nil {
        return err
    }
    //Close temporary file and move it to final output file
    file.Close()

    err = os.Rename(outputFile + ".tmp", outputFile)
    if err != nil {
        log.Printf("Failed to move temporary file to output file. %v\n")
        return internalError
    }

    return nil
}


func protoNodesToIPLDNodes(protoNodes []dag.ProtoNode) []ipld.Node {
    ipldNodes := make([]ipld.Node, len(protoNodes))
    for i, protoNode := range protoNodes {
        ipldNodes[i] = protoNode.Copy()
    }
    return ipldNodes
}

func bitswapPutFile(ctx context.Context, exchange *bitswap.Bitswap, bstore *blockstore.Blockstore, inputFile string) (cid.Cid, error) {
    if comboService == nil {
        blockService := blockservice.New(*bstore, exchange)
        dagService := dag.NewDAGService(blockService)
        comboService = &dag.ComboService{ Read: dagService, Write: dagService }
    }
    //Open input file for reading
    file, err := os.OpenFile(inputFile, os.O_RDONLY, 0644)
    if err != nil {
        log.Printf("Error opening file: %v. %v\n", inputFile, err)
        return cid.Cid{}, internalError
    }
    defer file.Close()

    log.Printf("Constructing Merkle DAG from file: %v\n", inputFile)

    //Construct Merkle DAG from bottom up
    buffer := make([]byte, chunkSize)
    bytesRead := 0

    //Initialize a buffered DAG
    bufferedDAG := ipld.NewBufferedDAG(ctx, comboService)

    //List of nodes in current layer
    currNodes := make([]ipld.Node, dagMaxChildren)
    currNodeCount := 0

    //List of nodes in next layer(can grow bigger than dagMaxChildren)
    var nextNodes []dag.ProtoNode

    layerIdx := 0
    totalBytes := 0

    //Read the file chunk by chunk and create parent nodes when neccessary
    for layerIdx == 0 {
        //Fill up this current chunk buffer
        for bytesRead < chunkSize {
            tempBuffer := make([]byte, chunkSize - bytesRead)
            n, err := file.Read(tempBuffer)
            if err != nil && err != io.EOF {
                log.Printf("Error reading file: %v. %v\n", inputFile, err)
                return cid.Cid{}, internalError
            } else {
                if n == 0 && err == io.EOF {
                    break
                }
            }
            for i := 0; i < n; i ++ {
                buffer[bytesRead + i] = tempBuffer[i]
            }
            bytesRead += n
        }
        //Check if we've reach the end of the file
        if bytesRead != chunkSize {
            //Check if a partial node/block should be created
            if bytesRead != 0 {
                currNodes[currNodeCount] = dag.NodeWithData(buffer[:bytesRead]).Copy()
                currNodeCount ++
            }
            layerIdx = 1
        } else {
            currNodes[currNodeCount] = dag.NodeWithData(buffer).Copy()
            currNodeCount ++
        }
        totalBytes += bytesRead
        bytesRead = 0

        //Construct a parent node and push it to the next layer
        if currNodeCount != 0 && (currNodeCount == dagMaxChildren || layerIdx != 0) {
            //Do not create a parent if the file has only one chunk
            if len(nextNodes) == 0 && currNodeCount == 1 {
                currNodes = currNodes[:1]
                break
            } else {
                nextNodes = append(nextNodes, *dag.NodeWithData([]byte{}))
                for i := 0; i < currNodeCount; i ++ {
                    nextNodes[len(nextNodes) - 1].AddNodeLink(fmt.Sprintf("0-%d-%d", len(nextNodes) - 1, i), currNodes[i])
                }
                err = bufferedDAG.AddMany(ctx, currNodes[:currNodeCount])
                if err != nil {
                    log.Printf("Failed to add nodes to DAG. %v\n", err)
                    return cid.Cid{}, internalError
                }
                err = bufferedDAG.Commit()
                if err != nil {
                    log.Printf("Failed to commit nodes to DAG. %v\n", err)
                    return cid.Cid{}, internalError
                }
                currNodeCount = 0;
            }
        }
    }

    //Continue building the Merkle DAG
    if len(nextNodes) != 0 {
        currNodes = protoNodesToIPLDNodes(nextNodes)
    }
    for len(currNodes) != 1 {
        nextNodes = make([]dag.ProtoNode, (len(currNodes) + dagMaxChildren - 1) / dagMaxChildren)
        currNodesIdx := 0
        for i := 0; i < len(nextNodes); i ++ {
            nextNodes[i] = *dag.NodeWithData([]byte{})
            for j := 0; j < dagMaxChildren; j ++ {
                if currNodesIdx == len(currNodes) {
                    break
                }
                nextNodes[i].AddNodeLink(fmt.Sprintf("%d-%d-%d", layerIdx, i, j), currNodes[currNodesIdx])
                currNodesIdx ++
            }
        }
        err = bufferedDAG.AddMany(ctx, currNodes)
        if err != nil {
            log.Printf("Failed to add nodes to DAG. %v\n", err)
            return cid.Cid{}, internalError
        }
        err = bufferedDAG.Commit()
        if err != nil {
            log.Printf("Failed to commit nodes to DAG. %v\n", err)
            return cid.Cid{}, internalError
        }
        currNodes = protoNodesToIPLDNodes(nextNodes)
        layerIdx ++
    }

    //Add final root node
    err = bufferedDAG.AddMany(ctx, currNodes)
    if err != nil {
        log.Printf("Failed to add root node to DAG. %v\n", err)
        return cid.Cid{}, internalError
    }
    err = bufferedDAG.Commit()
    if err != nil {
        log.Printf("Failed to commit nodes to DAG. %v\n", err)
        return cid.Cid{}, internalError
    }

    log.Printf("Successfully constructed and pushed Merkle DAG with root CID %v. Total bytes: %v\n", currNodes[0].Cid(), totalBytes)

    return currNodes[0].Cid(), nil
}

func fileShareFindProviders(ctx context.Context, kadDHT *dht.IpfsDHT, requestCid string) ([]peer.AddrInfo, error) {
    cid, err := cid.Decode(requestCid)
    if err != nil {
        log.Printf("Failed to decode cid %v. %v", requestCid, err)
        return []peer.AddrInfo{}, invalidParams
    }
    peerChan := kadDHT.FindProvidersAsync(ctx, cid, 10)
    var providers []peer.AddrInfo
    for p := range peerChan {
	    log.Printf("Found provider: %s\n", p.ID)
        providers = append(providers, p)
		// You can connect to the peer and request the block using Bitswap
	}

    return providers, nil
}
