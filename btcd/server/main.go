// Copyright (c) 2014-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"seawolf/coin/api"
	"syscall"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/rpcclient"
)

// -> Admin
// createwallet <name> <passphrase>
func spawnBtcd(ctx context.Context) *exec.Cmd {
	cmd := exec.Command("btcd", "-a", "130.245.173.221:8333", "--miningaddr", "1AMMu8eiCkyNA6z7e4w12udCN95eBTaBq1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process in the background
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start btcd: %v", err)
	}

	log.Printf("btcd is running with PID %d", cmd.Process.Pid)

	go func() {
		<-ctx.Done()
		log.Println("Shutting down btcd...")
		cmd.Process.Signal(syscall.SIGKILL)
	}()
	return cmd
}

func spawnWallet(ctx context.Context) *exec.Cmd {

	cmd := exec.Command("btcwallet", "-u", os.Getenv("btcdusername"), "-P", os.Getenv("btcdpassword"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process in the background
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start btcwallet: %v", err)
	}

	log.Printf("btcwallet is running with PID %d", cmd.Process.Pid)

	go func() {
		<-ctx.Done()
		log.Println("Shutting down btcwallet...")
		cmd.Process.Signal(syscall.SIGKILL)
	}()
	return cmd
}

func main() {
	// Only override the handlers for notifications you care about.
	// Also note most of the handlers will only be called if you register
	// for notifications.  See the documentation of the rpcclient
	// NotificationHandlers type for more details about each handler.

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	spawnBtcd(ctx)
	spawnWallet(ctx)

	time.Sleep(10 * time.Second)

	ntfnHandlers := rpcclient.NotificationHandlers{}

	// Connect to local btcwallet RPC server using websockets.
	certHomeDir := btcutil.AppDataDir("btcwallet", false)
	certs, err := os.ReadFile(filepath.Join(certHomeDir, "rpc.cert"))
	if err != nil {
		log.Fatal(err)
	}
	connCfg := &rpcclient.ConnConfig{
		Host:         "localhost:8332",
		Endpoint:     "ws",
		User:         "rpcuser",
		Pass:         "rpcpass",
		DisableTLS:   true,
		Certificates: certs,
	}

	client, err := rpcclient.New(connCfg, &ntfnHandlers)

	if err != nil {
		log.Fatal(err)
	}

	app := &api.App{
		RpcClient:  client,
		Passphrase: "cse416",
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
