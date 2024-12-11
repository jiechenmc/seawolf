// Copyright (c) 2014-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"seawolf/coin/api"
	"syscall"
	"time"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/creack/pty"
)

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

func spawnWallet(ctx context.Context, seed string) *exec.Cmd {

	//root/.btcwallet/mainnet/wallet.db

	var cmd *exec.Cmd

	_, err := os.Stat("/root/.btcwallet/mainnet/wallet.db")

	if !os.IsNotExist(err) {
		cmd = exec.Command("btcwallet", "-u", os.Getenv("btcdusername"), "-P", os.Getenv("btcdpassword"))
		err := cmd.Start()
		if err != nil {
			log.Fatalf("Failed to start cmd: %v", err)
		}

		log.Printf("btcwallet is running with PID %d", cmd.Process.Pid)

		go func() {
			<-ctx.Done()
			log.Println("Shutting down btcwallet...")
			cmd.Process.Signal(syscall.SIGKILL)
		}()
	} else {
		cmd = exec.Command("btcwallet", "-u", os.Getenv("btcdusername"), "-P", os.Getenv("btcdpassword"), "--create")

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		f, err := pty.Start(cmd)
		if err != nil {
			log.Fatalf("Failed to start pty: %v", err)
		}

		defer f.Close()

		log.Println("Writing Password")
		passphrase := "cse416\r"

		if _, err := f.Write([]byte(passphrase)); err != nil {
			log.Fatalf("Failed to write password: %v", err)
		}
		log.Println("Confirming Password")

		if _, err := f.Write([]byte(passphrase)); err != nil {
			log.Fatalf("Failed to confirm password: %v", err)
		}
		log.Println("ENCRYPTION")

		if _, err := f.Write([]byte("no\r")); err != nil {
			log.Fatalf("Failed to say no to encryption: %v", err)
		}

		if seed != "" {
			log.Println("SEED is Present")
			if _, err := f.Write([]byte("yes\r")); err != nil {
				log.Fatalf("Failed to say Yes to seed: %v", err)
			}
			log.Println("ENTERING SEED")
			if _, err := f.Write([]byte(fmt.Sprintf("%s\r", seed))); err != nil {
				log.Fatalf("Failed to enter seed: %v", err)
			}
		} else {
			log.Println("NO SEED")
			if _, err := f.Write([]byte("no\r")); err != nil {
				log.Fatalf("Failed to say no to seed: %v", err)
			}
		}

		log.Println("Confirm seed is kept safe")
		if _, err := f.Write([]byte("OK\r")); err != nil {
			log.Fatalf("Failed to confirm wallet seed is kept safe: %v", err)
		}

		io.Copy(os.Stdout, f)

		cmd = exec.Command("btcwallet", "-u", os.Getenv("btcdusername"), "-P", os.Getenv("btcdpassword"))
		err = cmd.Start()
		if err != nil {
			log.Fatalf("Failed to start cmd: %v", err)
		}

		log.Printf("btcwallet is running with PID %d", cmd.Process.Pid)

		go func() {
			<-ctx.Done()
			log.Println("Shutting down btcwallet...")
			cmd.Process.Signal(syscall.SIGKILL)
		}()

	}

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
	spawnWallet(ctx, os.Getenv("WALLET_SEED"))

	ntfnHandlers := rpcclient.NotificationHandlers{}

	// Connect to local btcwallet RPC server using websockets.
	// certHomeDir := btcutil.AppDataDir("btcwallet", false)
	// certs, err := os.ReadFile(filepath.Join(certHomeDir, "rpc.cert"))
	// if err != nil {
	// 	log.Fatal(err)
	// }

	for {
		conn, err := net.DialTimeout("tcp", "localhost:8332", 2*time.Second)
		if err == nil {
			conn.Close()
			break
		}
		fmt.Println("Waiting for btcwallet to be ready...")
		time.Sleep(1 * time.Second)
	}

	connCfg := &rpcclient.ConnConfig{
		Host:       "localhost:8332",
		Endpoint:   "ws",
		User:       "rpcuser",
		Pass:       "rpcpass",
		DisableTLS: true,
		// Certificates: certs,
	}

	client, err := rpcclient.New(connCfg, &ntfnHandlers)

	if err != nil {
		log.Panic(err)
	}

	app := &api.App{
		RpcClient:  client,
		Passphrase: "cse416",
	}

	http.HandleFunc("/balance", app.BalanceHandler)
	http.HandleFunc("/transfer", app.TransferHandler)
	// http.HandleFunc("/account", app.AccountHandler)
	http.HandleFunc("/transactions", app.TransactionHandler)
	// http.HandleFunc("/history", app.HistoryHandler)

	fmt.Println("Server is listening on port 8080...")
	err = http.ListenAndServe(":8080", nil) // Start the HTTP server on port 8080

	if err != nil {
		fmt.Println("Error starting server:", err)
	}
	<-ctx.Done()
	defer client.Shutdown()
}
