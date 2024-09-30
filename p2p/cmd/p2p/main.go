package main

import (
    "fmt"
    "net"
    "os"
    "os/signal"
    "syscall"
    "github.com/jiechenmc/seawolf/p2p/internal/api"
)

const socketPath = "seawolf_p2p.sock"

func cleanup(exitCode int) {
    os.Remove(socketPath)
    os.Exit(exitCode)
}

func sigtermHandler(c <-chan os.Signal) {
    <-c
    cleanup(1)
}

func main() {
    // ln, err := net.Listen("tcp", "127.0.0.1:1234")
    os.Remove(socketPath);
    socket, err := net.Listen("unix", socketPath)
    if err != nil {
        fmt.Printf("Failed to listen to socket %v!\n", socketPath)
        os.Exit(1)
    }
    err = os.Chmod(socketPath, 0600)
    if err != nil {
        fmt.Printf("Failed to set socket permissions!\n")
        os.Exit(1)
    }
    // Cleanup the socket file
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go sigtermHandler(c);
    
    api.APIServer().ServeListener(socket)

    //Clean up
    cleanup(0)
}
