package models

type AddressBalance struct {
	Tick    string `json:"tick"`
	Balance uint64 `json:"balance"`
	Locked  uint64 `json:"locked"`
	Dec     int    `json:"decimals"`
}

// AddressPortfolio represents an address and all its token balances
type AddressPortfolio struct {
	Address  string            `json:"address"`
	Balances []*AddressBalance `json:"balances"`
}
