package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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

	address, err := GetAccountAddress(app.RpcClient, data.Account)

	if err != nil {
		WriteErrorResponse(w, err)
	} else {
		WriteSuccessResponse(w, address.String())
	}
}
