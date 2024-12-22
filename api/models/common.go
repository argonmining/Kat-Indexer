package models

type PaginationParams struct {
    Limit  int    `json:"limit"`
    Cursor string `json:"cursor,omitempty"`
}

type PaginatedResponse struct {
    Success    bool        `json:"success"`
    Data      interface{} `json:"data,omitempty"`
    Error     string      `json:"error,omitempty"`
    NextCursor string     `json:"next_cursor,omitempty"`
    Total     int64      `json:"total"`
}

type FilterParams struct {
    StartTime int64  `json:"start_time,omitempty"`
    EndTime   int64  `json:"end_time,omitempty"`
    SortBy    string `json:"sort_by,omitempty"`
    SortDir   string `json:"sort_dir,omitempty"`
}

type MarketOrder struct {
    Tick      string `json:"tick"`
    TxID      string `json:"txid"`
    Address   string `json:"address"`
    Amount    int64  `json:"amount"`
    Price     int64  `json:"price"`
    Timestamp int64  `json:"timestamp"`
}

type MarketStats struct {
    Tick           string  `json:"tick"`
    Price24h      int64   `json:"price_24h"`
    Volume24h     int64   `json:"volume_24h"`
    PriceChange24h float64 `json:"price_change_24h"`
}
