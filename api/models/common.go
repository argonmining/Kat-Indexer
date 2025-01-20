package models

// Common response structure
type Response struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

// PaginationInfo provides standardized pagination information across all endpoints.
// Some fields are optional depending on the pagination type:
// - For total-based pagination: TotalPages and TotalRecords are used
// - For cursor-based pagination: HasMore is used
type PaginationInfo struct {
	CurrentPage  int  `json:"currentPage,omitempty"`
	PageSize     int  `json:"pageSize"`
	TotalPages   int  `json:"totalPages,omitempty"`
	TotalRecords int  `json:"totalRecords,omitempty"`
	HasMore      bool `json:"hasMore,omitempty"`
}
