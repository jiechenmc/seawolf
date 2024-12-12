package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const proxyProtocol = "/orcanet/p2p/seawolf/proxy"
const proxyRequestTimeout = time.Second * 5
const tcpPort = ":8083"

type ProxyStatus struct {
	PeerID  string `json:"peer_id"`
	IsProxy bool   `json:"is_proxy"`
}

type ProxyNode struct {
	host        host.Host
	kadDHT      *dht.IpfsDHT
	proxies     map[peer.ID]bool
	proxyLock   sync.Mutex
	connected   bool
	proxyPeerID peer.ID
	clients     map[peer.ID]bool
}

func ProxyNodeCreate(hostNode host.Host, kadDHT *dht.IpfsDHT, isProxy bool) *ProxyNode {
	pn := &ProxyNode{
		host:        hostNode,
		kadDHT:      kadDHT,
		proxies:     make(map[peer.ID]bool),
		connected:   false,
		proxyPeerID: "",
	}
	hostNode.SetStreamHandler(proxyProtocol, pn.proxyStreamHandler)
	return pn
}

func (pn *ProxyNode) startTCPListener(proxyPeerID peer.ID) {
	listener, err := net.Listen("tcp", tcpPort)
	if err != nil {
		log.Fatalf("Failed to start TCP listener: %v", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		pn.proxyLock.Lock()
		if pn.connected {
			go pn.handleForwarding(conn)
		} else {
			conn.Close()
		}
		pn.proxyLock.Unlock()
	}
}

func (pn *ProxyNode) handleForwarding(conn net.Conn) {
	defer conn.Close()
	stream, err := p2pOpenStream(context.Background(), proxyProtocol, pn.host, pn.kadDHT, pn.proxyPeerID.String())
	if err != nil {
		log.Printf("Failed to create libp2p stream: %v", err)
		return
	}
	defer stream.Close()
	// Forward data from TCP connection to libp2p stream
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				log.Printf("Failed to read from TCP connection: %v", err)
				return
			}

			err = stream.Send(buf[:n])
			if err != nil {
				log.Printf("Failed to send data over libp2p stream: %v", err)
				return
			}
		}
	}()

	// Forward data from libp2p stream to TCP connection
	for {
		data, err := stream.Read(1, time.Minute)
		if err != nil {
			log.Printf("Failed to read from libp2p stream: %v", err)
			return
		}

		_, err = conn.Write(data)
		if err != nil {
			log.Printf("Failed to write to TCP connection: %v", err)
			return
		}
	}
}

// add to service
func (pn *ProxyNode) ConnectToProxy(proxyPeerID peer.ID) error {

	// check if proxyPeerID is a valid proxy
	if !pn.IsProxy(proxyPeerID) {
		return fmt.Errorf("peer is not a valid proxy")
	}

	// Establish a libp2p stream to the proxy
	stream, err := p2pOpenStream(context.Background(), proxyProtocol, pn.host, pn.kadDHT, proxyPeerID.String())
	if err != nil {
		log.Printf("Failed to create libp2p stream: %v", err)
		return err
	}
	defer stream.Close()

	err = stream.SendString("REQUEST\n")
	if err != nil {
		return err
	}

	str, err := stream.ReadString('\n', proxyRequestTimeout)
	if err != nil {
		return err
	}

	if str != "ACCEPT\n" {
		return fmt.Errorf("proxy rejected connection")
	}

	// FIXME get wallet address & price per byte from proxy

	pn.proxyLock.Lock()
	defer pn.proxyLock.Unlock()

	if pn.connected {
		return fmt.Errorf("Already connected to a proxy")
	}

	pn.connected = true
	pn.proxyPeerID = proxyPeerID
	return nil
}

func (pn *ProxyNode) IsProxy(peerID peer.ID) bool {
	pn.proxyLock.Lock()
	defer pn.proxyLock.Unlock()

	isProxy, ok := pn.proxies[peerID]
	return ok && isProxy
}

func (pn *ProxyNode) proxyStreamHandler(s network.Stream) {
	stream := p2pWrapStream(&s)
	pn.proxyLock.Lock()
	val, ok := pn.clients[stream.RemotePeerID]
	pn.proxyLock.Unlock()
	if ok && val {
		conn, err := net.Dial("tcp", "localhost:8082")
		if err != nil {
			log.Printf("Failed to connect to tcp server: %v", err)
			stream.Close()
			return
		}
		go pn.handleTraffic(conn, stream)
	} else {
		defer stream.Close()
		req, err := stream.ReadString('\n', proxyRequestTimeout)
		if err != nil {
			return
		}

		switch req {
		case "REQUEST\n":
			err = stream.SendString("ACCEPT\n")
			if err != nil {
				return
			}
			pn.proxyLock.Lock()
			pn.clients[stream.RemotePeerID] = true
			pn.proxyLock.Unlock()

		default:
			return
		}
	}
}

func (pn *ProxyNode) handleTraffic(conn net.Conn, stream *P2PStream) {
	defer conn.Close()
	defer stream.Close()
	// Forward data from TCP connection to libp2p stream
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				log.Printf("Failed to read from TCP connection: %v", err)
				return
			}

			err = stream.Send(buf[:n])
			if err != nil {
				log.Printf("Failed to send data over libp2p stream: %v", err)
				return
			}
		}
	}()

	// Forward data from libp2p stream to TCP connection
	for {
		data, err := stream.Read(1, time.Minute)
		if err != nil {
			log.Printf("Failed to read from libp2p stream: %v", err)
			return
		}

		_, err = conn.Write(data)
		if err != nil {
			log.Printf("Failed to write to TCP connection: %v", err)
			return
		}
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
	pn.proxyLock.Lock()
	pn.proxies[pn.host.ID()] = true
	pn.proxyLock.Unlock()

	status := ProxyStatus{
		PeerID:  pn.host.ID().String(),
		IsProxy: true,
	}
	statusBytes, err := json.Marshal(status)
	if err != nil {
		return err
	}

	return pn.kadDHT.PutValue(ctx, "/proxies/"+pn.host.ID().String(), statusBytes)
	// peerID := pn.host.ID().String()
	// status := ProxyStatus{
	// 	PeerID:  peerID,
	// 	IsProxy: true,
	// }
	// statusBytes, err := json.Marshal(status)
	// if err != nil {
	// 	log.Printf("Error marshaling proxy status: %v\n", err)
	// 	return err
	// }

	// err = pn.kadDHT.PutValue(ctx, "/proxies/"+peerID, statusBytes)
	// if err != nil {
	// 	log.Printf("Error registering as proxy: %v\n", err)
	// 	return err
	// }
	// log.Printf("Successfully registered as proxy: %s\n", peerID)
	// return nil
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
