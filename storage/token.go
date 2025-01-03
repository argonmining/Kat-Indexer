package storage

import (
	"encoding/json"
	"fmt"
	"kasplex-executor/api/models"
	"log"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/gocql/gocql"
)

func GetTokenBalances(tick string) ([]*models.TokenBalance, error) {
	log.Printf("Fetching all balances for tick: %s", tick)

	// Use a larger page size since we need all records
	query := sRuntime.sessionCassa.Query(`
		SELECT address, tick, dec, balance, locked 
		FROM stbalance 
		WHERE tick = ?`,
		tick,
	).PageSize(10000) // Larger page size for bulk retrieval

	iter := query.Iter()
	var address, balance, locked string
	var dec int
	var tickResult string
	balances := make([]*models.TokenBalance, 0, 1000) // Start with reasonable capacity

	for iter.Scan(&address, &tickResult, &dec, &balance, &locked) {
		// Only append non-zero balances
		bal := parseStringToUint64(balance)
		lock := parseStringToUint64(locked)
		if bal > 0 || lock > 0 {
			balances = append(balances, &models.TokenBalance{
				Address: address,
				Balance: bal,
				Locked:  lock,
				Dec:     dec,
			})
		}
	}

	if err := iter.Close(); err != nil {
		log.Printf("ERROR: Failed to close iterator for tick %s: %v", tick, err)
		return nil, err
	}

	log.Printf("Found %d non-zero balances for tick: %s", len(balances), tick)
	return balances, nil
}

func GetTokenInfo(tick string) (*models.TokenInfo, error) {
	// Get token info from Cassandra
	var meta, minted string
	var opMod uint64
	var mtsMod int64

	if err := sRuntime.sessionCassa.Query("SELECT meta, minted, opmod, mtsmod FROM sttoken WHERE p2tick = ? AND tick = ?", tick[:2], tick).Scan(&meta, &minted, &opMod, &mtsMod); err != nil {
		return nil, err
	}

	return &models.TokenInfo{
		Tick:   tick,
		Meta:   meta,
		Minted: parseStringToUint64(minted),
		OpMod:  int64(opMod),
		MtsMod: mtsMod,
	}, nil
}

func parseStringToUint64(s string) uint64 {
	// Helper function to parse string to uint64
	var result uint64
	json.Unmarshal([]byte(s), &result)
	return result
}

func GetTokenHoldersPaginated(tick string, page, pageSize int) ([]models.HolderInfo, int, error) {
	// First get token info to get max supply
	tokenInfo, err := GetTokenInfo(tick)
	if err != nil {
		log.Printf("ERROR: Failed to get token info for %s: %v", tick, err)
		return nil, 0, err
	}

	// Parse meta to get max supply
	var metaData map[string]interface{}
	if err := json.Unmarshal([]byte(tokenInfo.Meta), &metaData); err != nil {
		log.Printf("ERROR: Failed to parse meta for tick %s: %v", tick, err)
		return nil, 0, err
	}

	maxStr, ok := metaData["max"].(string)
	if !ok {
		log.Printf("ERROR: Max value not found or not a string in meta for tick %s", tick)
		return nil, 0, err
	}

	// Parse max supply using big.Int
	maxInt := new(big.Int)
	maxInt.SetString(maxStr, 10)

	// Use existing index on stbalance table
	query := sRuntime.sessionCassa.Query(`
		SELECT address, balance, locked 
		FROM stbalance 
			WHERE tick = ? 
		ALLOW FILTERING`,
		tick,
	).PageSize(2000)

	iter := query.Iter()
	var address, balance, locked string
	holders := make([]models.HolderInfo, 0, 2000)

	// Collect all non-zero balances
	for iter.Scan(&address, &balance, &locked) {
		bal := parseStringToUint64(balance)
		lock := parseStringToUint64(locked)
		total := bal + lock
		if total > 0 {
			holders = append(holders, models.HolderInfo{
				Address: address,
				Balance: bal,
				Locked:  lock,
			})
		}
	}

	if err := iter.Close(); err != nil {
		return nil, 0, err
	}

	// Sort by total balance (balance + locked)
	sort.Slice(holders, func(i, j int) bool {
		totalI := holders[i].Balance + holders[i].Locked
		totalJ := holders[j].Balance + holders[j].Locked
		return totalI > totalJ
	})

	// Calculate pagination
	total := len(holders)
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}

	// Calculate ranks and shares for the page using big.Int/big.Float for accuracy
	for i := start; i < end; i++ {
		total := holders[i].Balance + holders[i].Locked
		holders[i].Rank = i + 1

		// Calculate share using big numbers
		totalBig := new(big.Int).SetUint64(total)
		share := new(big.Float).Quo(
			new(big.Float).SetInt(totalBig),
			new(big.Float).SetInt(maxInt),
		)
		share = share.Mul(share, new(big.Float).SetInt64(100))
		shareFloat, _ := share.Float64()
		holders[i].Share = shareFloat
	}

	return holders[start:end], total, nil
}

func GetTokenOperationsPaginated(tick string, offset, pageSize int) ([]models.Operation, bool, error) {
	log.Printf("Fetching operations for tick: %s, offset: %d, pageSize: %d", tick, offset, pageSize)

	// Query the materialized view which is already ordered by opscore DESC
	query := sRuntime.sessionCassa.Query(`
		SELECT oprange, opscore, txid, state, script, tickaffc, addressaffc 
		FROM oplist_by_tick 
		WHERE tickaffc >= ? AND tickaffc < ?
		LIMIT ?
		ALLOW FILTERING`,
		tick+"=",   // Start of range (e.g., "NACHO=")
		tick+"=~",  // End of range (~ comes after any digit in ASCII)
		pageSize+1, // Get one extra to determine if there are more pages
	).PageSize(pageSize + 1).RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: 3})

	iter := query.Iter()
	var oprange int64
	var opScore uint64
	var txid, state, script, tickAffc, addressaffc string
	operations := make([]models.Operation, 0, pageSize)
	hasMore := false
	recordCount := 0

	for iter.Scan(&oprange, &opScore, &txid, &state, &script, &tickAffc, &addressaffc) {
		// Skip records for pagination
		if recordCount < offset {
			recordCount++
			continue
		}

		// If we've got more than pageSize records, just set hasMore and break
		if len(operations) >= pageSize {
			hasMore = true
			break
		}

		// Get detailed operation data from opdata table
		var opdataState, opdataScript string
		err := sRuntime.sessionCassa.Query(`
			SELECT state, script
			FROM opdata 
			WHERE txid = ?`,
			txid,
		).Scan(&opdataState, &opdataScript)

		if err != nil {
			if err != gocql.ErrNotFound {
				log.Printf("Error fetching opdata for txid %s: %v", txid, err)
				continue
			}
			continue
		}

		// Parse script to get operation details
		var scriptData map[string]interface{}
		if err := json.Unmarshal([]byte(opdataScript), &scriptData); err != nil {
			log.Printf("Error parsing script JSON for txid %s: %v", txid, err)
			continue
		}

		// Parse state to get fee and tx details
		var stateData map[string]interface{}
		if err := json.Unmarshal([]byte(opdataState), &stateData); err != nil {
			log.Printf("Error parsing state JSON for txid %s: %v", txid, err)
			continue
		}

		// Initialize operation with safe defaults
		op := models.Operation{
			HashRev:  txid,
			OpAccept: opdataState,
			OpError:  "",
			MtsAdd:   strconv.FormatInt(time.Now().UnixMilli(), 10),
			MtsMod:   strconv.FormatInt(time.Now().UnixMilli(), 10),
		}

		// Extract values from stateData
		if fee, ok := stateData["fee"].(float64); ok {
			op.FeeRev = strconv.FormatFloat(fee, 'f', 0, 64)
		}
		if feeLeast, ok := stateData["feeleast"].(float64); ok {
			op.FeeLeast = strconv.FormatFloat(feeLeast, 'f', 0, 64)
		}
		if opAccept, ok := stateData["opaccept"].(float64); ok {
			op.TxAccept = strconv.FormatFloat(opAccept, 'f', 0, 64)
		}

		// Safely extract values from scriptData
		if p, ok := scriptData["p"].(string); ok {
			op.P = p
		}
		if opType, ok := scriptData["op"].(string); ok {
			op.Op = opType
		}
		if tickVal, ok := scriptData["tick"].(string); ok {
			op.Tick = tickVal
		}
		if amt, ok := scriptData["amt"].(string); ok {
			op.Amt = amt
		}
		if from, ok := scriptData["from"].(string); ok {
			op.From = from
		}
		if to, ok := scriptData["to"].(string); ok {
			op.To = to
		}

		op.OpScore = strconv.FormatUint(opScore, 10)
		operations = append(operations, op)
		recordCount++
	}

	if err := iter.Close(); err != nil {
		log.Printf("Error closing iterator: %v", err)
		return nil, false, fmt.Errorf("failed to fetch operations: %v", err)
	}

	log.Printf("Successfully fetched %d operations for tick %s", len(operations), tick)
	return operations, hasMore, nil
}

func GetAllTokens() ([]models.TokenListItem, error) {
	// Query all tokens from the sttoken table
	query := sRuntime.sessionCassa.Query(`
		SELECT tick, meta, minted, opmod, mtsmod 
		FROM sttoken`).PageSize(5000)

	iter := query.Iter()
	var tick, meta, minted string
	var opMod uint64
	var mtsMod int64
	tokens := make([]models.TokenListItem, 0, 5000)

	// Keep iterating until we get all tokens
	for iter.Scan(&tick, &meta, &minted, &opMod, &mtsMod) {
		// Parse meta JSON
		var metaData map[string]interface{}
		if err := json.Unmarshal([]byte(meta), &metaData); err != nil {
			log.Printf("ERROR: Failed to parse meta for tick %s: %v", tick, err)
			continue
		}

		token := models.TokenListItem{
			Tick:       tick,
			Max:        metaData["max"].(string),
			Lim:        metaData["lim"].(string),
			Pre:        metaData["pre"].(string),
			To:         metaData["to"].(string),
			Dec:        int(metaData["dec"].(float64)),
			Minted:     minted,
			OpScoreAdd: uint64(metaData["opadd"].(float64)),
			OpScoreMod: opMod,
			State:      "finished",                // Assuming all tokens in DB are finished
			HashRev:    metaData["txid"].(string), // Using txid as hashRev
			MtsAdd:     int64(metaData["mtsadd"].(float64)),
		}
		tokens = append(tokens, token)
	}

	if err := iter.Close(); err != nil {
		log.Printf("ERROR: Failed to close iterator: %v", err)
		return nil, err
	}

	log.Printf("INFO: Found %d tokens in total", len(tokens))
	return tokens, nil
}

// GetOperationByHash retrieves a single operation by its transaction hash
func GetOperationByHash(hash string) (*models.Operation, error) {
	log.Printf("DEBUG: Fetching operation for hash: %s", hash)

	// Get detailed operation data from opdata table
	var state, script string
	err := sRuntime.sessionCassa.Query(`
		SELECT state, script
		FROM opdata 
		WHERE txid = ?`,
		hash,
	).Scan(&state, &script)

	if err != nil {
		if err == gocql.ErrNotFound {
			log.Printf("DEBUG: No operation found for hash: %s", hash)
			return nil, nil
		}
		log.Printf("ERROR: Failed to fetch from opdata for hash %s: %v", hash, err)
		return nil, err
	}

	// Parse script to get operation details
	var scriptData map[string]interface{}
	if err := json.Unmarshal([]byte(script), &scriptData); err != nil {
		log.Printf("ERROR: Failed to parse script JSON for hash %s: %v", hash, err)
		return nil, fmt.Errorf("failed to parse operation data: %v", err)
	}

	// Parse state to get additional details
	var stateData map[string]interface{}
	if err := json.Unmarshal([]byte(state), &stateData); err != nil {
		log.Printf("ERROR: Failed to parse state JSON for hash %s: %v", hash, err)
		return nil, fmt.Errorf("failed to parse state data: %v", err)
	}

	// Initialize operation with data from state
	operation := &models.Operation{
		HashRev:  hash,
		OpAccept: "", // We'll clear this since we're extracting the data
		OpError:  "", // Will be populated if there's an error in state
		MtsAdd:   strconv.FormatInt(time.Now().UnixMilli(), 10),
		MtsMod:   strconv.FormatInt(time.Now().UnixMilli(), 10),
	}

	// Extract values from stateData
	if fee, ok := stateData["fee"].(float64); ok {
		operation.FeeRev = strconv.FormatFloat(fee, 'f', 0, 64)
	}
	if feeLeast, ok := stateData["feeleast"].(float64); ok {
		operation.FeeLeast = strconv.FormatFloat(feeLeast, 'f', 0, 64)
	}
	if opAccept, ok := stateData["opaccept"].(float64); ok {
		operation.TxAccept = strconv.FormatFloat(opAccept, 'f', 0, 64)
	}
	if blockAccept, ok := stateData["blockaccept"].(string); ok {
		operation.BlockAccept = blockAccept
	}
	if checkpoint, ok := stateData["checkpoint"].(string); ok {
		operation.Checkpoint = checkpoint
	}

	// Safely extract required fields from script
	if p, ok := scriptData["p"].(string); ok {
		operation.P = p
	}
	if op, ok := scriptData["op"].(string); ok {
		operation.Op = op
	}
	if tick, ok := scriptData["tick"].(string); ok {
		operation.Tick = tick
	}
	if amt, ok := scriptData["amt"].(string); ok {
		operation.Amt = amt
	}
	if from, ok := scriptData["from"].(string); ok {
		operation.From = from
	}
	if to, ok := scriptData["to"].(string); ok {
		operation.To = to
	}

	// Try to get additional operation data from oplist, but don't fail if not found
	var opScore uint64
	err = sRuntime.sessionCassa.Query(`
		SELECT opscore
		FROM oplist 
		WHERE txid = ?`,
		hash,
	).Scan(&opScore)

	if err == nil {
		operation.OpScore = strconv.FormatUint(opScore, 10)
	} else if err != gocql.ErrNotFound {
		log.Printf("WARN: Could not fetch opscore for hash %s: %v", hash, err)
	}

	log.Printf("DEBUG: Successfully fetched operation for hash: %s", hash)
	return operation, nil
}
