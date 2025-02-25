package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"kasplex-executor/api/models"
	"kasplex-executor/storage"
)

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

// GetAllTransactions returns all transactions with pagination support
func GetAllTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendResponse(w, http.StatusMethodNotAllowed, false, nil, "Method not allowed")
		return
	}

	// Parse pagination parameters
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 {
		pageSize = 1000
	} else if pageSize > 5000 {
		pageSize = 5000
	}

	// Parse lastScore if provided
	var lastScore *uint64
	if lastScoreStr := r.URL.Query().Get("lastScore"); lastScoreStr != "" {
		if score, err := strconv.ParseUint(lastScoreStr, 10, 64); err == nil {
			lastScore = &score
		}
	}

	// Get all operations with pagination
	operations, hasMore, err := storage.GetAllOperationsPaginated(lastScore, pageSize)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch transactions: "+err.Error())
		return
	}

	// Create pagination info with requested pageSize
	paginationInfo := &models.PaginationInfo{
		PageSize: pageSize,
		HasMore:  hasMore,
	}

	sendPaginatedResponse(w, http.StatusOK, true, operations, paginationInfo, "")
}
