package handlers

import (
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"kasplex-executor/api/models"
	"kasplex-executor/storage"
)

// GetTokenSnapshot needs to be exported (capital G)
func GetTokenSnapshot(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Snapshot endpoint hit for tick: %s", r.URL.Query().Get("tick"))

	if r.Method != http.MethodGet {
		sendResponse(w, http.StatusMethodNotAllowed, false, nil, "Method not allowed")
		return
	}

	// Get and validate the tick parameter
	tick := sanitizeString(r.URL.Query().Get("tick"))
	if !validateTick(tick) {
		log.Printf("DEBUG: Invalid tick parameter: %s", tick)
		sendResponse(w, http.StatusBadRequest, false, nil, "Invalid tick parameter: must be 4-6 uppercase letters (A-Z)")
		return
	}

	// Get token info first to get the max supply
	tokenInfo, err := storage.GetTokenInfo(tick)
	if err != nil {
		log.Printf("ERROR: Failed to fetch token info for %s: %v", tick, err)
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch token info: "+err.Error())
		return
	}

	// Get token balances from storage
	log.Printf("DEBUG: Fetching balances for tick: %s", tick)
	balances, err := storage.GetTokenBalances(tick)
	if err != nil {
		log.Printf("ERROR: Failed to fetch token balances for %s: %v", tick, err)
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch token balances: "+err.Error())
		return
	}

	log.Printf("DEBUG: Found %d balances for tick: %s", len(balances), tick)

	// Process the snapshot
	snapshot := processSnapshot(tick, balances, tokenInfo)
	sendResponse(w, http.StatusOK, true, snapshot, "")
}

func processSnapshot(tick string, balances []*models.TokenBalance, tokenInfo *models.TokenInfo) *models.TokenSnapshot {
	log.Printf("DEBUG: Processing snapshot for tick %s with %d balances", tick, len(balances))

	snapshot := &models.TokenSnapshot{
		Tick:      tick,
		Timestamp: time.Now().Unix(),
		Summary:   models.SnapshotSummary{},
		Holders:   make([]models.TokenHolder, 0),
	}

	// Parse token max from meta
	var metaData map[string]interface{}
	log.Printf("DEBUG: Token meta: %s", tokenInfo.Meta)

	if err := json.Unmarshal([]byte(tokenInfo.Meta), &metaData); err != nil {
		log.Printf("ERROR: Failed to parse meta for tick %s: %v", tick, err)
		return snapshot
	}

	maxStr, ok := metaData["max"].(string)
	if !ok {
		log.Printf("ERROR: Max value not found or not a string in meta for tick %s", tick)
		return snapshot
	}

	log.Printf("DEBUG: Max supply (string): %s", maxStr)

	var lockedTokens uint64
	var totalBalance uint64

	// Process each balance
	for _, balance := range balances {
		if balance.Balance > 0 || balance.Locked > 0 {
			lockedTokens += balance.Locked
			totalBalance += balance.Balance + balance.Locked

			holder := models.TokenHolder{
				Address: balance.Address,
				Balance: balance.Balance,
				Locked:  balance.Locked,
			}

			snapshot.Holders = append(snapshot.Holders, holder)
		}
	}

	log.Printf("DEBUG: Found %d holders, total balance: %d, locked: %d", len(snapshot.Holders), totalBalance, lockedTokens)

	// Calculate shares and summary
	for i := range snapshot.Holders {
		total := snapshot.Holders[i].Balance + snapshot.Holders[i].Locked
		// Parse maxStr as big.Int for accurate share calculation
		maxInt := new(big.Int)
		maxInt.SetString(maxStr, 10)
		if maxInt.Sign() > 0 {
			totalBig := new(big.Int).SetUint64(total)
			share := new(big.Float).Quo(
				new(big.Float).SetInt(totalBig),
				new(big.Float).SetInt(maxInt),
			)
			share = share.Mul(share, new(big.Float).SetInt64(100))
			shareFloat, _ := share.Float64()
			snapshot.Holders[i].Share = shareFloat
		}
	}

	// Convert locked tokens to string
	lockedStr := strconv.FormatUint(lockedTokens, 10)

	// Calculate circulating supply using big.Int for accuracy
	maxInt := new(big.Int)
	maxInt.SetString(maxStr, 10)
	lockedInt := new(big.Int)
	lockedInt.SetString(lockedStr, 10)
	circulatingInt := new(big.Int).Sub(maxInt, lockedInt)

	// Fill summary with string values
	snapshot.Summary = models.SnapshotSummary{
		TotalSupply:       maxStr,
		HoldersCount:      len(snapshot.Holders),
		LockedTokens:      lockedStr,
		CirculatingSupply: circulatingInt.String(),
	}

	log.Printf("DEBUG: Final summary - Total: %s, Holders: %d, Locked: %s, Circulating: %s",
		maxStr, len(snapshot.Holders), lockedStr, circulatingInt.String())

	return snapshot
}

// GetTokenCirculatingSupply returns only the circulating supply for a given token
func GetTokenCirculatingSupply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendResponse(w, http.StatusMethodNotAllowed, false, nil, "Method not allowed")
		return
	}

	// Get and validate the tick parameter
	tick := sanitizeString(r.URL.Query().Get("tick"))
	if !validateTick(tick) {
		sendResponse(w, http.StatusBadRequest, false, nil, "Invalid tick parameter: must be 4-6 uppercase letters (A-Z)")
		return
	}

	// Get token info to get the max supply
	tokenInfo, err := storage.GetTokenInfo(tick)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch token info: "+err.Error())
		return
	}

	// Parse token max from meta
	var metaData map[string]interface{}
	if err := json.Unmarshal([]byte(tokenInfo.Meta), &metaData); err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to parse meta data")
		return
	}

	maxStr, ok := metaData["max"].(string)
	if !ok {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Max value not found or invalid in token metadata")
		return
	}

	// Get token balances to calculate locked tokens
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

	// Calculate circulating supply using big.Int for accuracy
	maxInt := new(big.Int)
	maxInt.SetString(maxStr, 10)
	lockedInt := new(big.Int)
	lockedInt.SetString(lockedStr, 10)
	circulatingInt := new(big.Int).Sub(maxInt, lockedInt)

	// Create response with just the circulating supply
	response := map[string]string{
		"circulatingSupply": circulatingInt.String(),
	}

	sendResponse(w, http.StatusOK, true, response, "")
}
