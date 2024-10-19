package api

import (
    "fmt"
    "io"
    "bufio"
    "strings"
    "context"
    "log"
    "encoding/json"
    "time"
    // ipfslog "github.com/ipfs/go-log/v2"
    "github.com/libp2p/go-libp2p/core/network"
    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/multiformats/go-multiaddr"
    "github.com/libp2p/go-libp2p/core/crypto"
    "github.com/libp2p/go-libp2p/core/peerstore"
    "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
    "github.com/libp2p/go-libp2p/core/protocol"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    record "github.com/libp2p/go-libp2p-record"
)

/* CODE FROM TA TUTORIAL */

type PeerStatus struct {
    PeerID peer.ID              `json:"peer_id"`
    Addrs []multiaddr.Multiaddr `json:"addrs,omitempty"`
    IsConnected bool            `json:"is_connected"`
}

type CustomValidator struct{}

func (v *CustomValidator) Validate(key string, value []byte) error {
    return nil
}

func (v *CustomValidator) Select(key string, values [][]byte) (int, error) {
    return 0, nil
}

const relayNodeAddr = "/ip4/130.245.173.221/tcp/4001/p2p/12D3KooWDpJ7As7BWAwRMfu1VU2WCqNjvq387JEYKDBj4kx6nXTN"
// const bootstrapNodeAddr = "/ip4/130.245.173.221/tcp/4001/p2p/12D3KooWDpJ7As7BWAwRMfu1VU2WCqNjvq387JEYKDBj4kx6nXTN"
// const bootstrapNodeAddr = "/ip4/130.245.173.222/tcp/61000/p2p/12D3KooWQd1K1k8XA9xVEzSAu7HUCodC7LJB6uW5Kw4VwkRdstPE"
// const relayNodeAddr = "/ip4/130.245.136.245/tcp/4001/p2p/12D3KooWBTMg3kCjcKQLaTVze2Aeks3s9ibiGMRYkVi3saDXBZeZ"
// var bootstrapNodeAddrs = [3]string{"/ip4/130.245.173.222/tcp/61000/p2p/12D3KooWQd1K1k8XA9xVEzSAu7HUCodC7LJB6uW5Kw4VwkRdstPE", "/ip4/130.245.136.245/tcp/4001/p2p/12D3KooWBTMg3kCjcKQLaTVze2Aeks3s9ibiGMRYkVi3saDXBZeZ", "/ip4/130.245.136.239/tcp/4001/p2p/12D3KooWESnJyyxB3J86N9DimRqz2NHkm7EohBXXs8FRJNFZHQGo"}
var bootstrapNodeAddrs = [1]string{"/ip4/130.245.173.222/tcp/61000/p2p/12D3KooWQd1K1k8XA9xVEzSAu7HUCodC7LJB6uW5Kw4VwkRdstPE"}
// const bootstrapNodeAddr = "/ip4/130.245.173.221/tcp/4001/p2p/12D3KooWDpJ7As7BWAwRMfu1VU2WCqNjvq387JEYKDBj4kx6nXTN/p2p-circuit/p2p/12D3KooWBTMg3kCjcKQLaTVze2Aeks3s9ibiGMRYkVi3saDXBZeZ"

func p2pCreateHost(ctx context.Context, privKey *crypto.PrivKey) (host.Host, error) {
    customAddr, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
    relayInfo, err := peer.AddrInfoFromString(relayNodeAddr) // converts multiaddr string to peer.addrInfo
    node, err := libp2p.New(
            libp2p.ListenAddrs(customAddr),
            libp2p.Identity(*privKey),
            libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{*relayInfo}),
            libp2p.EnableRelayService(),
            libp2p.EnableNATService(),
            libp2p.EnableHolePunching(),
            libp2p.EnableAutoNATv2(),
    )
    if err != nil {
        log.Printf("Failed to create libp2p host. %v\n", err)
        return nil, internalError
    }

    // Connect to relay server
    err = p2pConnectToPeer(ctx, node, relayNodeAddr)
    if err != nil {
        closeErr := node.Close()
        if closeErr != nil {
            log.Panic("Failed to clean up libp2p host after relay server connection failure")
        }
        return nil, peerConnectionError
    }

    err = p2pMakeReservation(ctx, node)
    if err != nil {
        closeErr := node.Close()
        if closeErr != nil {
            log.Panic("Failed to clean up libp2p host after relay reservation failure")
        }
        return nil, err
    }

    return node, err
}

func p2pMakeReservation(ctx context.Context, node host.Host) error {
    relayInfo, err := peer.AddrInfoFromString(relayNodeAddr)
    if err != nil {
        log.Printf("Failed to create addrInfo from string representation of relay multiaddr: %v", err)
        return internalError
    }
    reservation, err := client.Reserve(ctx, node, *relayInfo)
    if err != nil {
        log.Printf("Failed to make reservation on relay: %v", err)
        return internalError
    }

    log.Printf("Reserved slot on relay: %v\n", reservation)
    return nil
}

func p2pCreateDHT(ctx context.Context, h host.Host) (*dht.IpfsDHT, error) {
    // Set up the DHT instance
    kadDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeClient))
    if err != nil {
        log.Printf("Failed to create DHT instance. %v", err)
        return nil, internalError
    }

    // Bootstrap the DHT (connect to other peers to join the DHT network)
    err = kadDHT.Bootstrap(ctx)
    if err != nil {
        log.Printf("Failed to bootstrap DHT instance. %v", err)
        return nil, internalError
    }

    // Configure the DHT to use the custom validator
    kadDHT.Validator = record.NamespacedValidator{
        "orcanet": &CustomValidator{}, // Add a custom validator for the "orcanet" namespace
    }

    return kadDHT, nil
}

func p2pConnectToPeerID(ctx context.Context, node host.Host, kadDHT *dht.IpfsDHT, peerIDStr string) error {
    peerID, err := peer.Decode(peerIDStr)
    if err != nil {
        log.Printf("Failed to decode peer ID string '%v'. %v\n", peerIDStr, err)
        return invalidParams
    }

    timeoutCtx, cancel := context.WithTimeout(ctx, time.Second * 10)
    //Attempt to find peer in own peerstore or via DHT
    peerStatus, err := p2pFindPeer(timeoutCtx, node, kadDHT, peerIDStr)
    cancel()
    if err == nil && len(peerStatus.Addrs) != 0 {
        //If we're already connected, return immediately
        if peerStatus.IsConnected {
            log.Printf("Already connected to peer: %v\n", peerIDStr)
            return nil
        }
        timeoutCtx, cancel = context.WithTimeout(ctx, time.Second * 10)
		err = node.Connect(timeoutCtx, peer.AddrInfo{ ID: peerID })
        cancel()
        //Return if we've successfully connected
        if err == nil {
            log.Printf("Successfully connected to peer: %v\n", peerIDStr)
            return nil
        }
    }
    //Fallback by trying to connect via relay
    err = p2pConnectToPeerUsingRelay(ctx, node, peerIDStr)
    return err
}

// Here peerAddr is the String format of Multiaddr of a peer
func p2pConnectToPeer(ctx context.Context, node host.Host, peerAddr string) error {
    addr, err := multiaddr.NewMultiaddr(peerAddr) // convert string to Multiaddr
    if err != nil {
        log.Printf("Failed to parse peer address: %s", err)
        return internalError
    }

    info, err := peer.AddrInfoFromP2pAddr(addr) // returns a peer.AddrInfo, containing the multiaddress and ID of the node.
    if err != nil {
        log.Printf("Failed to get AddrInfo from Multiaddr: %s", err)
        return internalError
    }

    err = node.Connect(ctx, *info)
    if err != nil {
        log.Printf("Failed to connect to peer: %s", err)
        return peerConnectionError
    }
    // after successful connection to the peer, add it to the peerstore
    // Peerstore is a local storage of the host(peer) where it stores the other peers
    node.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

    log.Println("Connected to:", info.ID)

    return nil
}

// Function to print currently connected peers
func p2pPrintConnectedPeers(host host.Host) {
    log.Println("Currently connected peers:")
    peers := host.Network().Peers()
    for _, p := range peers {
        log.Printf("Peer ID: %s\n", p)
        // Optionally print addresses
        addrs := host.Peerstore().Addrs(p)
        for _, addr := range addrs {
            log.Printf(" - Address: %s\n", addr)
        }
    }
}

// Function to print all known peers
func p2pPrintKnownPeers(host host.Host) {
    log.Println("Known peers:")
    peers := host.Peerstore().Peers()
    for _, p := range peers {
        log.Printf("Peer ID: %s\n", p)
        // Optionally print addresses
        addrs := host.Peerstore().Addrs(p)
        for _, addr := range addrs {
            log.Printf(" - Address: %s\n", addr)
        }
    }
}

func p2pPrintRoutingTable(dhtService *dht.IpfsDHT) {
    // Retrieve and print the routing table
    peers := dhtService.RoutingTable().ListPeers()
    log.Println("DHT Routing Table:")
    for _, peerID := range peers {
        log.Printf("%v\n",peerID)
    }
}

func p2pConnectToPeerUsingRelay(ctx context.Context, node host.Host, targetPeerID string) error {
    targetPeerID = strings.TrimSpace(targetPeerID)
    relayAddr, err := multiaddr.NewMultiaddr(relayNodeAddr)
    if err != nil {
        log.Printf("Failed to create relay multiaddr: %v", err)
        return internalError
    }
    peerMultiaddr := relayAddr.Encapsulate(multiaddr.StringCast("/p2p-circuit/p2p/" + targetPeerID))

    relayedAddrInfo, err := peer.AddrInfoFromP2pAddr(peerMultiaddr)
    if err != nil {
        log.Println("Failed to get relayed AddrInfo: %w", err)
        return internalError
    }

    // Connect to the peer through the relay
    err = node.Connect(ctx, *relayedAddrInfo)
    if err != nil {
        log.Println("Failed to connect to peer through relay: %w", err)
        return internalError
    }
    node.Peerstore().AddAddrs(relayedAddrInfo.ID, relayedAddrInfo.Addrs, peerstore.PermanentAddrTTL)

    log.Printf("Connected to peer via relay: %s\n", targetPeerID)
    return nil
}

func p2pDeleteHost(node host.Host) error {
    err := node.Close()
    if err != nil {
        log.Panic("Failed to clean up libp2p host after DHT creation failure")
    }
    return err
}

func p2pIsConnected(node host.Host, peerID peer.ID) bool {
    for _, p := range node.Network().Peers() {
        if p == peerID {
            return true
        }
    }
    return false
}

func p2pGetPeers(node host.Host) []PeerStatus {
    peers := node.Peerstore().Peers()
    var peerStatuses []PeerStatus
    for _, p := range peers {
        if p == node.ID() {
            continue
        }
        connected := p2pIsConnected(node, p)
        addrs := node.Peerstore().Addrs(p)
        peerStatuses = append(peerStatuses, PeerStatus{ p, addrs,  connected })
    }
    return peerStatuses
}

func p2pFindPeer(ctx context.Context, node host.Host, kadDHT *dht.IpfsDHT, peerIDStr string) (PeerStatus, error) {
    peerID, err := peer.Decode(peerIDStr)
    if err != nil {
        log.Printf("Failed to decode peer ID string '%v'. %v\n", peerIDStr, err)
        return PeerStatus{}, invalidParams
    }
    //Attempt to find peer in local peerstore
    peers := p2pGetPeers(node)
    for _, p := range peers {
        if p.PeerID == peerID {
            return p, nil
        }
    }
    //Fall back to DHT if we can't find peer in peerstore
    addrInfo, err := kadDHT.FindPeer(ctx, peerID)
    if err != nil {
        log.Printf("Failed to find peer. %v\n", err)
        return PeerStatus{}, peerNotFound
    }
    return PeerStatus{ addrInfo.ID, addrInfo.Addrs, false }, nil
}

func p2pSetupStreamHandlers(node host.Host, kadDHT *dht.IpfsDHT, messages chan string) {
    //Handler for /orcanet/p2p/seawolf/messages protocol for simple message sending
    node.SetStreamHandler("/orcanet/p2p/seawolf/messages", func(s network.Stream) {
        defer s.Close()
        buf := bufio.NewReader(s)
        message, err := buf.ReadString('\n')
        if err != nil {
            if err != io.EOF {
                log.Printf("/orcanet/p2p/seawolf/messages: Error reading from stream: %v\n", err)
                return
            }
        } else {
            message = message[:len(message) - 1] //Remove new line
        }
        log.Printf("/orcanet/p2p/seawolf/messages: Got new message: %v\n", message)
        messages <- message
    })
    //Handler for /orcanet/p2p for peer discovery
    relayInfo, _ := peer.AddrInfoFromString(relayNodeAddr)
    node.SetStreamHandler("/orcanet/p2p", func(s network.Stream) {
        defer s.Close()
        ctx := context.Background()

        buf := bufio.NewReader(s)
        peerAddr, err := buf.ReadString('\n')
        if err != nil {
            if err != io.EOF {
                log.Printf("/orcanet/p2p: error reading from stream: %v\n", err)
                return
            }
        }
        peerAddr = strings.TrimSpace(peerAddr)
        var data map[string]interface{}
        err = json.Unmarshal([]byte(peerAddr), &data)
        if err != nil {
            log.Printf("/orcanet/p2p: error unmarshaling JSON: %v\n", err)
            return
        }
        if knownPeers, ok := data["known_peers"].([]interface{}); ok {
            for _, peer := range knownPeers {
                if peerMap, ok := peer.(map[string]interface{}); ok {
                    if peerID, ok := peerMap["peer_id"].(string); ok {
                        if string(peerID) != string(relayInfo.ID) {
                            log.Printf("/orcanet/p2p: Found new peer %v\n", peerID)
                            p2pConnectToPeerID(ctx, node, kadDHT, peerID)
                        }
                    }
                }
            }
        }
    })
}

func p2pSendMessage(ctx context.Context, node host.Host, peerIDStr string, message string) error {
    peerID, err := peer.Decode(peerIDStr)
    if err != nil {
        log.Printf("Failed to decode peer ID string '%v'. %v\n", peerIDStr, err)
        return invalidParams
    }

    stream, err := node.NewStream(network.WithAllowLimitedConn(ctx, "/orcanet/p2p/seawolf/messages"), peerID, "/orcanet/p2p/seawolf/messages")
    if err != nil {
        log.Printf("Failed to open stream after multiple attempts. %v", err)
        return internalError
    }
    defer stream.Close()

    writer := bufio.NewWriter(stream)
    fmt.Fprintln(writer, message)
    err = writer.Flush()
    if err != nil {
        log.Printf("Failed to send message to stream. %v\n", err)
        return internalError
    }
    return nil
}

type P2PStream struct {
    RemotePeerID peer.ID
    NetworkStream *network.Stream
    ReadWriter *bufio.ReadWriter
}

func p2pOpenStream(ctx context.Context, protocolStr string, node host.Host, peerIDStr string) (P2PStream, error) {
    peerID, err := peer.Decode(peerIDStr)
    if err != nil {
        log.Printf("Failed to decode peer ID string '%v'. %v\n", peerIDStr, err)
        return P2PStream{}, invalidParams
    }

    stream, err := node.NewStream(network.WithAllowLimitedConn(ctx, protocolStr), peerID, protocol.ID(protocolStr))
    if err != nil {
        log.Printf("Failed to open stream after multiple attempts. %v", err)
        return P2PStream{}, internalError
    }

    reader := bufio.NewReader(stream)
    writer := bufio.NewWriter(stream)
    rw := bufio.NewReadWriter(reader, writer)

    return P2PStream{ peerID, &stream, rw }, err
}

func p2pWrapStream(stream *network.Stream) P2PStream {
    reader := bufio.NewReader(*stream)
    writer := bufio.NewWriter(*stream)
    rw := bufio.NewReadWriter(reader, writer)
    return P2PStream{ (*stream).Conn().RemotePeer(), stream, rw }
}

func (s P2PStream) Send(bytes []byte) error {
    _, err := s.ReadWriter.Write(bytes)
    if err != nil {
        goto failed
    }
    err = s.ReadWriter.Flush()
failed:
    if err != nil {
        log.Printf("%v: Failed to write to stream. %v\n", (*s.NetworkStream).Protocol(), err)
        return err
    }
    return nil
}

func (s P2PStream) SendString(str string) error {
    _, err := fmt.Fprint(s.ReadWriter, str)
    if err != nil {
        goto failed
    }
    err = s.ReadWriter.Flush()
failed:
    if err != nil {
        log.Printf("%v: Failed to write string to stream. %v\n", (*s.NetworkStream).Protocol(), err)
        return err
    }
    return nil
}

func (s P2PStream) Read(n int, timeout time.Duration) ([]byte, error) {
    if timeout != 0 {
        (*s.NetworkStream).SetReadDeadline(time.Now().Add(timeout))
    }
    var err error
    bytes := make([]byte, n)
    //Read n bytes
    for i := 0; i < n; i ++ {
        bytes[i], err = s.ReadWriter.ReadByte()
        if err != nil {
            log.Printf("%v: Failed to read from stream. %v\n", (*s.NetworkStream).Protocol(), err)
            return []byte{}, err
        }
    }
    return bytes, nil
}

func (s P2PStream) ReadString(delim byte, timeout time.Duration) (string, error) {
    if timeout != 0 {
        (*s.NetworkStream).SetReadDeadline(time.Now().Add(timeout))
    }
    str, err := s.ReadWriter.ReadString(delim)
    //Return an error even encountering EOF, the delimiter should be part of protocol
    if err != nil {
        log.Printf("%v: Failed to read string from stream. %v\n", (*s.NetworkStream).Protocol(), err)
        return "", err
    }
    return str, nil
}

func (s P2PStream) Close() {
    (*s.NetworkStream).Close()
}
