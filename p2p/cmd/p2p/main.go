package main

import (
    "fmt"
    "net"
    "os"
    "github.com/jiechenmc/seawolf/p2p/internal/api"
)

const listen_address = "127.0.0.1:8081"

func main() {
    tcp_socket, err := net.Listen("tcp", "127.0.0.1:8081")
    if err != nil {
        fmt.Printf("Failed to listen to port %v!\n", "127.0.0.1:8081")
        os.Exit(1)
    }
    api.APIServer().ServeListener(tcp_socket)
}
