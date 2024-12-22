package models

type AddressBalance struct {
    Tick    string `json:"tick"`
    Balance int64  `json:"balance"`
    Locked  int64  `json:"locked"`
    Dec     int    `json:"decimals"`
}

type AddressResponse struct {
    Success bool        `json:"success"`
    Error   string     `json:"error,omitempty"`
    Data    interface{} `json:"data,omitempty"`
}