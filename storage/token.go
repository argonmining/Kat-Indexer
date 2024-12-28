package storage

import (
	"encoding/json"
	"kasplex-executor/api/models"
	"log"
	"sort"
	"strings"

	"github.com/gocql/gocql"
)

func GetTokenBalances(tick string) ([]*models.TokenBalance, error) {
	// Pre-allocate slice with a reasonable size
	balances := make([]*models.TokenBalance, 0, 20000)

	// Update page size to 2000
	query := sRuntime.sessionCassa.Query(`
		SELECT address, tick, dec, balance, locked 
		FROM stbalance 
		WHERE tick = ?`,
		tick,
	).PageSize(2000)

	iter := query.Iter()

	var address, balance, locked string
	var dec int
	var tickResult string

	for iter.Scan(&address, &tickResult, &dec, &balance, &locked) {
		balances = append(balances, &models.TokenBalance{
			Address: address,
			Balance: parseStringToUint64(balance),
			Locked:  parseStringToUint64(locked),
			Dec:     dec,
		})
	}

	if err := iter.Close(); err != nil {
		log.Printf("ERROR: Failed to close iterator for tick %s: %v", tick, err)
		return nil, err
	}

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

func GetTokenOperationsPaginated(tick string, page, pageSize int) ([]models.Operation, int, error) {
	// First, get the current oprange
	var currentOpRange int64
	err := sRuntime.sessionCassa.Query(`
		SELECT oprange 
		FROM oplist 
		WHERE tickaffc = ? 
		LIMIT 1 
		ALLOW FILTERING`,
		tick,
	).Scan(&currentOpRange)
	if err != nil {
		if err != gocql.ErrNotFound {
			return nil, 0, err
		}
		log.Printf("No operations found for tick %s", tick)
		return []models.Operation{}, 0, nil
	}

	log.Printf("Found currentOpRange: %d for tick %s", currentOpRange, tick)

	// Calculate the range of opranges to query (e.g., last 10 ranges)
	rangeStart := currentOpRange - 10
	if rangeStart < 0 {
		rangeStart = 0
	}

	log.Printf("Querying operations between ranges %d and %d", rangeStart, currentOpRange)

	// Query operations within these ranges
	query := sRuntime.sessionCassa.Query(`
		SELECT oprange, opscore, txid, state, script, tickaffc 
		FROM oplist 
		WHERE oprange >= ? AND oprange <= ? AND tickaffc = ?
		ALLOW FILTERING`,
		rangeStart, currentOpRange, tick,
	).PageSize(2000)

	iter := query.Iter()
	var opRange, opScore int64
	var txid, state, script, tickAffc string
	operations := make([]models.Operation, 0, 2000)
	operationCount := 0

	for iter.Scan(&opRange, &opScore, &txid, &state, &script, &tickAffc) {
		operationCount++
		// Get detailed operation data from opdata table
		var stBefore, stAfter string
		err := sRuntime.sessionCassa.Query(`
			SELECT stbefore, stafter 
			FROM opdata 
			WHERE txid = ?`,
			txid,
		).Scan(&stBefore, &stAfter)

		if err != nil {
			if err != gocql.ErrNotFound {
				log.Printf("Error fetching opdata for txid %s: %v", txid, err)
				continue
			}
			log.Printf("No opdata found for txid %s", txid)
			continue
		}

		op := parseOperationDetails(script, txid, state, opScore, stBefore, stAfter)
		if op != nil {
			operations = append(operations, *op)
		}
	}

	if err := iter.Close(); err != nil {
		return nil, 0, err
	}

	log.Printf("Found %d raw operations, parsed into %d valid operations",
		operationCount, len(operations))

	// Sort operations by timestamp (opScore) in descending order
	sort.Slice(operations, func(i, j int) bool {
		return operations[i].Timestamp > operations[j].Timestamp
	})

	// Calculate pagination
	total := len(operations)
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	if start >= total {
		return []models.Operation{}, total, nil
	}

	return operations[start:end], total, nil
}

func parseOperationDetails(script, txid, state string, timestamp int64, stBefore, stAfter string) *models.Operation {
	parts := strings.Split(script, "|")
	if len(parts) < 3 {
		return nil
	}

	// Parse state changes to determine from/to addresses and amount
	var from, to string
	var amount uint64

	// Extract operation details from state changes
	if strings.Contains(stBefore, KeyPrefixStateBalance) {
		beforeParts := strings.Split(stBefore, ",")
		afterParts := strings.Split(stAfter, ",")

		// Parse the state changes to extract addresses and amount
		for i := 0; i < len(beforeParts); i++ {
			if strings.HasPrefix(beforeParts[i], KeyPrefixStateBalance) {
				balanceParts := strings.Split(beforeParts[i], "_")
				if len(balanceParts) >= 2 {
					from = balanceParts[1]
					// Compare before and after balances to determine amount
					if i < len(afterParts) {
						beforeBal := parseStringToUint64(beforeParts[i])
						afterBal := parseStringToUint64(afterParts[i])
						if beforeBal > afterBal {
							amount = beforeBal - afterBal
						}
					}
				}
				break
			}
		}

		// Find the recipient address
		for i := 0; i < len(afterParts); i++ {
			if strings.HasPrefix(afterParts[i], KeyPrefixStateBalance) && !strings.Contains(afterParts[i], from) {
				balanceParts := strings.Split(afterParts[i], "_")
				if len(balanceParts) >= 2 {
					to = balanceParts[1]
					break
				}
			}
		}
	}

	return &models.Operation{
		TxID:      txid,
		Type:      parts[0], // operation type from script
		From:      from,
		To:        to,
		Amount:    amount,
		Timestamp: timestamp,
		State:     state,
	}
}
