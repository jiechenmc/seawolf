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
    "github.com/ipfs/go-datastore"
    libbytes "bytes"
    dssync "github.com/ipfs/go-datastore/sync"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    dag "github.com/ipfs/boxo/ipld/merkledag"
    cid "github.com/ipfs/go-cid"
    blockstore "github.com/ipfs/boxo/blockstore"
)

const fileShareProtocol = "/orcanet/p2p/seawolf/fileshare"
const fileShareWantHaveTimeout = time.Second * 5
const fileShareWantTimeout = time.Second * 10
const fileShareFindProvidersTimeout = time.Second * 1
const fileShareOpenStreamTimeout = time.Second * 1
const fileShareIdleTimeout = time.Second * 60
const fileShareDirectory = "fileshare"
const fileShareUploadsDirectory = "fileshare/uploads"
const fileShareDownloadsDirectory = "fileshare/downloads"

var nextSessionIDLock sync.Mutex
var nextSessionID = 0
var chunkSize = 256 * 1024
var dagMaxChildren = 10
var comboService *dag.ComboService = nil

type FileShareNode struct {
    host host.Host
    kadDHT *dht.IpfsDHT
    bstore blockstore.Blockstore
    mstore map[cid.Cid]FileShareFileMeta
    sessionStore map[int]*FileShareSession
    rSessionStore map[peer.ID]map[int]*FileShareRemoteSession
    mstoreLock sync.Mutex
    sessionStoreLock sync.Mutex
    rSessionStoreLock sync.Mutex
}

type Pausable struct {
    pauseLock sync.Mutex
    Paused int              `json:"paused"`
    resumeChannel chan bool
}

type FileShareSession struct {
    SessionID int                   `json:"session_id"`
    ReqCid string                   `json:"req_cid"`
    RxBytes uint64                  `json:"rx_bytes"`
    Complete bool                   `json:"is_complete"`
    Result int                      `json:"result"`
    Pausable
    node *FileShareNode
    streamMap map[peer.ID]*P2PStream
    streamLock sync.Mutex
    reqLocks map[peer.ID]*sync.Mutex
    reqLocksLock sync.Mutex
    statsLock sync.Mutex
    sessionContext context.Context
}

type FileShareRemoteSession struct {
    remoteSessionID int
    remotePeerID peer.ID
    txBytesLock sync.Mutex
    txBytes uint64
    Pausable
}

type FileShareFileDiscoveryInfo struct {
    Size uint64                     `json:"size"`
    DataCid string                  `json:"data_cid"`
    Providers []FileShareProvider   `json:"providers"`
}

type FileShareProvider struct {
    PeerID peer.ID          `json:"peer_id"`
    Price float64           `json:"price"`
    Name string             `json:"file_name"`
}

type FileShareFileMeta struct {
    Size uint64             `json:"size"`
    Price float64           `json:"price"`
    Name string             `json:"file_name"`
}

type FileShareUploadedFile struct {
    FileShareFileMeta
    DataCid string                  `json:"data_cid"`
}

func (r *FileShareFileMeta) Marshal() ([]byte, error) {
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

func (r *FileShareFileMeta) Unmarshal(bytes []byte) error {
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
    if nameByteLen == 0 || len(bytes) != 8 + 8 + 1 + int(nameByteLen) {
        return invalidParams
    }
    r.Name = string(bytes[17:17 + nameByteLen])
    return nil
}

func NewPausable() *Pausable {
    return &Pausable{
        pauseLock: sync.Mutex{},
        Paused: 0,
        resumeChannel: make(chan bool, 0),
    }
}

func (p *Pausable) Pause() {
    p.pauseLock.Lock()
    if p.Paused == 0 {
        p.Paused = 1
    }
    p.pauseLock.Unlock()
}

func (p *Pausable) Resume() {
    p.pauseLock.Lock()
    if p.Paused != 0 {
        for ; p.Paused > 1; {
            p.resumeChannel <- true
            p.Paused --
        }
        p.Paused = 0
    }
    p.pauseLock.Unlock()
}

func (p *Pausable) Wait() {
    p.pauseLock.Lock()
    if p.Paused != 0 {
        p.Paused ++
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
        host: node,
        kadDHT: kadDHT,
        bstore: blkStore,
        mstore: make(map[cid.Cid]FileShareFileMeta),
        sessionStore: make(map[int]*FileShareSession),
        rSessionStore: make(map[peer.ID]map[int]*FileShareRemoteSession),
        mstoreLock: sync.Mutex{},
        sessionStoreLock: sync.Mutex{},
        rSessionStoreLock: sync.Mutex{},
    }

    node.SetStreamHandler(fileShareProtocol, fsNode.fileShareStreamHandler)

    // Read files database for existing uploaded files
    fileMetadataMap, err := dbGetFiles(nil, node.ID().String())
    if err == nil {
        for cidStr, fileMeta := range fileMetadataMap {
            cid, err := cid.Decode(cidStr)
            if err != nil {
                // Remove corrupted entry with invalid cid
                dbRemoveFile(nil, node.ID().String(), cidStr)
                continue
            }

            diskCid, err := fsNode.PutFile(context.Background(), fileShareUploadsDirectory + "/" + fileMeta.Name, fileMeta.Price)
            if err != nil || cid != diskCid {
                // Remove corrupted entry with invalid cid or file
                dbRemoveFile(nil, node.ID().String(), cidStr)
                continue
            }
        }
    }

    return fsNode
}

func (f *FileShareNode) fileShareStreamHandler(s network.Stream) {
    stream := p2pWrapStream(&s)
    defer stream.Close()
    for {
        req, err := stream.ReadString('\n', fileShareIdleTimeout)
        if err != nil {
            return
        }

        switch req {
            case "WANT HAVE\n":
                err = f.handleWantHave(context.Background(), stream)
                if err != nil {
                    return
                }
            case "WANT META\n":
                err = f.handleWantMeta(context.Background(), stream)
                if err != nil {
                    return
                }
            case "WANT DATA\n":
                err = f.handleWantData(context.Background(), stream)
                if err != nil {
                    return
                }
            case "PAUSE\n":
                err = f.handlePause(stream)
                if err != nil {
                    return
                }
            case "RESUME\n":
                err = f.handleResume(stream)
                if err != nil {
                    return
                }
            case "DISCOVER\n":
                err = f.handleDiscover(stream)
                if err != nil {
                    return
                }
            case "CLOSE\n":
                return
            default:
                return
        }
    }
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

//Request:  "WANT META\n<cid>\n"
//Response: "HERE\n<size>\n<byte1><byte2>..."
func (f *FileShareNode) handleWantMeta(ctx context.Context, stream *P2PStream) error {
    //Get requested CID
    cidStr, err := stream.ReadString('\n', fileShareWantTimeout)
    if err != nil {
        return err
    }
    cid, err := cid.Decode(cidStr[:len(cidStr) - 1])
    if err != nil {
        return err
    }

    //Look for metadata in meta data store
    f.mstoreLock.Lock()
    fileMetadata, ok := f.mstore[cid]
    f.mstoreLock.Unlock()
    //This is request for metadata given data cid
    if ok {
        rawData, err := fileMetadata.Marshal()
        if err != nil {
            log.Printf("Failed to marshal file metadata. %v \n", err)
            return err
        }
        err = stream.SendString(fmt.Sprintf("HERE\n%d\n", len(rawData)))
        if err != nil {
            return err
        }
        err = stream.Send(rawData)
        if err != nil {
            return err
        }
        return nil
    }
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
            rSession.Wait()

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
    } else {
        if err != nil {
            return err
        } else {
            goto Failed
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

    rSession.Resume()
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

    rSession.Pause()
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
    f.mstoreLock.Lock()
    for dataCid, _ := range f.mstore {
        knownCids = append(knownCids, dataCid)
        i ++
        if i == maxCount {
            break
        }
    }
    f.mstoreLock.Unlock()
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


func (f *FileShareNode) SessionCreate(ctx context.Context, reqCidStr string) *FileShareSession {
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
        Pausable: *NewPausable(),
        statsLock: sync.Mutex{},
        reqLocks: make(map[peer.ID]*sync.Mutex),
        reqLocksLock: sync.Mutex{},
        ReqCid: reqCidStr,
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

    session.streamLock.Lock()
    for peerID, stream := range session.streamMap {
        stream.Close()
        delete(session.streamMap, peerID)
    }
    session.streamLock.Unlock()
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
            Pausable: *NewPausable(),
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

func (f *FileShareNode) PauseSession(sessionID int) error {
    f.sessionStoreLock.Lock()
    session, ok := f.sessionStore[sessionID]
    f.sessionStoreLock.Unlock()
    if !ok {
        return sessionNotFound
    }

    session.PauseSession()
    return nil
}

func (f *FileShareNode) ResumeSession(sessionID int) error {
    f.sessionStoreLock.Lock()
    session, ok := f.sessionStore[sessionID]
    f.sessionStoreLock.Unlock()
    if !ok {
        return sessionNotFound
    }

    session.ResumeSession()
    return nil
}

func (f *FileShareNode) HasFile(fileCid cid.Cid) bool {
    has, err := f.bstore.Has(context.Background(), fileCid)
    if err != nil {
        return false
    }
    return has
}

func (s *FileShareSession) GetStream(peerID peer.ID) (*P2PStream, error) {
    s.streamLock.Lock()
    stream, ok := s.streamMap[peerID]
    s.streamLock.Unlock()
    if !ok {
        timeoutCtx, cancel := context.WithTimeout(s.sessionContext, fileShareOpenStreamTimeout)
        newStream, err := p2pOpenStream(timeoutCtx, fileShareProtocol, s.node.host, s.node.kadDHT, peerID.String())
        cancel()
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

func (s *FileShareSession) GetRequestLock(peerID peer.ID) *sync.Mutex {
    s.reqLocksLock.Lock()
    defer s.reqLocksLock.Unlock()
    _, ok := s.reqLocks[peerID]
    if !ok {
        s.reqLocks[peerID] = &sync.Mutex{}
    }
    lock := s.reqLocks[peerID]
    return lock
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
    reqLock := s.GetRequestLock(peerID)
    reqLock.Lock()
    defer reqLock.Unlock()
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

func (s *FileShareSession) SendWantMeta(peerID peer.ID, c cid.Cid) []byte {
    reqLock := s.GetRequestLock(peerID)
    reqLock.Lock()
    defer reqLock.Unlock()

    //Send WANT request
    err := s.sendString(peerID, fmt.Sprintf("WANT META\n%s\n", c.String()))
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
    reqLock := s.GetRequestLock(peerID)
    reqLock.Lock()
    defer reqLock.Unlock()
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
                s.Wait()

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

func (s *FileShareSession) PauseSession() error {
    s.Pause()
    for peerID, _ := range s.streamMap {
        timeoutCtx, cancel := context.WithTimeout(s.sessionContext, fileShareOpenStreamTimeout)
        stream, err := p2pOpenStream(timeoutCtx, fileShareProtocol, s.node.host, s.node.kadDHT, peerID.String())
        cancel()
        if err != nil {
            return err
        }
        err = stream.SendString(fmt.Sprintf("PAUSE\n%d\n", s.SessionID))
        err = stream.SendString("CLOSE\n")
        stream.Close()
    }
    return nil
}


func (s *FileShareSession) ResumeSession() error {
    s.Resume()
    for peerID, _ := range s.streamMap {
        timeoutCtx, cancel := context.WithTimeout(s.sessionContext, fileShareOpenStreamTimeout)
        stream, err := p2pOpenStream(timeoutCtx, fileShareProtocol, s.node.host, s.node.kadDHT, peerID.String())
        cancel()
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
    reqLock := s.GetRequestLock(peerID)
    reqLock.Lock()
    defer reqLock.Unlock()
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
    session := f.SessionCreate(ctx, rootCidStr)
    defer func() {
        if deferCleanup {
            file.Close()
            f.SessionCleanup(session, 1)
        }
    }()

    var bytes []byte
    var dataChannel chan []byte
    var ok bool
    reqCid := rootCid
    fileMeta := FileShareFileMeta{}
    //Check local blockstore before asking peers
    has, err := f.bstore.Has(ctx, reqCid)
    if err != nil {
        return -1, internalError
    }
    if has {
        f.mstoreLock.Lock()
        fileMeta, ok = f.mstore[reqCid]
        f.mstoreLock.Unlock()
        if !ok {
            return -1, internalError
        }
        block, err := f.bstore.Get(ctx, reqCid)
        if err != nil {
            return -1, internalError
        }
        bytes = block.RawData()
        dataChannel <- bytes
        close(dataChannel)
    } else {
        bytes = session.SendWantMeta(providerID, reqCid)
        if bytes == nil {
            log.Printf("Failed to get file metadata.\n")
            return -1, internalError
        }
        err = fileMeta.Unmarshal(bytes)
        if err != nil {
            log.Printf("Failed to unmarshal file metadata.\n")
            return -1, internalError
        }
        dataChannel = session.SendWantData(providerID, reqCid)
        if dataChannel == nil {
            log.Printf("Failed to get file.\n")
            return -1, internalError
        }
    }

    deferCleanup = false
    go func() {
        bytesWritten := uint64(0)
        for data := range dataChannel {
            file.Write(data)
            bytesWritten += uint64(len(data))
        }
        file.Close()
        if bytesWritten != fileMeta.Size {
            f.SessionCleanup(session, 1)
            log.Printf("Wrong number of bytes received\n")
            return
        }
        //TODO compute hash to verify integrity of file
        err = os.Rename(outputFile + ".tmp", outputFile)
        if err != nil {
            f.SessionCleanup(session, 1)
            log.Printf("Failed to move temporary file to output file. %v\n")
            return
        }
        f.SessionCleanup(session, 0)
    }()
    return session.SessionID, nil
}

func (f *FileShareNode) PutFile(ctx context.Context, inputFile string, price float64) (cid.Cid, error) {
    //Open input file for reading
    buffer, err := readFile(inputFile)
    if err != nil {
        return cid.Cid{}, err
    }
    bytesRead := len(buffer)
    //Create data node and link it with root node
    node := dag.NodeWithData(buffer).Copy()

    //Create metadata node
    filename := filepath.Base(inputFile)
    fileMeta := FileShareFileMeta{ Size: uint64(bytesRead), Price: price, Name: filename }

    f.bstore.Put(ctx, node)

    f.mstoreLock.Lock()
    f.mstore[node.Cid()] = fileMeta
    f.mstoreLock.Unlock()

    err = f.kadDHT.Provide(ctx, node.Cid(), true)
    if err != nil {
        log.Printf("Failed to provide cid. %v\n", err)
        return cid.Cid{}, internalError
    }
    // Record file into database
    err = dbAddFile(nil, f.host.ID().String(), node.Cid().String(), filename, price)
    if err != nil {
        log.Printf("Failed to record file into database. %v\n", err)
        return cid.Cid{}, internalError
    }

    inputFilepath, err := filepath.Abs(inputFile)
    if err != nil {
        log.Printf("Failed to resolve absolute path for input file path. %v\n", err)
        return cid.Cid{}, internalError
    }
    dstFilePath, err := filepath.Abs(fileShareUploadsDirectory + "/" + filename)
    if err != nil {
        log.Printf("Failed to resolve absolute path for upload directory path. %v\n", err)
        return cid.Cid{}, internalError
    }

    // No need to copy to uploaded files directory if input file is already there
    if inputFilepath != dstFilePath {
        // Copy file to files directory
        err = os.MkdirAll(fileShareUploadsDirectory, 0750)
        if err != nil && !os.IsExist(err) {
            log.Printf("Failed to create uploads directory. %v\n", err)
            return cid.Cid{}, internalError
        }
        err = copyFile(inputFile, fileShareUploadsDirectory + "/" + filename)
        if err != nil {
            log.Printf("Failed to copy file to upload directory. %v\n", err)
            return cid.Cid{}, internalError
        }
    }

    return node.Cid(), nil
}

func (f *FileShareNode) Discover(ctx context.Context) []FileShareFileDiscoveryInfo {
    session := f.SessionCreate(ctx, "")
    defer f.SessionCleanup(session, 0)

    mapLock := sync.Mutex{}
    fileDiscoveryMap := make(map[cid.Cid]*FileShareFileDiscoveryInfo)

    p2pHost := f.host
    peerIDs := p2pHost.Peerstore().Peers()
    wg := sync.WaitGroup{}
    wg.Add(len(peerIDs))

    //Dump own mstore into set of data cids
    mapLock.Lock()
    f.mstoreLock.Lock()
    for dataCid, _ := range f.mstore {
        fileDiscoveryMap[dataCid] = nil
    }
    f.mstoreLock.Unlock()
    mapLock.Unlock()

    //Iterate through node's known peers and send discover requests
    for _, peerID := range peerIDs {
        go func(peerID peer.ID) {
            cids := session.SendDiscover(peerID, 1000)
            if cids != nil {
                mapLock.Lock()
                for _, dataCid := range cids {
                    fileDiscoveryMap[dataCid] = nil
                }
                mapLock.Unlock()
            }
            wg.Done()
        }(peerID)
    }
    wg.Wait()

    wg.Add(len(fileDiscoveryMap))
    //Iterate through all unique data cids and get discovery entry for each
    for dataCid, _ := range fileDiscoveryMap {
        go func() {
            fileDiscovery := session.DiscoverFile(ctx, dataCid, 1000)
            //Add unique files into map and accumulate providers for each file
            mapLock.Lock()
            fileDiscoveryMap[dataCid] = fileDiscovery
            mapLock.Unlock()
            wg.Done()
       }()
    }
    wg.Wait()

    fileDiscoveries := make([]FileShareFileDiscoveryInfo, 0, len(fileDiscoveryMap))
    for _, fileDiscovery := range fileDiscoveryMap {
        //If nil, no providers were found for this file
        if fileDiscovery == nil {
            continue
        }
        fileDiscoveries = append(fileDiscoveries, *fileDiscovery)
    }

    return fileDiscoveries
}

func (f *FileShareNode) GetFileDiscoveryInfo(ctx context.Context, reqCidStr string) (*FileShareFileDiscoveryInfo, error) {
    reqCid, err := cid.Decode(reqCidStr)
    if err != nil {
        return nil, invalidParams
    }

    session := f.SessionCreate(ctx, "")
    defer f.SessionCleanup(session, 0)

    fileDiscovery := session.DiscoverFile(ctx, reqCid, 1000)
    return fileDiscovery, nil
}

func (s *FileShareSession) DiscoverFile(ctx context.Context, reqCid cid.Cid, providers int) *FileShareFileDiscoveryInfo {
    ctxTimeout, cancel := context.WithTimeout(ctx, fileShareFindProvidersTimeout)
    providerAddrs, err := s.node.kadDHT.FindProviders(ctxTimeout, reqCid)
    cancel()
    if err != nil {
        return nil
    }

    lock := sync.Mutex{}
    wg := sync.WaitGroup{}
    fileDiscovery := FileShareFileDiscoveryInfo{
        DataCid: "",
        Size: 0,
        Providers: make([]FileShareProvider, 0, providers),
    }

    wg.Add(len(providerAddrs))
    for i, provider := range providerAddrs {
        go func(i int) {
            defer wg.Done()
            //If we've obtained the target providers, ignore every subsequent entry
            if len(fileDiscovery.Providers) == providers {
                return
            }
            fileMeta := FileShareFileMeta{}
            var ok bool
            //Get metadata
            if provider.ID == s.node.host.ID() {
                s.node.mstoreLock.Lock()
                fileMeta, ok = s.node.mstore[reqCid]
                s.node.mstoreLock.Unlock()
                if !ok {
                    return
                }
            } else {
                bytes := s.SendWantMeta(provider.ID, reqCid)
                if bytes == nil {
                    return
                }
                err = fileMeta.Unmarshal(bytes)
                if err != nil {
                    log.Printf("Error unmarshalling file metadata\n")
                    return
                }
            }

            provider := FileShareProvider{
                PeerID: provider.ID,
                Price: fileMeta.Price,
                Name: fileMeta.Name,
            }

            lock.Lock()
            fileDiscovery.DataCid = reqCid.String()
            fileDiscovery.Size = fileMeta.Size
            fileDiscovery.Providers = append(fileDiscovery.Providers, provider)
            lock.Unlock()

            //Add to metadata store
            s.node.mstoreLock.Lock()
            _, ok = s.node.mstore[reqCid]
            if !ok {
                s.node.mstore[reqCid] = fileMeta
            }
            s.node.mstoreLock.Unlock()
        }(i)
    }
    wg.Wait()
    if len(fileDiscovery.Providers) == 0 {
        return nil
    }
    return &fileDiscovery
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


func (f *FileShareNode) GetUploadedFiles() ([]FileShareUploadedFile, error) {
    f.mstoreLock.Lock()
    files := make([]FileShareUploadedFile, 0, len(f.mstore))
    for cid, fileMetadata := range f.mstore {
        has, err := f.bstore.Has(context.Background(), cid)
        if err == nil && has {
            files = append(files, FileShareUploadedFile {
                fileMetadata,
                cid.String(),
            })
        }
    }
    f.mstoreLock.Unlock()
    return files, nil
}

func readFile(filePath string) ([]byte, error) {
    absFilePath, err := filepath.Abs(filePath)
    if err != nil {
        log.Printf("Failed to resolve file path to upload directory")
        return nil, failedToOpenFile
    }
    //Open input file for reading
    file, err := os.OpenFile(absFilePath, os.O_RDONLY, 0644)
    if err != nil {
        log.Printf("Error opening file: %v. %v\n", filePath, err)
        return nil, failedToOpenFile
    }
    defer file.Close()
    buffer := []byte{}

    for {
        tempBuffer := make([]byte, chunkSize)
        n, err := file.Read(tempBuffer)
        if err != nil && err != io.EOF {
            log.Printf("Error reading file: %v. %v\n", filePath, err)
            return nil, internalError
        } else {
            if n == 0 && err == io.EOF {
                break
            }
        }
        buffer = append(buffer, tempBuffer[:n]...)
    }
    return buffer, nil
}

func copyFile(srcFilePath string, dstFilePath string) error {
    srcAbsFilePath, err := filepath.Abs(srcFilePath)
    if err != nil {
        log.Printf("Failed to resolve file path to src directory")
        return failedToOpenFile
    }
    dstAbsFilePath, err := filepath.Abs(dstFilePath)
    if err != nil {
        log.Printf("copyFile: Failed to resolve file path to dst directory")
        return failedToOpenFile
    }
    if srcAbsFilePath == dstAbsFilePath {
        return nil
    }
    //Open src file for reading
    srcFile, err := os.OpenFile(srcAbsFilePath, os.O_RDONLY, 0644)
    if err != nil {
        log.Printf("Error opening file: %v. %v\n", srcFilePath, err)
        return failedToOpenFile
    }
    defer srcFile.Close()
    //Open dst file for writing
    dstFile, err := os.OpenFile(dstAbsFilePath + ".tmp", os.O_RDWR | os.O_TRUNC | os.O_CREATE, 0700)
    if err != nil {
        log.Printf("Error opening file: %v. %v\n", dstFilePath, err)
        return failedToOpenFile
    }
    defer dstFile.Close()

    tempBuffer := make([]byte, chunkSize)
    for {
        n, err := srcFile.Read(tempBuffer)
        if err != nil && err != io.EOF {
            log.Printf("Error reading file: %v. %v\n", srcFilePath, err)
            return internalError
        } else {
            if n == 0 && err == io.EOF {
                break
            }
        }
        _, err = dstFile.Write(tempBuffer[:n])
        if err != nil {
            log.Printf("Error writing to file: %v. %v\n", dstFilePath, err)
            os.Remove(dstAbsFilePath)
            return internalError
        }
    }
    return os.Rename(dstAbsFilePath + ".tmp", dstAbsFilePath)
}
