package api

import (
    "log"
    "net/http"
    "github.com/ethereum/go-ethereum/rpc"
)

type API struct {
    rpcServer *rpc.Server
}

func APIServer() *API {
    //Create interface for frontend
    p2pService := new(P2PService)
    server := rpc.NewServer()
    server.RegisterName("p2p", p2pService)
    api := &API{ rpcServer: server }

    return api
}

func (a *API) Start(listenAddr string) {
    http.HandleFunc("/rpc", a.rpcServer.ServeHTTP)
    err := http.ListenAndServe(listenAddr, nil)
    if err != nil {
        log.Printf("Error starting server. %v\n", err)
    }
}
