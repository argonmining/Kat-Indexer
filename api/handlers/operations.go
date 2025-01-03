package handlers

import (
	"encoding/json"
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

	// Process each operation to extract data from opAccept
	for i := range operations {
		if operations[i].OpAccept != "" {
			var opAcceptData map[string]interface{}
			if err := json.Unmarshal([]byte(operations[i].OpAccept), &opAcceptData); err != nil {
				continue
			}

			// Extract values from opAcceptData
			if blockAccept, ok := opAcceptData["blockaccept"].(string); ok {
				operations[i].BlockAccept = blockAccept
			}
			if feeLeast, ok := opAcceptData["feeleast"].(float64); ok {
				operations[i].FeeLeast = strconv.FormatFloat(feeLeast, 'f', 0, 64)
			}
			if checkpoint, ok := opAcceptData["checkpoint"].(string); ok {
				operations[i].Checkpoint = checkpoint
			}

			// Clear the opAccept field as we've extracted all needed data
			operations[i].OpAccept = ""
		}
	}

	// Create pagination info
	paginationInfo := &models.PaginationInfo{
		CurrentPage: page,
		PageSize:    pageSize,
		HasMore:     hasMore,
	}

	sendPaginatedResponse(w, http.StatusOK, true, operations, paginationInfo, "")
}

// GetTransaction returns details of a single operation by its transaction hash
func GetTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendResponse(w, http.StatusMethodNotAllowed, false, nil, "Method not allowed")
		return
	}

	// Get and validate the hash parameter
	hash := r.URL.Query().Get("hash")
	if hash == "" {
		sendResponse(w, http.StatusBadRequest, false, nil, "Hash parameter is required")
		return
	}

	// Get operation details from storage
	operation, err := storage.GetOperationByHash(hash)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch transaction: "+err.Error())
		return
	}

	if operation == nil {
		sendResponse(w, http.StatusNotFound, false, nil, "Transaction not found")
		return
	}

	// Process operation to extract data from opAccept
	if operation.OpAccept != "" {
		var opAcceptData map[string]interface{}
		if err := json.Unmarshal([]byte(operation.OpAccept), &opAcceptData); err == nil {
			// Extract values from opAcceptData
			if blockAccept, ok := opAcceptData["blockaccept"].(string); ok {
				operation.BlockAccept = blockAccept
			}
			if feeLeast, ok := opAcceptData["feeleast"].(float64); ok {
				operation.FeeLeast = strconv.FormatFloat(feeLeast, 'f', 0, 64)
			}
			if checkpoint, ok := opAcceptData["checkpoint"].(string); ok {
				operation.Checkpoint = checkpoint
			}

			// Clear the opAccept field as we've extracted all needed data
			operation.OpAccept = ""
		}
	}

	sendResponse(w, http.StatusOK, true, operation, "")
}
