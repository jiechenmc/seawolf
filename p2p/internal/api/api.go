package api

import (
    "github.com/ethereum/go-ethereum/rpc"
)

func APIServer() *rpc.Server {
    //Create interface for frontend
    p2pService := new(P2PService)
    server := rpc.NewServer()
    server.RegisterName("p2p", p2pService)

    return server
}
