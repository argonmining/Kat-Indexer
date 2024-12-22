package models

type HoldersResponse struct {
	Holders    []HolderInfo   `json:"holders"`
	Pagination PaginationInfo `json:"pagination"`
}

type HolderInfo struct {
	Address string  `json:"address"`
	Balance int64   `json:"balance"`
	Locked  int64   `json:"locked"`
	Share   float64 `json:"share"`
	Rank    int     `json:"rank"`
}

type PaginationInfo struct {
	CurrentPage  int `json:"currentPage"`
	PageSize     int `json:"pageSize"`
	TotalPages   int `json:"totalPages"`
	TotalRecords int `json:"totalRecords"`
}
