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
const fileShareWantHaveTimeout = time.Second * 1
const fileShareWantTimeout = time.Second * 5
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
    sessionID int
    node *FileShareNode
    streamMap map[peer.ID]P2PStream
    streamLock sync.Mutex
    reqCid cid.Cid
    rxBytesLock sync.Mutex
    rxBytes uint64
    pausable *Pausable
    sessionContext context.Context
    complete bool
    result int
}

type FileShareRemoteSession struct {
    remoteSessionID int
    remotePeerID peer.ID
    txBytesLock sync.Mutex
    txBytes uint64
    pausable *Pausable
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
                default:
                    return
            }
        }
    })

    return fsNode
}

func (f *FileShareNode) handleWantHave(ctx context.Context, stream P2PStream) error {
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

func (f *FileShareNode) handleWant(ctx context.Context, stream P2PStream) error {
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

func (f *FileShareNode) handleWantData(ctx context.Context, stream P2PStream) error {
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

func (f *FileShareNode) handleResume(stream P2PStream) error {
    remoteSessionIDStr, err := stream.ReadString('\n', fileShareWantTimeout)
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

func (f *FileShareNode) handlePause(stream P2PStream) error {
    remoteSessionIDStr, err := stream.ReadString('\n', fileShareWantTimeout)
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

func (f *FileShareNode) SessionCreate(ctx context.Context, reqCid cid.Cid) *FileShareSession {
    nextSessionIDLock.Lock()
    sessionID := nextSessionID
    nextSessionID++
    nextSessionIDLock.Unlock()

    fileShareSession := &FileShareSession {
        sessionID: sessionID,
        node: f,
        streamMap: make(map[peer.ID]P2PStream),
        streamLock: sync.Mutex{},
        sessionContext: ctx,
        pausable: NewPausable(),
        rxBytesLock: sync.Mutex{},
        rxBytes: uint64(0),
    }

    f.sessionStoreLock.Lock()
    f.sessionStore[sessionID] = fileShareSession
    f.sessionStoreLock.Unlock()

    return fileShareSession
}

func (f *FileShareNode) SessionCleanup(session *FileShareSession, result int) {
    session.complete = true
    session.result = result
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

func (f *FileShareNode) PauseSession(sessionID int) error {
    f.sessionStoreLock.Lock()
    session, ok := f.sessionStore[sessionID]
    if !ok {
        return sessionNotFound
    }
    f.sessionStoreLock.Unlock()

    session.pausable.Pause()
    return nil
}

func (f *FileShareNode) ResumeSession(sessionID int) error {
    f.sessionStoreLock.Lock()
    session, ok := f.sessionStore[sessionID]
    if !ok {
        return sessionNotFound
    }
    f.sessionStoreLock.Unlock()

    session.pausable.Resume()
    return nil
}

func (s *FileShareSession) GetStream(peerID peer.ID) (P2PStream, error) {
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
            return stream, nil
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
    err := s.sendString(peerID, fmt.Sprintf("WANT DATA\n%s\n%s\n", s.sessionID, c.String()))
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

                if size - byteOffset < chunkSize {
                    chunkData, err = s.read(peerID, chunkSize, fileShareWantHaveTimeout)
                } else {
                    chunkData, err = s.read(peerID, size - byteOffset, fileShareWantHaveTimeout)
                }
                if err != nil {
                    close(dataChannel)
                    return
                }
                dataChannel <- chunkData
                s.rxBytesLock.Lock()
                s.rxBytes += uint64(len(chunkData))
                log.Printf("Total rx: %v bytes", s.rxBytes)
                s.rxBytesLock.Unlock()
            }
            close(dataChannel)
        }()
        return dataChannel
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
                log.Printf("Failed to parse bytes from provider. \n")
                return -1, internalError
            }
            //Get metadata and price
            rootBlock.Unmarshal(protoNode.Data())

            log.Printf("Downloading file %v, size: %v bytes, price: %v\n", rootBlock.Name, rootBlock.Size, rootBlock.Price)

            isRoot = false
            links := protoNode.Links()
            if len(links) == 1 {
                reqCid = links[0].Cid
            } else {
                log.Printf("Unexpected links from node. \n")
                return -1, internalError
            }
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
    return session.sessionID, nil
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

func bitswapFindProviders(ctx context.Context, kadDHT *dht.IpfsDHT, requestCid string) ([]peer.AddrInfo, error) {
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
