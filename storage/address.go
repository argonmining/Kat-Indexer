package storage

import (
	"fmt"
	"kasplex-executor/api/models"
	"sort"

	"github.com/gocql/gocql"
)

func GetAddressBalances(address string) ([]*models.AddressBalance, error) {
	// Get all token balances for a specific address
	balances := make([]*models.AddressBalance, 0)

	// Query Cassandra for balances with the given address
	iter := sRuntime.sessionCassa.Query("SELECT tick, dec, balance, locked FROM stbalance WHERE address = ?", address).Iter()

	var tick string
	var dec int
	var balance, locked string

	for iter.Scan(&tick, &dec, &balance, &locked) {
		balances = append(balances, &models.AddressBalance{
			Tick:    tick,
			Balance: parseStringToUint64(balance),
			Locked:  parseStringToUint64(locked),
			Dec:     dec,
		})
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return balances, nil
}

func GetTopHoldersByTokenCount(page, pageSize int) ([]HolderPortfolio, int, error) {
	// Use the balance index directly
	query := sRuntime.sessionCassa.Query(`
		SELECT address, tick, dec, balance, locked 
		FROM stbalance 
		WHERE balance >= '1'
		ALLOW FILTERING`).PageSize(2000)

	// Track unique addresses and their holdings
	addressMap := make(map[string]*HolderPortfolio)

	var address, tick string
	var dec int
	var balance, locked string

	iter := query.Iter()
	for iter.Scan(&address, &tick, &dec, &balance, &locked) {
		bal := parseStringToUint64(balance)
		lock := parseStringToUint64(locked)

		if bal == 0 && lock == 0 {
			continue
		}

		portfolio, exists := addressMap[address]
		if !exists {
			portfolio = &HolderPortfolio{
				Address:    address,
				TokenCount: 0,
				Holdings:   make([]PortfolioHolding, 0, 10),
				TotalValue: 0,
			}
			addressMap[address] = portfolio
		}

		// Add the holding
		portfolio.Holdings = append(portfolio.Holdings, PortfolioHolding{
			Tick:    tick,
			Balance: bal,
			Locked:  lock,
			Dec:     dec,
		})
		portfolio.TotalValue += bal + lock
	}

	if err := iter.Close(); err != nil {
		return nil, 0, err
	}

	// Convert map to slice and calculate token counts
	portfolios := make([]HolderPortfolio, 0, len(addressMap))
	for _, portfolio := range addressMap {
		// Count unique tokens
		tokens := make(map[string]bool)
		for _, holding := range portfolio.Holdings {
			tokens[holding.Tick] = true
		}
		portfolio.TokenCount = len(tokens)
		portfolios = append(portfolios, *portfolio)
	}

	// Sort by token count first, then by total value
	sort.Slice(portfolios, func(i, j int) bool {
		if portfolios[i].TokenCount != portfolios[j].TokenCount {
			return portfolios[i].TokenCount > portfolios[j].TokenCount
		}
		return portfolios[i].TotalValue > portfolios[j].TotalValue
	})

	// Handle pagination
	total := len(portfolios)
	start := (page - 1) * pageSize
	if start >= total {
		return []HolderPortfolio{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return portfolios[start:end], total, nil
}

// GetAllAddressesPaginated returns a paginated list of all addresses with their balances
func GetAllAddressesPaginated(lastAddress string, pageSize int) ([]models.AddressPortfolio, bool, error) {
	var query *gocql.Query
	if lastAddress == "" {
		// First page query - get the first batch of addresses
		query = sRuntime.sessionCassa.Query(`
			SELECT DISTINCT address 
			FROM stbalance 
			LIMIT ?`,
			pageSize+1,
		)
	} else {
		// Query for the next page using the provided cursor
		query = sRuntime.sessionCassa.Query(`
			SELECT DISTINCT address 
			FROM stbalance 
			WHERE address > ?
			LIMIT ?`,
			lastAddress,
			pageSize+1,
		)
	}

	// Set query options
	query = query.PageSize(pageSize + 1)

	// First get the list of addresses
	iter := query.Iter()
	var address string
	addresses := make([]string, 0, pageSize)
	hasMore := false
	count := 0

	for iter.Scan(&address) {
		if count >= pageSize {
			hasMore = true
			break
		}
		addresses = append(addresses, address)
		count++
	}

	if err := iter.Close(); err != nil {
		return nil, false, fmt.Errorf("error closing iterator: %v", err)
	}

	// Now get balances for each address
	result := make([]models.AddressPortfolio, 0, len(addresses))
	for _, addr := range addresses {
		// Get balances for this address
		balances, err := GetAddressBalances(addr)
		if err != nil {
			continue
		}

		// Only include addresses that have non-zero balances
		hasNonZeroBalance := false
		for _, bal := range balances {
			if bal.Balance > 0 || bal.Locked > 0 {
				hasNonZeroBalance = true
				break
			}
		}

		if hasNonZeroBalance {
			portfolio := models.AddressPortfolio{
				Address:  addr,
				Balances: balances,
			}
			result = append(result, portfolio)
		}
	}

	return result, hasMore, nil
}
