package models

type TokenBalance struct {
	Address string `json:"address"`
	Balance int64  `json:"balance"`
	Locked  int64  `json:"locked"`
	Dec     int    `json:"decimals"`
}

type TokenInfo struct {
	Tick   string `json:"tick"`
	Meta   string `json:"meta"`
	Minted int64  `json:"minted"`
	OpMod  int64  `json:"op_mod"`
	MtsMod int64  `json:"mts_mod"`
}

type TokenResponse struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// TokenSnapshot represents a complete snapshot of token holders
type TokenSnapshot struct {
	Tick      string          `json:"tick"`
	Timestamp int64           `json:"timestamp"`
	Holders   []TokenHolder   `json:"holders"`
	Summary   SnapshotSummary `json:"summary"`
}

// TokenHolder represents a single holder's balance
type TokenHolder struct {
	Address string  `json:"address"`
	Balance int64   `json:"balance"`
	Locked  int64   `json:"locked"`
	Share   float64 `json:"share"` // Percentage of total supply
}

// SnapshotSummary provides overview statistics
type SnapshotSummary struct {
	TotalSupply       int64 `json:"totalSupply"`
	HoldersCount      int   `json:"holdersCount"`
	LockedTokens      int64 `json:"lockedTokens"`
	CirculatingSupply int64 `json:"circulatingSupply"`
}
