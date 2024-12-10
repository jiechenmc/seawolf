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
// unlockWallet
// createAccount
// getAddressesByAccount
// getAccountAddress
// getAccountFromAddress
// getbalance
// sendFrom <account> <addr> <amt>
//
// --> Transactions are processed when blocks are mined! <--

func UnlockWallet(client *rpcclient.Client, passphrase string) error {
	err := client.WalletPassphrase(passphrase, 3600)
	return err
}

func CreateAccount(client *rpcclient.Client, accountName string, passphrase string) error {
	err := UnlockWallet(client, passphrase)
	if err != nil {
		return err
	}
	err = client.CreateNewAccount(accountName)
	return err
}

//@ NOT USED
// func GetAddressesByAccount(client *rpcclient.Client, account string) ([]btcutil.Address, error) {
// 	return client.GetAddressesByAccount(account)
// }

func GetAccountAddress(client *rpcclient.Client, account string) (btcutil.Address, error) {
	return client.GetAccountAddress(account)
}

func GetAccountFromAddress(client *rpcclient.Client, addressStr string) (string, error) {
	address, err := btcutil.DecodeAddress(addressStr, &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}
	return client.GetAccount(address)
}

func GetBalance(client *rpcclient.Client, account string) (float64, error) {
	balance, err := client.GetBalance(account)
	return balance.ToBTC(), err
}

func SendToAddress(client *rpcclient.Client, fromAccount string, addressStr string, btcAmount float64, passphrase string) (*chainhash.Hash, error) {
	err := UnlockWallet(client, passphrase)

	if err != nil {
		// log.Fatalf("The wallet was not able to be unlocked\n%v", err)
		return nil, err
	}

	address, err := btcutil.DecodeAddress(addressStr, &chaincfg.MainNetParams)

	if err != nil {
		// log.Fatalf("Failed to decode address: %v", err)
		return nil, err
	}

	chain, err := client.SendFrom(fromAccount, btcutil.Address(address), btcutil.Amount(btcAmount*1e8))
	return chain, err
}
