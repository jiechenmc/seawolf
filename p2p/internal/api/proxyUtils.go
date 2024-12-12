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
const proxyDataProtocol = "/orcanet/p2p/seawolf/proxydata"
const proxyRequestTimeout = time.Second * 5
const tcpPort = ":8083"

type ProxyStatus struct {
	PeerID        string  `json:"peer_id"`
	IsProxy       bool    `json:"is_proxy"`
	Price         float64 `json:"price"`
	WalletAddress string  `json:"wallet_address"`
}

type ProxyNode struct {
	host        host.Host
	kadDHT      *dht.IpfsDHT
	proxies     map[peer.ID]ProxyStatus
	proxyLock   sync.Mutex
	connected   bool
	proxyPeerID peer.ID
	clients     map[peer.ID]bool
	bytesRx     int64
	bytesTx     int64
	listener    *net.Listener
}

type BytesTransferred struct {
	RxBytes int64 `json:"rx_bytes"`
	TxBytes int64 `json:"tx_bytes"`
}

func ProxyNodeCreate(hostNode host.Host, kadDHT *dht.IpfsDHT) (*ProxyNode, error) {
	pn := &ProxyNode{
		host:        hostNode,
		kadDHT:      kadDHT,
		proxies:     make(map[peer.ID]ProxyStatus),
		connected:   false,
		proxyPeerID: "",
		clients:     make(map[peer.ID]bool),
		bytesRx:     0,
		bytesTx:     0,
	}
	hostNode.SetStreamHandler(proxyProtocol, pn.proxyStreamHandler)
	hostNode.SetStreamHandler(proxyDataProtocol, pn.proxyDataStreamHandler)
	err := pn.startTCPListener()
	return pn, err
}

func (pn *ProxyNode) startTCPListener() error {
	listener, err := net.Listen("tcp", tcpPort)
	if err != nil {
        log.Printf("Failed to start TCP listener: %v", err)
        return internalError
    }
    pn.listener = &listener
    go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Failed to accept connection: %v", err)
				break
			}
			pn.proxyLock.Lock()
			if pn.connected {
				go pn.handleForwarding(conn)
			} else {
				conn.Close()
			}
			pn.proxyLock.Unlock()
		}
    }()
    return nil
}

func (pn *ProxyNode) handleForwarding(conn net.Conn) {
	defer conn.Close()
	stream, err := p2pOpenStream(context.Background(), proxyDataProtocol, pn.host, pn.kadDHT, pn.proxyPeerID.String())
	if err != nil {
		log.Printf("Failed to create libp2p stream: %v", err)
		return
	}
	defer stream.Close()
	// Forward data from TCP connection to libp2p stream
	go func() {
		buf := make([]byte, 1024*1024)
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
			pn.proxyLock.Lock()
			pn.bytesTx += int64(n)
			pn.proxyLock.Unlock()
		}
	}()

	// Forward data from libp2p stream to TCP connection
	for {
		buf := make([]byte, 1024*1024)
		n, err := stream.ReadBuffer(buf, time.Minute)
		if err != nil {
			log.Printf("Failed to read from libp2p stream: %v", err)
			return
		}

		_, err = conn.Write(buf[:n])
		if err != nil {
			log.Printf("Failed to write to TCP connection: %v", err)
			return
		}
		pn.proxyLock.Lock()
		pn.bytesRx += int64(n)
		pn.proxyLock.Unlock()
	}
}

// add to service
func (pn *ProxyNode) ConnectToProxy(proxyPeerID peer.ID) error {

	// check if proxyPeerID is a valid proxy
	if !pn.IsProxy(proxyPeerID) {
		return fmt.Errorf("peer is not a valid proxy")
	}
	if pn.IsProxy(pn.host.ID()) {
		return fmt.Errorf("You cannot connect to a proxy if you are a proxy")
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

	pn.proxyLock.Lock()
	defer pn.proxyLock.Unlock()

	if pn.connected {
		return fmt.Errorf("Already connected to a proxy")
	}

	pn.bytesRx = 0
	pn.bytesTx = 0

	pn.connected = true
	pn.proxyPeerID = proxyPeerID
	return nil
}

func (pn *ProxyNode) DisconnectFromProxy() error {
	pn.proxyLock.Lock()
	defer pn.proxyLock.Unlock()

	if !pn.connected {
		return fmt.Errorf("Not connected to a proxy")
	}
	stream, err := p2pOpenStream(context.Background(), proxyProtocol, pn.host, pn.kadDHT, pn.proxyPeerID.String())
	if err != nil {
		pn.connected = false
		pn.proxyPeerID = ""
		log.Printf("Failed to create libp2p stream: %v", err)
		return nil
	}
	defer stream.Close()

	stream.SendString("DISCONNECT\n")

	pn.connected = false
	pn.proxyPeerID = ""
	return nil
}

func (pn *ProxyNode) IsProxy(peerID peer.ID) bool {
	pn.proxyLock.Lock()
	defer pn.proxyLock.Unlock()

	status, ok := pn.proxies[peerID]
	return ok && status.IsProxy
}

func (pn *ProxyNode) proxyStreamHandler(s network.Stream) {
	stream := p2pWrapStream(&s)
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

	case "DISCONNECT\n":
		pn.proxyLock.Lock()
		pn.clients[stream.RemotePeerID] = false
		pn.proxyLock.Unlock()
		return
	default:
		return
	}
}

func (pn *ProxyNode) proxyDataStreamHandler(s network.Stream) {
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
		stream.Close()
	}
}

func (pn *ProxyNode) handleTraffic(conn net.Conn, stream *P2PStream) {
	defer conn.Close()
	defer stream.Close()
	// Forward data from TCP connection to libp2p stream
	go func() {
		buf := make([]byte, 1024*1024)
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
			pn.proxyLock.Lock()
			pn.bytesRx += int64(n)
			pn.proxyLock.Unlock()
		}
	}()

	// Forward data from libp2p stream to TCP connection
	for {
		buf := make([]byte, 1024*1024)
		n, err := stream.ReadBuffer(buf, time.Minute)
		if err != nil {
			log.Printf("Failed to read from libp2p stream: %v", err)
			return
		}

		_, err = conn.Write(buf[:n])
		if err != nil {
			log.Printf("Failed to write to TCP connection: %v", err)
			return
		}
		pn.proxyLock.Lock()
		pn.bytesTx += int64(n)
		pn.proxyLock.Unlock()
	}
}

func (pn *ProxyNode) RegisterAsProxy(ctx context.Context, price float64, walletAddr string) error {
	status := ProxyStatus{
		PeerID:        pn.host.ID().String(),
		IsProxy:       true,
		Price:         price,
		WalletAddress: walletAddr,
	}

	pn.proxyLock.Lock()
	pn.proxies[pn.host.ID()] = status
	pn.bytesRx = 0
	pn.bytesTx = 0
	pn.proxyLock.Unlock()

	statusBytes, err := json.Marshal(status)
	if err != nil {
		return err
	}

	return pn.kadDHT.PutValue(ctx, "/orcanet/proxies/"+pn.host.ID().String(), statusBytes)
}

func (pn *ProxyNode) UnregisterAsProxy(ctx context.Context) error {
	peerID := pn.host.ID().String()
	status := ProxyStatus{
		PeerID:        peerID,
		IsProxy:       false,
		Price:         0,
		WalletAddress: "",
	}
	pn.proxyLock.Lock()
	pn.proxies[pn.host.ID()] = status
	pn.proxyLock.Unlock()
	statusBytes, err := json.Marshal(status)
	if err != nil {
		log.Printf("Error marshaling proxy status: %v\n", err)
		return err
	}

	err = pn.kadDHT.PutValue(ctx, "/orcanet/proxies/"+peerID, statusBytes)
	if err != nil {
		log.Printf("Error unregistering as proxy: %v\n", err)
		return err
	}
	log.Printf("Successfully unregistered as proxy: %s\n", peerID)
	return nil
}

func (pn *ProxyNode) GetBytes() BytesTransferred {
	pn.proxyLock.Lock()
	defer pn.proxyLock.Unlock()
	return BytesTransferred{pn.bytesRx, pn.bytesTx}
}

func (pn *ProxyNode) GetAllProxies(ctx context.Context) ([]ProxyStatus, error) {
	pn.proxyLock.Lock()
	defer pn.proxyLock.Unlock()
	pn.proxies = make(map[peer.ID]ProxyStatus)
	var proxies []ProxyStatus
	peerIDs := pn.host.Peerstore().Peers()
	for _, key := range peerIDs {
		value, err := pn.kadDHT.GetValue(ctx, "/orcanet/proxies/"+key.String())
		if err == nil {
			var status ProxyStatus
			err = json.Unmarshal(value, &status)
			if err == nil && status.IsProxy {
				proxies = append(proxies, status)
				pn.proxies[key] = status
			}
		}
	}
	return proxies, nil
}

func (pn *ProxyNode) Close() {
    if pn.listener != nil {
        (*pn.listener).Close()
        pn.listener = nil
    }
}
