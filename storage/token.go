package storage

import (
	"encoding/json"
	"kasplex-executor/api/models"
	"log"
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
	var totalSupply uint64

	// First pass: collect all non-zero balances and calculate total supply
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
			totalSupply += total
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

	// Calculate ranks and shares for the page
	for i := start; i < end; i++ {
		total := holders[i].Balance + holders[i].Locked
		holders[i].Rank = i + 1
		holders[i].Share = float64(total) / float64(totalSupply) * 100
	}

	return holders[start:end], total, nil
}

func GetTokenOperationsPaginated(tick string, offset, pageSize int) ([]models.Operation, bool, error) {
	log.Printf("Fetching operations for tick: %s, offset: %d, pageSize: %d", tick, offset, pageSize)

	// Query using the index on tickaffc
	query := sRuntime.sessionCassa.Query(`
		SELECT oprange, opscore, txid, state, script, tickaffc, addressaffc 
		FROM oplist 
		WHERE tickaffc >= ? AND tickaffc < ?
		LIMIT ?
		ALLOW FILTERING`,
		tick+"=",   // Start of range
		tick+"=~",  // End of range (~ comes after any digit in ASCII)
		pageSize+1, // Get one extra record to determine if there are more pages
	).PageSize(pageSize + 1)

	iter := query.Iter()
	var oprange int64
	var opScore uint64
	var txid, state, script, tickAffc, addressaffc string
	operations := make([]models.Operation, 0, pageSize)
	hasMore := false

	for iter.Scan(&oprange, &opScore, &txid, &state, &script, &tickAffc, &addressaffc) {
		// If we've got more than pageSize records, just set hasMore and break
		if len(operations) >= pageSize {
			hasMore = true
			break
		}

		// Get detailed operation data from opdata table
		var opdataState, opdataScript, stBefore, stAfter string
		err := sRuntime.sessionCassa.Query(`
			SELECT state, script, stbefore, stafter 
			FROM opdata 
			WHERE txid = ?`,
			txid,
		).Scan(&opdataState, &opdataScript, &stBefore, &stAfter)

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

		op := models.Operation{
			P:          scriptData["p"].(string),
			Op:         scriptData["op"].(string),
			Tick:       scriptData["tick"].(string),
			Amt:        scriptData["amt"].(string),
			From:       scriptData["from"].(string),
			To:         scriptData["to"].(string),
			OpScore:    strconv.FormatUint(opScore, 10),
			HashRev:    txid,
			FeeRev:     "0",
			TxAccept:   "1",
			OpAccept:   opdataState,
			OpError:    "",
			Checkpoint: "",
			MtsAdd:     strconv.FormatInt(time.Now().UnixMilli(), 10),
			MtsMod:     strconv.FormatInt(time.Now().UnixMilli(), 10),
		}

		operations = append(operations, op)
	}

	if err := iter.Close(); err != nil {
		log.Printf("Error closing iterator: %v", err)
		return nil, false, err
	}

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
