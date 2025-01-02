package handlers

import (
	"net/http"
	"strconv"

	"kasplex-executor/api/models"
	"kasplex-executor/storage"
)

func GetTokenOperations(w http.ResponseWriter, r *http.Request) {
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
	if pageSize < 1 || pageSize > 2000 {
		pageSize = 2000
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get operations with pagination
	operations, hasMore, err := storage.GetTokenOperationsPaginated(tick, offset, pageSize)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch operations: "+err.Error())
		return
	}

	response := models.OperationsResponse{
		Message: "successful",
		HasMore: hasMore,
		Result:  operations,
	}

	sendResponse(w, http.StatusOK, true, response, "")
}
