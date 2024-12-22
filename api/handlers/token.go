package handlers

import (
	"kasplex-executor/storage"
	"net/http"
)

func GetTokenBalances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendResponse(w, http.StatusMethodNotAllowed, false, nil, "Method not allowed")
		return
	}

	tick := r.URL.Query().Get("tick")
	if !validateTick(tick) {
		sendResponse(w, http.StatusBadRequest, false, nil, "Invalid tick parameter")
		return
	}

	// Query storage for balances
	balances, err := storage.GetTokenBalances(tick)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch balances")
		return
	}

	sendResponse(w, http.StatusOK, true, balances, "")
}

func GetTokenInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendResponse(w, http.StatusMethodNotAllowed, false, nil, "Method not allowed")
		return
	}

	tick := r.URL.Query().Get("tick")
	if !validateTick(tick) {
		sendResponse(w, http.StatusBadRequest, false, nil, "Invalid tick parameter")
		return
	}

	// Query storage for token info
	info, err := storage.GetTokenInfo(tick)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch token info")
		return
	}

	sendResponse(w, http.StatusOK, true, info, "")
}
