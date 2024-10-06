// Copyright (c) 2014-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/davecgh/go-spew/spew"
)

// -> Admin
// createwallet <name> <passphrase>

// Note to self
//
// -> Money
// getbalance
// listunspent
// listreceivedbyaddress
// sendtoaddress <addr> <amt>
//
// --> Transactions are processed when blocks are mined! <--

// -> Logistics
// getblockchaininfo
// getaccountaddress <account>
// gettransaction <txid>
// getrawmempool <- mempool shows pending transactions

// current addresses for testing:
// SQvC2vyTrCtZnEoqhRMJozK3k2ovauhCEt
// SV3AKDppayuuBVADVSbiAs6Nxgr7HMmfw6
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

	// Get the list of unspent transaction outputs (utxos) that the
	// connected wallet has at least one private key for.
	unspent, err := client.ListUnspent()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Num unspent outputs (utxos): %d", len(unspent))
	if len(unspent) > 0 {
		log.Printf("First utxo:\n%v", spew.Sdump(unspent[0]))
	}

	// For this example gracefully shutdown the client after 10 seconds.
	// Ordinarily when to shutdown the client is highly application
	// specific.
	// log.Println("Client shutdown in 10 seconds...")
	// time.AfterFunc(time.Second*10, func() {
	// 	log.Println("Client shutting down...")
	// 	client.Shutdown()
	// 	log.Println("Client shutdown complete.")
	// })

	// Wait until the client either shuts down gracefully (or the user
	// terminates the process with Ctrl+C).
	client.WaitForShutdown()
}
