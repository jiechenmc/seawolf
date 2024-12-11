package api

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const proxyProtocol = "/orcanet/p2p/seawolf/proxy"
const proxyRequestTimeout = time.Second * 10

type ProxyStatus struct {
	PeerID  string `json:"peer_id"`
	IsProxy bool   `json:"is_proxy"`
}

type ProxyNode struct {
	host      host.Host
	kadDHT    *dht.IpfsDHT
	proxies   map[peer.ID]bool
	proxyLock sync.Mutex
}

func ProxyNodeCreate(hostNode host.Host, kadDHT *dht.IpfsDHT) *ProxyNode {
	pn := &ProxyNode{
		host:    hostNode,
		kadDHT:  kadDHT,
		proxies: make(map[peer.ID]bool),
	}
	hostNode.SetStreamHandler(proxyProtocol, pn.proxyStreamHandler)
	return pn
}

func (pn *ProxyNode) proxyStreamHandler(s network.Stream) {
	stream := p2pWrapStream(&s)
	defer stream.Close()
	req, err := stream.ReadString('\n', proxyRequestTimeout)
	if err != nil {
		return
	}

	switch req {
	case "REGISTER\n":
		err = pn.handleRegisterProxy(context.Background(), stream)
		if err != nil {
			stream.Close()
		}
	case "UNREGISTER\n":
		err = pn.handleUnregisterProxy(context.Background(), stream)
		if err != nil {
			stream.Close()
		}
	default:
		stream.Close()
	}
}

func (pn *ProxyNode) handleRegisterProxy(ctx context.Context, stream *P2PStream) error {
	peerID := stream.RemotePeerID
	pn.proxyLock.Lock()
	pn.proxies[peerID] = true
	pn.proxyLock.Unlock()

	status := ProxyStatus{
		PeerID:  peerID.String(),
		IsProxy: true,
	}
	statusBytes, err := json.Marshal(status)
	if err != nil {
		log.Printf("Error marshaling proxy status: %v\n", err)
		return err
	}

	err = pn.kadDHT.PutValue(ctx, "/proxies/"+peerID.String(), statusBytes)
	if err != nil {
		log.Printf("Error registering proxy: %v\n", err)
		return err
	}
	log.Printf("Successfully registered proxy: %s\n", peerID.String())
	return nil
}

func (pn *ProxyNode) handleUnregisterProxy(ctx context.Context, stream *P2PStream) error {
	peerID := stream.RemotePeerID
	pn.proxyLock.Lock()
	pn.proxies[peerID] = false
	pn.proxyLock.Unlock()

	status := ProxyStatus{
		PeerID:  peerID.String(),
		IsProxy: false,
	}
	statusBytes, err := json.Marshal(status)
	if err != nil {
		log.Printf("Error marshaling proxy status: %v\n", err)
		return err
	}

	err = pn.kadDHT.PutValue(ctx, "/proxies/"+peerID.String(), statusBytes)
	if err != nil {
		log.Printf("Error unregistering proxy: %v\n", err)
		return err
	}
	log.Printf("Successfully unregistered proxy: %s\n", peerID.String())
	return nil
}

func (pn *ProxyNode) RegisterAsProxy(ctx context.Context) error {
	peerID := pn.host.ID().String()
	status := ProxyStatus{
		PeerID:  peerID,
		IsProxy: true,
	}
	statusBytes, err := json.Marshal(status)
	if err != nil {
		log.Printf("Error marshaling proxy status: %v\n", err)
		return err
	}

	err = pn.kadDHT.PutValue(ctx, "/proxies/"+peerID, statusBytes)
	if err != nil {
		log.Printf("Error registering as proxy: %v\n", err)
		return err
	}
	log.Printf("Successfully registered as proxy: %s\n", peerID)
	return nil
}

func (pn *ProxyNode) UnregisterAsProxy(ctx context.Context) error {
	peerID := pn.host.ID().String()
	status := ProxyStatus{
		PeerID:  peerID,
		IsProxy: false,
	}
	statusBytes, err := json.Marshal(status)
	if err != nil {
		log.Printf("Error marshaling proxy status: %v\n", err)
		return err
	}

	err = pn.kadDHT.PutValue(ctx, "/proxies/"+peerID, statusBytes)
	if err != nil {
		log.Printf("Error unregistering as proxy: %v\n", err)
		return err
	}
	log.Printf("Successfully unregistered as proxy: %s\n", peerID)
	return nil
}

func (pn *ProxyNode) GetAllProxies(ctx context.Context) ([]ProxyStatus, error) {
	var proxies []ProxyStatus
	for _, key := range pn.kadDHT.RoutingTable().ListPeers() {
		value, err := pn.kadDHT.GetValue(ctx, "/proxies/"+key.String())
		if err == nil {
			var status ProxyStatus
			err = json.Unmarshal(value, &status)
			if err == nil && status.IsProxy {
				proxies = append(proxies, status)
			}
		}
	}
	return proxies, nil
}
