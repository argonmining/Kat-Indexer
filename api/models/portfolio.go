package models

type PortfolioHolding struct {
	Tick    string `json:"tick"`
	Balance uint64 `json:"balance"`
	Locked  uint64 `json:"locked"`
	Dec     int    `json:"decimals"`
}

type HolderPortfolio struct {
	Address    string             `json:"address"`
	TokenCount int                `json:"tokenCount"`
	TotalValue uint64             `json:"totalValue"`
	Holdings   []PortfolioHolding `json:"holdings"`
}

type TopHoldersResponse struct {
	Holders    []HolderPortfolio `json:"holders"`
	Pagination PaginationInfo    `json:"pagination"`
}
