package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/things-go/go-socks5"
)

// GetLocalIP returns the non-loopback local IP address of the host
func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		// Check the address type and if it is not a loopback address then return it
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no non-loopback address found")
}

func GetPublicIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org?format=text")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Failed to get public IP", err
	}
	return string(ip), nil
}

func main() {
	// Create a SOCKS5 server
	server := socks5.NewServer(
		socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
	)

	// Start the SOCKS5 proxy on localhost port 8082
	go func() {
		if err := server.ListenAndServe("tcp", ":8082"); err != nil {
			panic(err)
		}
	}()

	// Get the local IP address
	ip, err := GetLocalIP()
	if err != nil {
		log.Printf("Error getting local IP: %v\n", err)
		return
	}

	port := "8082"
	err = nil // make a json rpc request to register as Proxy  proxyNode.RegisterAsProxy(context.Background(), ip, port)
	if err != nil {
		log.Printf("Error registering as proxy: %v\n", err)
		return
	}

	log.Printf("Successfully registered as proxy on %s:%s\n", ip, port)

	// Keep the main function running
	select {}
}
