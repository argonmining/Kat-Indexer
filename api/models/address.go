package models

type AddressBalance struct {
	Tick    string `json:"tick"`
	Balance uint64 `json:"balance"`
	Locked  uint64 `json:"locked"`
	Dec     int    `json:"decimals"`
}

type AddressResponse struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
