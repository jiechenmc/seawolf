package main

import (
    "github.com/jiechenmc/seawolf/p2p/internal/api"
)

const listen_address = "127.0.0.1:8081"

func main() {
    api.APIServer().Start(listen_address)
}
