package handlers

import (
	"kasplex-executor/storage"
	"net/http"
)

func GetAddressBalances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendResponse(w, http.StatusMethodNotAllowed, false, nil, "Method not allowed")
		return
	}

	address := r.URL.Query().Get("address")
	if address == "" {
		sendResponse(w, http.StatusBadRequest, false, nil, "Address parameter is required")
		return
	}

	// Query storage for address balances
	balances, err := storage.GetAddressBalances(address)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch address balances")
		return
	}

	sendResponse(w, http.StatusOK, true, balances, "")
}
