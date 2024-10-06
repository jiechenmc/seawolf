package api

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
)

// Note to self
//
// -> Money
// getbalance
// listunspent
// listreceivedbyaddress
// sendtoaddress <addr> <amt>
//
// --> Transactions are processed when blocks are mined! <--

func GetBalance(client *rpcclient.Client) (float64, error) {
	balance, err := client.GetBalance("*")
	return balance.ToBTC(), err
}

func SendToAddress(client *rpcclient.Client, addressStr string, btcAmount float64, passphrase string) (*chainhash.Hash, error) {
	err := client.WalletPassphrase(passphrase, 3600)

	if err != nil {
		// log.Fatalf("The wallet was not able to be unlocked\n%v", err)
		return nil, err
	}

	// TODO: change this when we start to connect to the TA's network
	address, err := btcutil.DecodeAddress(addressStr, &chaincfg.SimNetParams)

	if err != nil {
		// log.Fatalf("Failed to decode address: %v", err)
		return nil, err
	}

	chain, err := client.SendToAddress(btcutil.Address(address), btcutil.Amount(btcAmount*1e8))
	return chain, err
}
