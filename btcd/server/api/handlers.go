package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/btcsuite/btcd/rpcclient"
)

type App struct {
	RpcClient  *rpcclient.Client
	Passphrase string
}

type TransferRequestData struct {
	Account string `json:"account"`
	Address string `json:"address"`
	Amount  int    `json:"amount"`
}

type AccountRequestData struct {
	Account string `json:"account"`
}

type TransactionRequestData struct {
	TxID string `json:"txid"`
}

func WriteSuccessResponse(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{"status": "success", "message": message}
	json.NewEncoder(w).Encode(response)
}

func WriteErrorResponse(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{"status": "error", "message": err.Error()}
	json.NewEncoder(w).Encode(response)
}

func (app *App) BalanceHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	// GET PARAMS

	query := params.Get("q")
	fmt.Printf("/balance query: %+v\n", query)
	balance, err := GetBalance(app.RpcClient, query)

	//

	if err != nil {
		log.Fatalf(err.Error())
	}

	fmt.Fprintf(w, "%v", balance)
}

func (app *App) TransferHandler(w http.ResponseWriter, r *http.Request) {

	// Decode the JSON body into the struct
	var data TransferRequestData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("/transfer: %+v\n", data)

	txid, err := SendToAddress(app.RpcClient, data.Account, data.Address, float64(data.Amount), app.Passphrase)
	if err != nil {
		WriteErrorResponse(w, err)
	} else {
		WriteSuccessResponse(w, txid.String())
	}
}

func (app *App) AccountHandler(w http.ResponseWriter, r *http.Request) {

	var data AccountRequestData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("/account: %+v\n", data)

	addr, err := GetAccountAddress(app.RpcClient, data.Account)

	if err != nil {
		WriteErrorResponse(w, err)
	} else {
		WriteSuccessResponse(w, addr.String())
	}
}

func (app *App) TransactionHandler(w http.ResponseWriter, r *http.Request) {

	var data TransactionRequestData

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("/transaction: %+v\n", data)

	txs, err := GetTransaction(app.RpcClient, data.TxID)
	if err != nil {
		WriteErrorResponse(w, err)
	} else {
		WriteSuccessResponse(w, strconv.Itoa(int(txs.Confirmations)))
	}
}

// func (app *App) HistoryHandler(w http.ResponseWriter, r *http.Request) {

// 	fmt.Printf("/history\n")

// 	txs, err := ListTransactionHistory(app.RpcClient, "default")

// 	if err != nil {
// 		WriteErrorResponse(w, err)
// 	} else {
// 		WriteSuccessResponse(w, strconv.Itoa(len(txs)))
// 	}
// }
