package api

import (
    "log"
    "net/http"
    "github.com/ethereum/go-ethereum/rpc"
)

type API struct {
    rpcServer *rpc.Server
}

func enableCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set CORS headers
        w.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins or set specific origin
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS") // Allow methods
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization") // Allow headers

        // Handle preflight requests (OPTIONS request)
        if r.Method == "OPTIONS" {
            return
        }

        next.ServeHTTP(w, r)
    })
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
    // http.HandleFunc("/rpc", a.rpcServer.ServeHTTP)
    http.Handle("/rpc", enableCORS(http.HandlerFunc(a.rpcServer.ServeHTTP)))
    err := http.ListenAndServe(listenAddr, nil)
    if err != nil {
        log.Printf("Error starting server. %v\n", err)
    }
}
