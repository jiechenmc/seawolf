// Copyright (c) 2014-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"seawolf/coin/api"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/rpcclient"
)

// -> Admin
// createwallet <name> <passphrase>

// current addresses for testing:
// SQvC2vyTrCtZnEoqhRMJozK3k2ovauhCEt
// SV3AKDppayuuBVADVSbiAs6Nxgr7HMmfw6
// Si9teS1ayrGzhynLCYGaRA73wnuGgGbBq7
// seed:
// 6779ea5e457d009b17b842510c755d89d781cb56507d05ae3c9efec062567b26

func main() {
	// Only override the handlers for notifications you care about.
	// Also note most of the handlers will only be called if you register
	// for notifications.  See the documentation of the rpcclient
	// NotificationHandlers type for more details about each handler.
	ntfnHandlers := rpcclient.NotificationHandlers{
		OnAccountBalance: func(account string, balance btcutil.Amount, confirmed bool) {
			log.Printf("New balance for account %s: %v", account,
				balance)
		},
		OnWalletLockState: func(locked bool) {
			log.Printf("%v", locked)
		},
	}

	// Connect to local btcwallet RPC server using websockets.
	certHomeDir := btcutil.AppDataDir("btcwallet", false)
	certs, err := os.ReadFile(filepath.Join(certHomeDir, "rpc.cert"))
	if err != nil {
		log.Fatal(err)
	}
	connCfg := &rpcclient.ConnConfig{
		Host:         "localhost:18554",
		Endpoint:     "ws",
		User:         "rpcuser",
		Pass:         "rpcpass",
		Certificates: certs,
	}

	client, err := rpcclient.New(connCfg, &ntfnHandlers)
	if err != nil {
		log.Fatal(err)
	}

	app := &api.App{
		RpcClient:  client,
		Passphrase: "2578547813",
	}

	http.HandleFunc("/balance", app.BalanceHandler)
	http.HandleFunc("/transfer", app.TransferHandler)
	http.HandleFunc("/account", app.AccountHandler)

	fmt.Println("Server is listening on port 8080...")
	err = http.ListenAndServe(":8080", nil) // Start the HTTP server on port 8080

	if err != nil {
		fmt.Println("Error starting server:", err)
	}

	defer client.Shutdown()
}
