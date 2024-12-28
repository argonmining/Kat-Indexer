package models

type OperationsResponse struct {
	Operations []Operation    `json:"operations"`
	Pagination PaginationInfo `json:"pagination"`
}

type Operation struct {
	TxID      string `json:"txId"`
	Type      string `json:"type"` // e.g., "transfer", "mint", "burn"
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	Timestamp int64  `json:"timestamp"`
	State     string `json:"state"` // success/failed
}
