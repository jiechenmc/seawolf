package api

import (
    "time"
    "log"
    "errors"
    "context"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/core/routing"
    dht "github.com/libp2p/go-libp2p-kad-dht"

	// "path/filepath"
	// "github.com/btcsuite/btcd/btcutil"
	// "github.com/btcsuite/btcd/rpcclient"
)

type P2PService struct {
    username *string
    p2pHost *host.Host
    kadDHT *dht.IpfsDHT
    fsNode *FileShareNode
    chatNode *ChatNode
    walletAddress string
}

func (s *P2PService) ConnectToPeer(peerID string) (string, error) {
    if s.p2pHost == nil || s.username == nil {
        return "", notLoggedIn
    }

    err := p2pConnectToPeerID(context.Background(), *s.p2pHost, s.kadDHT, peerID)
    if err != nil {
        return "", err
    }
    return "success", nil
}

func (s *P2PService) FindPeer(peerID string) (PeerStatus, error) {
    if s.p2pHost == nil || s.username == nil {
        return PeerStatus{}, notLoggedIn
    }
    if peerID == (*s.p2pHost).ID().String() {
        return PeerStatus{ (*s.p2pHost).ID(), (*s.p2pHost).Addrs(), true }, nil
    }
    peers := p2pGetPeers(*s.p2pHost)
    //Check if peer is in local peerstore
    for _, peerStatus := range peers {
        if peerStatus.PeerID.String() == peerID {
            return peerStatus, nil
        }
    }

    //Find peer
    peer, err := p2pFindPeer(context.Background(), *s.p2pHost, s.kadDHT, peerID)
    if err != nil {
        return PeerStatus{}, err
    }
    return peer, nil
}

func (s *P2PService) GetPeers() ([]PeerStatus, error) {
    if s.p2pHost == nil || s.username == nil {
        return nil, notLoggedIn
    }
    p2pPrintConnectedPeers(*s.p2pHost)
    p2pPrintRoutingTable(s.kadDHT)
    p2pPrintKnownPeers(*s.p2pHost)

    peers := p2pGetPeers(*s.p2pHost)

    // return &PeerStatus{ peers[0].peerID, peers[0].addresses, peers[0].isConnected }, nil
    return peers, nil
}

func (s *P2PService) GetValue(key string) (string, error) {
    if s.username == nil || s.kadDHT == nil {
        return "", notLoggedIn
    }
    scopedKey := "/orcanet/" + key
    value, err := s.kadDHT.GetValue(context.Background(), scopedKey)
    if err != nil {
        log.Printf("Failed to get value for key %v. %v", scopedKey, err)
        if err == routing.ErrNotFound {
            return "", keyNotFound
        }
        return "", internalError
    }
    return string(value), nil
}

func (s *P2PService) PutValue(key string, value string) (string, error) {
    if s.username == nil || s.kadDHT == nil {
        return "", notLoggedIn
    }
    scopedKey := "/orcanet/" + key
    err := s.kadDHT.PutValue(context.Background(), scopedKey, []byte(value))
    if err != nil {
        log.Printf("Failed to put value for key %v. %v", scopedKey, err)
        if err == routing.ErrNotFound {
            return "", keyNotFound
        }
        return "", internalError
    }
    return "success", nil
}

func (s *P2PService) Login( username string, password string) (string, error) {
    if s.p2pHost != nil || s.username != nil {
        return "", alreadyLoggedIn
    }
    db, err := dbOpen()
    if err != nil {
        return "", err
    }
    defer db.Close()

    var passwordHash []byte
    var privateKeyCiphertext []byte
    var privateKeyIV []byte
    var privateKeySalt []byte
    var walletAddress string

    //Get user info from database
    count, err := dbGetUser(db, username, &passwordHash, &privateKeyCiphertext, &privateKeyIV, &privateKeySalt, &walletAddress)
    if err != nil {
        return "", err
    }
    if count == 0 {
        log.Printf("Attempted login from unregistered user '%v'\n", username)
        return "", invalidCredentials
    }

    passwordBytes := []byte(password)
    if !cipherCompareHashAndPassword(passwordHash, passwordBytes) {
        log.Printf("Attempted login to user '%v' failed\n", username)
        return "", invalidCredentials
    }

    privateKey, err := cipherDecryptPrivateKey(passwordBytes, privateKeyCiphertext, privateKeyIV, privateKeySalt)

    ctx := context.Background()
    //Create libp2p host with private key
    newHost, err := p2pCreateHost(ctx, &privateKey)
    if err != nil {
        return "", err
    }
    s.p2pHost = &newHost
    log.Printf("Successfully created libp2p host with peer ID: %v\n", (*s.p2pHost).ID())

    //Connect to at least one bootstrap node
    connSuccess := false
    for _, bootstrapNodeAddr := range bootstrapNodeAddrs {
        err = p2pConnectToPeer(ctx, *s.p2pHost, bootstrapNodeAddr)
        if err == nil {
            connSuccess = true
        }
    }
    if !connSuccess {
        //Delete libp2p host
        p2pDeleteHost(*s.p2pHost)
        s.p2pHost = nil
        return "", err
    }

    s.kadDHT, err = p2pCreateDHT(ctx, *s.p2pHost)
    if err != nil {
        //Delete libp2p host
        p2pDeleteHost(*s.p2pHost)
        s.p2pHost = nil
        return "", err
    }
    log.Printf("Successfully created DHT instance\n")

    s.username = &username
    log.Printf("Successfully logged in user '%v'\n", *s.username)

    p2pSetupStreamHandlers(*s.p2pHost, s.kadDHT);

    s.fsNode = FileShareNodeCreate(*s.p2pHost, s.kadDHT, walletAddress)
    s.chatNode = ChatNodeCreate(*s.p2pHost, s.kadDHT, s.fsNode)

    return (*s.p2pHost).ID().String(), nil
}

func (s *P2PService) Logout() (string, error) {
    if s.username == nil {
        return "", notLoggedIn
    }
    (*s.p2pHost).Close()
    s.username = nil;
    s.fsNode = nil;
    s.p2pHost = nil;
    s.chatNode = nil;
    return "success", nil
}

func (s *P2PService) Register(username string, password string, seed string) (string, error) {
    //Optional seed parameter for private key generation
    var seedBytes []byte = nil
    if seed != "" {
        seedBytes = []byte(seed)
    }

    db, err := dbOpen()
    if err != nil {
        return "", err
    }
    defer db.Close()

    //Query for username(return error if username already exists)
    count, err := dbGetUser(db, username, nil, nil, nil, nil, nil)
    if count == 1 {
        return "", errors.New("Username already exists")
    }

    passwordBytes := []byte(password)
    //Hash password
    passwordHash, err := cipherEncryptPassword(passwordBytes)
    if err != nil {
        return "", err
    }

    //Username does not exist. Generate a key pair for libp2p.
    var privateKeyCiphertext []byte
    var privateKeyIV []byte
    var privateKeySalt []byte
    privateKeyCiphertext, err = cipherGenerateEncryptedPrivateKey(passwordBytes, seedBytes, &privateKeyIV, &privateKeySalt)
    if err != nil {
        return "", err
    }

    err = dbAddUser(db, username, passwordHash, privateKeyCiphertext, privateKeyIV, privateKeySalt)
    if err != nil {
        return "", err
    }

    log.Printf("Successfully registered user '%v'\n", username)
    return "success", nil
}

func (s *P2PService) PutFile(inputFile string, price float64) (string, error) {
    if s.username == nil || s.fsNode == nil {
        log.Printf("Attempted to put file when not logged in\n")
        return "", notLoggedIn
    }
    cid, err := s.fsNode.PutFile(context.Background(), inputFile, price)
    if err != nil {
        return "", err
    }

    return cid.String(), nil
}

func (s *P2PService) GetFile(providerID string, cid string, outputFile string) (int, error) {
    if s.username == nil || s.fsNode == nil {
        log.Printf("Attempted to put file when not logged in\n")
        return -1, notLoggedIn
    }
    // err := bitswapGetFile(context.Background(), s.exchange, s.bstore, cid, outputFile)
    sessionID, err := s.fsNode.GetFile(context.Background(), providerID, cid, outputFile)
    if err != nil {
        return -1, err
    }
    return sessionID, nil
}

func (s *P2PService) DeleteFile(cid string) error {
    if s.username == nil || s.fsNode == nil {
        log.Printf("Attempted to delete file when not logged in\n")
        return notLoggedIn
    }
    return s.fsNode.DeleteFile(cid)
}

func (s *P2PService) GetUploads() ([]FileShareFile, error) {
    if s.username == nil || s.fsNode == nil {
        log.Printf("Attempted to get uploads when not logged in\n")
        return nil, notLoggedIn
    }
    return s.fsNode.GetUploads()
}

func (s *P2PService) GetDownloads() ([]FileShareFile, error) {
    if s.username == nil || s.fsNode == nil {
        log.Printf("Attempted to get downloads when not logged in\n")
        return nil, notLoggedIn
    }
    return s.fsNode.GetDownloads()
}

func (s *P2PService) Pause(sessionID int) error {
    if s.username == nil || s.fsNode == nil {
        log.Printf("Attempted to pause session when not logged in\n")
        return notLoggedIn
    }
    err := s.fsNode.PauseSession(sessionID)
    if err != nil {
        return err
    }
    return nil
}

func (s *P2PService) Resume(sessionID int) error {
    if s.username == nil || s.fsNode == nil {
        log.Printf("Attempted to resume session when not logged in\n")
        return notLoggedIn
    }
    err := s.fsNode.ResumeSession(sessionID)
    if err != nil {
        return err
    }
    return nil
}

func (s *P2PService) GetSession(sessionID int) (*FileShareSession, error) {
    if s.username == nil || s.fsNode == nil {
        log.Printf("Attempted to get session when not logged in\n")
        return nil, notLoggedIn
    }
    session, err := s.fsNode.GetSession(sessionID)
    if err != nil {
        return nil, err
    }
    return session, nil
}

func (s *P2PService) FindProviders(cid string) ([]peer.AddrInfo, error) {
    if s.username == nil || s.fsNode == nil {
        log.Printf("Attempted to find providers when not logged in\n")
        return nil, notLoggedIn
    }
    ctx := context.Background()
    // Query the DHT to find the providers of the block
    ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*2)
    defer cancel()

    providers, err := fileShareFindProviders(ctxTimeout, s.kadDHT, cid)
    if err != nil {
        return nil, err
    }

    return providers, nil

}

func (s *P2PService) DiscoverFiles() ([]FileShareFileDiscoveryInfo, error) {
    if s.username == nil || s.fsNode == nil {
        log.Printf("Attempted to discover files when not logged in\n")
        return nil, notLoggedIn
    }
    return s.fsNode.Discover(context.Background()), nil
}

func (s *P2PService) DiscoverFile(reqCid string) (*FileShareFileDiscoveryInfo, error) {
    if s.username == nil || s.fsNode == nil {
        log.Printf("Attempted to discover file when not logged in\n")
        return nil, notLoggedIn
    }
    return s.fsNode.GetFileDiscoveryInfo(context.Background(), reqCid)
}

func (s *P2PService) GetChat(peerID string, chatID int) (*ChatRoom, error) {
    if s.username == nil || s.chatNode == nil {
        log.Printf("Attempted to get chat when not logged in\n")
        return nil, notLoggedIn
    }
    return s.chatNode.GetChat(peerID, chatID)
}

func (s *P2PService) GetChats() ([]*ChatRoom, error) {
    if s.username == nil || s.chatNode == nil {
        log.Printf("Attempted to get chats when not logged in\n")
        return nil, notLoggedIn
    }
    return s.chatNode.GetChats(), nil
}

func (s *P2PService) GetMessages(peerID string, chatID int) ([]Message, error) {
    if s.username == nil || s.chatNode == nil {
        log.Printf("Attempted to get messages when not logged in\n")
        return nil, notLoggedIn
    }
    return s.chatNode.GetMessages(peerID, chatID)
}

func (s *P2PService) SendMessage(peerID string, chatID int, text string) (*Message, error) {
    if s.username == nil || s.chatNode == nil {
        log.Printf("Attempted to send message when not logged in\n")
        return nil, notLoggedIn
    }
    return s.chatNode.SendMessage(peerID, chatID, text)
}

func (s *P2PService) GetIncomingChatRequests() ([]*ChatRequest, error) {
    if s.username == nil || s.chatNode == nil {
        log.Printf("Attempted to get incoming chat requests when not logged in\n")
        return nil, notLoggedIn
    }
    return s.chatNode.GetIncomingRequests(), nil
}

func (s *P2PService) GetOutgoingChatRequests() ([]*ChatRequest, error) {
    if s.username == nil || s.chatNode == nil {
        log.Printf("Attempted to get outgoing chat requests when not logged in\n")
        return nil, notLoggedIn
    }
    return s.chatNode.GetOutgoingRequests(), nil
}

func (s *P2PService) SendChatRequest(peerID string, fileCid string) (*ChatRequest, error) {
    if s.username == nil || s.chatNode == nil {
        log.Printf("Attempted to get send chat request when not logged in\n")
        return nil, notLoggedIn
    }
    return s.chatNode.SendRequest(context.Background(), peerID, fileCid)
}

func (s *P2PService) AcceptChatRequest(peerID string, requestID int) (*ChatRoom, error) {
    if s.username == nil || s.chatNode == nil {
        log.Printf("Attempted to get accept chat request when not logged in\n")
        return nil, notLoggedIn
    }
    return s.chatNode.AcceptRequest(peerID, requestID)
}

func (s *P2PService) DeclineChatRequest(peerID string, requestID int) error {
    if s.username == nil || s.chatNode == nil {
        log.Printf("Attempted to get accept chat request when not logged in\n")
        return notLoggedIn
    }
    return s.chatNode.DeclineRequest(peerID, requestID)
}


func (s *P2PService) CloseChat(peerID string, chatID int) (*ChatRoom, error) {
    if s.username == nil || s.chatNode == nil {
        log.Printf("Attempted to get accept chat request when not logged in\n")
        return nil, notLoggedIn
    }
    return s.chatNode.CloseChat(peerID, chatID)
}

func (s *P2PService) SetWalletAddress(walletAddress string) error {
    if s.username == nil || s.chatNode == nil {
        log.Printf("Attempted to set wallet address when not logged in\n")
        return notLoggedIn
    }
    err := dbSetWalletAddress(nil, *s.username, walletAddress)
    if err != nil {
        return err
    }
    s.fsNode.SetWalletAddress(walletAddress)
    return nil
}
