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

var chunkSize = 256 * 1024
var dagMaxChildren = 10
var comboService *dag.ComboService = nil

type Provider struct {
    PeerID peer.ID          `json:"peer_id"`
    PricePerByte float64    `json:"price_per_byte"`
}

type FileShareNode struct {
    Host host.Host
    DHT *dht.IpfsDHT
    bstore blockstore.Blockstore
}

type FileShareSession struct {
    Node *FileShareNode
    StreamMap map[peer.ID]P2PStream
    BytesMap map[peer.ID]int
    HavesMap map[cid.Cid][]peer.ID
    streamLock sync.Mutex
    havesLock sync.Mutex
    sessionContext context.Context
}

type RootBlock struct {
    Size uint64             `json:"size"`
    BytePrice float64       `json:"price_per_byte"`
    Owner peer.ID           `json:"owner_peer_id"`
    Name string             `json:"name"`
}

func (r *RootBlock) Marshal() ([]byte, error) {
    ownerBytes, err := r.Owner.MarshalBinary()
    if err != nil {
        log.Printf("Attempted to marshal invalid peer ID %v. %v\n", r.Owner, err)
        return nil, internalError
    }
    var ownerBytesLen uint8
    var nameByteLen uint8
    if len(ownerBytes) > 255 {
        return nil, invalidParams
    } else {
        ownerBytesLen = uint8(len(ownerBytes))
    }
    if len(r.Name) > 255 {
        return nil, invalidParams
    } else {
        nameByteLen = uint8(len(r.Name))
    }
    bytes := make([]byte, 0, 8 + 8 + 1 + 1 + len(ownerBytes) + len(r.Name))
    bytes, _ = binary.Append(bytes, binary.BigEndian, r.Size)
    bytes, _ = binary.Append(bytes, binary.BigEndian, r.BytePrice)
    bytes, _ = binary.Append(bytes, binary.BigEndian, ownerBytesLen)
    bytes, _ = binary.Append(bytes, binary.BigEndian, nameByteLen)
    bytes, _ = binary.Append(bytes, binary.BigEndian, ownerBytes)
    bytes, _ = binary.Append(bytes, binary.BigEndian, []byte(r.Name))
    return bytes, nil
}

func (r *RootBlock) Unmarshal(bytes []byte) error {
    var ownerBytesLen uint8
    var nameByteLen uint8
    var ownerBytes []byte

    buf := libbytes.NewReader(bytes)
    err := binary.Read(buf, binary.BigEndian, &r.Size)
    if err != nil {
        return invalidParams
    }
    err = binary.Read(buf, binary.BigEndian, &r.BytePrice)
    if err != nil {
        return invalidParams
    }
    err = binary.Read(buf, binary.BigEndian, &ownerBytesLen)
    if err != nil {
        return invalidParams
    }
    err = binary.Read(buf, binary.BigEndian, &nameByteLen)
    if err != nil {
        return invalidParams
    }
    if ownerBytesLen == 0 || nameByteLen == 0 || len(bytes) < 8 + 8 + 1 + 1 + int(ownerBytesLen) + int(nameByteLen) {
        return invalidParams
    }
    ownerBytes = bytes[18:18 + ownerBytesLen]
    err = r.Owner.UnmarshalBinary(ownerBytes)
    if err != nil {
        return invalidParams
    }
    r.Name = string(bytes[18 + ownerBytesLen:18 + ownerBytesLen + nameByteLen])
    return nil
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
                    err = handleWantHave(context.Background(), stream, fsNode)
                    if err != nil {
                        return
                    }
                case "WANT\n":
                    err = handleWant(context.Background(), stream, fsNode)
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

func handleWantHave(ctx context.Context, stream P2PStream, fsNode *FileShareNode) error {
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
        has, err := fsNode.bstore.Has(context.Background(), cid)
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

func handleWant(ctx context.Context, stream P2PStream, fsNode *FileShareNode) error {
    cidStr, err := stream.ReadString('\n', fileShareWantTimeout)
    if err != nil {
        return err
    }
    cid, err := cid.Decode(cidStr[:len(cidStr) - 1])
    if err != nil {
        return err
    }
    //Query local blockstore for cid
    has, err := fsNode.bstore.Has(context.Background(), cid)
    if err == nil && has {
        block, err := fsNode.bstore.Get(context.Background(), cid)
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

func fileShareSessionCreate(ctx context.Context, node *FileShareNode) *FileShareSession {
    return &FileShareSession {
        Node: node,
        StreamMap: make(map[peer.ID]P2PStream),
        BytesMap: make(map[peer.ID]int),
        HavesMap: make(map[cid.Cid][]peer.ID),
        streamLock: sync.Mutex{},
        havesLock: sync.Mutex{},
        sessionContext: ctx,
    }
}

func (s *FileShareSession) GetStream(peerID peer.ID) (P2PStream, error) {
    s.streamLock.Lock()
    stream, ok := s.StreamMap[peerID]
    s.streamLock.Unlock()
    if !ok {
        newStream, err := p2pOpenStream(s.sessionContext, fileShareProtocol, s.Node.Host, peerID.String())
        if err == nil {
            s.streamLock.Lock()
            //If stream was created while we were attempting to create a new one, discard new stream
            stream, ok := s.StreamMap[peerID]
            if !ok {
                s.StreamMap[peerID] = newStream
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
    stream, ok := s.StreamMap[peerID]
    if ok {
        stream.Close()
        delete(s.StreamMap, peerID)
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

func (s *FileShareSession) CreateKnownPeerStreams() {
    p2pHost := s.Node.Host
    peerIDs := p2pHost.Peerstore().Peers()
    wg := sync.WaitGroup{}
    wg.Add(len(peerIDs))
    //Iterate through node's known peers and get stream
    for _, peerID := range peerIDs {
        go func(peerID peer.ID) {
            if peerID != p2pHost.ID() {
                //Don't care about errors
                s.GetStream(peerID)
            }
            wg.Done()
        }(peerID)
    }
    wg.Wait()
}

func (s *FileShareSession) GetCids(reqCids []cid.Cid) ([][]byte, error) {
    //Initialize streams with peers we know
    s.CreateKnownPeerStreams()
    wg := sync.WaitGroup{}
    wg.Add(len(s.StreamMap))
    for peerID, stream := range s.StreamMap {
        go func(peerID peer.ID, stream P2PStream) {
            peerCids := s.SendWantHave(peerID, reqCids)
            if peerCids != nil {
                //Sanity/safety check to make sure the cid sent by peer is actually in reqCids
                validCids := make([]cid.Cid, 0, len(peerCids))
                for _, c := range peerCids {
                    for _, reqCid := range reqCids {
                        if reqCid == c {
                            validCids = append(validCids, c)
                        }
                    }
                }
                s.havesLock.Lock()
                for _, c := range peerCids {
                    peerIDs, ok := s.HavesMap[c]
                    if !ok {
                        peerIDs = []peer.ID{}
                        s.HavesMap[c] = peerIDs
                    }
                    s.HavesMap[c] = append(peerIDs, peerID)
                }
                s.havesLock.Unlock()
            }
            wg.Done()
        }(peerID, stream)
    }
    //Wait until we've populate HAVEs from direct peers
    wg.Wait()
    results := make([][]byte, len(reqCids))
    wg.Add(len(reqCids))
    //Now that we have the 'haves' list, we send wants in addition to querys the DHT for providers for missing CIDs
    for i, reqCid := range reqCids {
        go func(i int, reqCid cid.Cid) {
            defer wg.Done()
            peerIDs, ok := s.HavesMap[reqCid]
            if !ok {
                ctxTimeout, cancel := context.WithTimeout(s.sessionContext, time.Second * fileShareFindProvidersTimeout)
                peerAddrInfos, err := s.Node.DHT.FindProviders(ctxTimeout, reqCid)
                cancel()
                if err != nil {
                    results[i] = nil
                    return
                }
                peerIDs = make([]peer.ID, len(peerAddrInfos))
                for j, addrInfo := range peerAddrInfos {
                    peerIDs[j] = addrInfo.ID
                }
            }
            //Send WANTs to peers
            for _, peerID := range peerIDs {
                data := s.SendWant(peerID, reqCid)
                if data != nil {
                    //Successfully obtained data
                    results[i] = data
                    return
                }
                results[i] = nil
            }
        }(i, reqCid)
    }
    wg.Wait()
    return results, nil
}

func (f *FileShareNode) GetFile(ctx context.Context, rootCid string, outputFile string) error {
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

    //Create a fileshare session
    session := fileShareSessionCreate(ctx, f)

    var bytes [][]byte

    //Check local blockstore before asking peers
    has, err := f.bstore.Has(ctx, root)
    if err != nil {
        return internalError
    }
    if has {
        block, err := f.bstore.Get(ctx, root)
        if err != nil {
            return internalError
        }
        bytes = append(bytes, block.RawData())
    } else {
        //For simplicity, file is just one block
        bytes, err = session.GetCids([]cid.Cid{root})
        if err != nil {
            return internalError
        }
    }

    pbNode := pb.PBNode{}
    if bytes[0] == nil {
        log.Printf("Failed to get file.\n")
        return internalError
    }
    err = pbNode.Unmarshal(bytes[0])

    file.Write(pbNode.Data)
    
    file.Close()
    err = os.Rename(outputFile + ".tmp", outputFile)
    if err != nil {
        log.Printf("Failed to move temporary file to output file. %v\n")
        return internalError
    }
    return nil
}

func (f *FileShareNode) PutFile(ctx context.Context, inputFile string) (cid.Cid, error) {
    //Open input file for reading
    file, err := os.OpenFile(inputFile, os.O_RDONLY, 0644)
    if err != nil {
        log.Printf("Error opening file: %v. %v\n", inputFile, err)
        return cid.Cid{}, internalError
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

    node := dag.NodeWithData(buffer).Copy()
    f.bstore.Put(ctx, node)
    err = f.DHT.Provide(ctx, node.Cid(), false)
    if err != nil {
        log.Printf("Failed to provide cid. %v\n", err)
        return node.Cid(), nil
    }

    return node.Cid(), nil
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
