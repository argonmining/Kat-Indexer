package handlers

import (
	"encoding/json"
	"kasplex-executor/api/models"
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

	// Parse the meta string into a map
	var metaData map[string]interface{}
	if err := json.Unmarshal([]byte(info.Meta), &metaData); err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to parse meta data")
		return
	}

	// Create a new response structure
	response := map[string]interface{}{
		"tick":    info.Tick,
		"max":     metaData["max"],
		"lim":     metaData["lim"],
		"pre":     metaData["pre"],
		"dec":     metaData["dec"],
		"from":    metaData["from"],
		"to":      metaData["to"],
		"txid":    metaData["txid"],
		"opadd":   metaData["opadd"],
		"mtsadd":  metaData["mtsadd"],
		"minted":  info.Minted,
		"op_mod":  info.OpMod,
		"mts_mod": info.MtsMod,
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

	response := models.TokenListResponse{
		Result: tokens,
	}

	sendResponse(w, http.StatusOK, true, response, "")
}
