package api

import (
    "context"
    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/multiformats/go-multiaddr"
    "github.com/libp2p/go-libp2p/core/crypto"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    record "github.com/libp2p/go-libp2p-record"
)

type CustomValidator struct{}

func (v *CustomValidator) Validate(key string, value []byte) error {
	return nil
}

func (v *CustomValidator) Select(key string, values [][]byte) (int, error) {
	return 0, nil
}

const relayNodeAddr = "/ip4/130.245.173.221/tcp/4001/p2p/12D3KooWDpJ7As7BWAwRMfu1VU2WCqNjvq387JEYKDBj4kx6nXTN"
const bootstrapNodeAddr = "/ip4/130.245.173.222/tcp/61000/p2p/12D3KooWQd1K1k8XA9xVEzSAu7HUCodC7LJB6uW5Kw4VwkRdstPE"

func createLibp2pHost(privKey *crypto.PrivKey) (host.Host, error) {
	customAddr, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
	relayInfo, err := peer.AddrInfoFromString(relayNodeAddr) // converts multiaddr string to peer.addrInfo
	node, err := libp2p.New(
				libp2p.ListenAddrs(customAddr),
                libp2p.Identity(*privKey),
                libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{*relayInfo}),
				libp2p.EnableRelayService(),
	)
	return node, err
}

func setupDHT(ctx context.Context, h host.Host) (*dht.IpfsDHT, error) {
	// Set up the DHT instance
	kadDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeClient))
	if err != nil {
        return nil, err
	}

	// Bootstrap the DHT (connect to other peers to join the DHT network)
	err = kadDHT.Bootstrap(ctx)
	if err != nil {
        return nil, err
	}

    // Configure the DHT to use the custom validator
    kadDHT.Validator = record.NamespacedValidator{
	    "orcanet": &CustomValidator{}, // Add a custom validator for the "orcanet" namespace
    }
	return kadDHT, err
}
