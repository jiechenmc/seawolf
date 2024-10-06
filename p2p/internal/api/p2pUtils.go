package api

import (
    "strings"
    "context"
    "log"
    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/multiformats/go-multiaddr"
    "github.com/libp2p/go-libp2p/core/crypto"
    "github.com/libp2p/go-libp2p/core/peerstore"
    "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    record "github.com/libp2p/go-libp2p-record"
)

/* CODE FROM TA TUTORIAL */

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
const bootstrapNodeAddr = "/ip4/130.245.136.245/tcp/4001/p2p/12D3KooWBTMg3kCjcKQLaTVze2Aeks3s9ibiGMRYkVi3saDXBZeZ"

func p2pCreateHost(ctx context.Context, privKey *crypto.PrivKey) (host.Host, error) {
    customAddr, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
    relayInfo, err := peer.AddrInfoFromString(relayNodeAddr) // converts multiaddr string to peer.addrInfo
    node, err := libp2p.New(
            libp2p.ListenAddrs(customAddr),
            libp2p.Identity(*privKey),
            libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{*relayInfo}),
            libp2p.EnableRelayService(),
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

    log.Printf("Successfully reserved slot on relay node\n")
    return nil
}

func p2pCreateDHT(ctx context.Context, h host.Host) (*dht.IpfsDHT, error) {
    // Set up the DHT instance
    kadDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
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


// host.Peerstore().AddAddrs(peerAddrInfo.ID, peerAddrInfo.Addrs, peerstore.PermanentAddrTTL)
