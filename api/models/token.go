package models

type TokenBalance struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
	Locked  uint64 `json:"locked"`
	Dec     int    `json:"decimals"`
}

type TokenInfo struct {
	Tick   string `json:"tick"`
	Meta   string `json:"meta"`
	Minted uint64 `json:"minted"`
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
	Balance uint64  `json:"balance"`
	Locked  uint64  `json:"locked"`
	Share   float64 `json:"share"` // Percentage of total supply
}

// SnapshotSummary provides overview statistics
type SnapshotSummary struct {
	TotalSupply       uint64 `json:"totalSupply"`
	HoldersCount      int    `json:"holdersCount"`
	LockedTokens      uint64 `json:"lockedTokens"`
	CirculatingSupply uint64 `json:"circulatingSupply"`
}
