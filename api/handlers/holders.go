package handlers

import (
	"kasplex-executor/api/models"
	"kasplex-executor/storage"
	"net/http"
	"strconv"
)

func GetTokenHolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendResponse(w, http.StatusMethodNotAllowed, false, nil, "Method not allowed")
		return
	}

	// Get and validate parameters
	tick := sanitizeString(r.URL.Query().Get("tick"))
	if !validateTick(tick) {
		sendResponse(w, http.StatusBadRequest, false, nil, "Invalid tick parameter")
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 || pageSize > 20000 {
		pageSize = 20000
	}

	// Get holders with pagination
	holders, total, err := storage.GetTokenHoldersPaginated(tick, page, pageSize)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch holders: "+err.Error())
		return
	}

	// Calculate pagination info
	totalPages := (total + pageSize - 1) / pageSize
	paginationInfo := &models.PaginationInfo{
		CurrentPage:  page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: total,
	}

	// Send the holders slice directly as the result
	sendPaginatedResponse(w, http.StatusOK, true, holders, paginationInfo, "")
}
