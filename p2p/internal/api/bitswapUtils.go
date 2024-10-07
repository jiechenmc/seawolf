package api

import (
    "os"
    "io"
    "fmt"
    "log"
    "context"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/ipfs/boxo/bitswap"
    "github.com/ipfs/boxo/bitswap/network"
    "github.com/ipfs/boxo/blockservice"
    "github.com/ipfs/go-datastore"
    pb "github.com/ipfs/boxo/ipld/merkledag/pb"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    ipld "github.com/ipfs/go-ipld-format"
    dag "github.com/ipfs/boxo/ipld/merkledag"
    cid "github.com/ipfs/go-cid"
    blockstore "github.com/ipfs/boxo/blockstore"
)

var chunkSize = 256 * 1024
var dagMaxChildren = 10

var comboService *dag.ComboService = nil

func bitswapCreate(ctx context.Context, node host.Host, kadDHT *dht.IpfsDHT) (*bitswap.Bitswap, *blockstore.Blockstore) {
    //Create datastore
    ds := datastore.NewMapDatastore()

    //Create a blockstore
    bstore := blockstore.NewBlockstore(ds)

    //Create a bitswap network
    bsNetwork := network.NewFromIpfsHost(node, kadDHT)

    //Create and return bitswap instance
    exchange := bitswap.New(ctx, bsNetwork, bstore)

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

    //Keep a stack of Cid()s to 'visit'
    stack := make([]cid.Cid, 1028)
    stack[0] = root
    stackPointer := 1

    pbNode := pb.PBNode{}
    //Iterate the Merkle DAG in depth first search fashion
    //TODO: fetch all child nodes at once instead of one by one
    for stackPointer != 0 {
        stackPointer --
        node, err := comboService.Get(ctx, stack[stackPointer])
        if err != nil {
            log.Printf("Failed to fetch node in Merkle DAG. %v\n", err)
            file.Close()
            return internalError
        }
        links := node.Links()
        if len(links) == 0 {
            err = pbNode.Unmarshal(node.RawData())
            if err != nil {
                log.Printf("Failed to unmarshal raw data. %v", err)
                return internalError
            }
            //We've reached a data node/block
            _, err = file.Write(pbNode.Data)
            if err != nil {
                log.Printf("Error writing to file: %v. %v\n", tmpOutputFile, err)
                file.Close()
                return internalError
            }
        } else {
            //Push the link Cid()s onto the stack in reverse order
            for i := len(links) - 1; i >= 0; i -- {
                stack[stackPointer] = links[i].Cid
                stackPointer ++
            }
        }
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
                    nextNodes[len(nextNodes) - 1].AddNodeLink(fmt.Sprintf("child-0-%d-%d", len(nextNodes) - 1, i), currNodes[i])
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
                nextNodes[i].AddNodeLink(fmt.Sprintf("child-%d-%d-%d", layerIdx, i, j), currNodes[currNodesIdx])
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
