package storage

import (
	"kasplex-executor/api/models"
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
			Balance: parseStringToInt64(balance),
			Locked:  parseStringToInt64(locked),
			Dec:     dec,
		})
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return balances, nil
}
