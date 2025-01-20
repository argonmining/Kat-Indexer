package models

type HoldersResponse struct {
	Holders    []HolderInfo   `json:"holders"`
	Pagination PaginationInfo `json:"pagination"`
}

type HolderInfo struct {
	Address string  `json:"address"`
	Balance uint64  `json:"balance"`
	Locked  uint64  `json:"locked"`
	Share   float64 `json:"share"`
	Rank    int     `json:"rank"`
}
