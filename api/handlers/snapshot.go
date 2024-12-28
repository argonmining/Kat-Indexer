package handlers

import (
	"log"
	"net/http"
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
	snapshot := processSnapshot(tick, balances)
	sendResponse(w, http.StatusOK, true, snapshot, "")
}

func processSnapshot(tick string, balances []*models.TokenBalance) *models.TokenSnapshot {
	snapshot := &models.TokenSnapshot{
		Tick:      tick,
		Timestamp: time.Now().Unix(),
		Holders:   make([]models.TokenHolder, 0),
		Summary:   models.SnapshotSummary{},
	}

	var totalSupply uint64
	var lockedTokens uint64

	// Process each balance
	for _, balance := range balances {
		if balance.Balance > 0 || balance.Locked > 0 {
			totalSupply += uint64(balance.Balance) + uint64(balance.Locked)
			lockedTokens += uint64(balance.Locked)

			holder := models.TokenHolder{
				Address: balance.Address,
				Balance: balance.Balance,
				Locked:  balance.Locked,
			}
			snapshot.Holders = append(snapshot.Holders, holder)
		}
	}

	// Calculate shares and summary
	for i := range snapshot.Holders {
		total := snapshot.Holders[i].Balance + snapshot.Holders[i].Locked
		if totalSupply > 0 {
			snapshot.Holders[i].Share = (float64(total) / float64(totalSupply)) * 100.0
		}
	}

	// Fill summary
	snapshot.Summary = models.SnapshotSummary{
		TotalSupply:       totalSupply,
		HoldersCount:      len(snapshot.Holders),
		LockedTokens:      lockedTokens,
		CirculatingSupply: totalSupply - lockedTokens,
	}

	return snapshot
}
