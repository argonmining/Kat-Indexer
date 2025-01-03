package handlers

import (
	"encoding/json"
	"kasplex-executor/storage"
	"math/big"
	"net/http"
	"strconv"
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

	// Parse the meta string into a map
	var metaData map[string]interface{}
	if err := json.Unmarshal([]byte(info.Meta), &metaData); err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to parse meta data")
		return
	}

	// Get txid value and handle potential nil case
	var hashRev interface{}
	if txid, exists := metaData["txid"]; exists && txid != nil {
		hashRev = txid
	}

	// Get token balances to calculate locked tokens and circulating supply
	balances, err := storage.GetTokenBalances(tick)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch token balances")
		return
	}

	// Calculate total locked tokens
	var lockedTokens uint64
	for _, balance := range balances {
		lockedTokens += balance.Locked
	}

	// Convert locked tokens to string for big number operations
	lockedStr := strconv.FormatUint(lockedTokens, 10)

	// Get max supply from metadata
	maxStr, ok := metaData["max"].(string)
	if !ok {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Max value not found or invalid in token metadata")
		return
	}

	// Calculate circulating supply using big.Int for accuracy
	maxInt := new(big.Int)
	maxInt.SetString(maxStr, 10)
	lockedInt := new(big.Int)
	lockedInt.SetString(lockedStr, 10)
	circulatingInt := new(big.Int).Sub(maxInt, lockedInt)

	// Create a new response structure
	response := map[string]interface{}{
		"tick":              info.Tick,
		"max":               metaData["max"],
		"lim":               metaData["lim"],
		"pre":               metaData["pre"],
		"dec":               metaData["dec"],
		"from":              metaData["from"],
		"to":                metaData["to"],
		"hashRev":           hashRev,
		"opadd":             metaData["opadd"],
		"mtsadd":            metaData["mtsadd"],
		"minted":            info.Minted,
		"op_mod":            info.OpMod,
		"mts_mod":           info.MtsMod,
		"lockedTokens":      lockedStr,
		"circulatingSupply": circulatingInt.String(),
	}

	sendResponse(w, http.StatusOK, true, response, "")
}

func GetAllTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendResponse(w, http.StatusMethodNotAllowed, false, nil, "Method not allowed")
		return
	}

	tokens, err := storage.GetAllTokens()
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch tokens: "+err.Error())
		return
	}

	sendResponse(w, http.StatusOK, true, tokens, "")
}
