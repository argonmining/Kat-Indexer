package handlers

import (
	"kasplex-executor/api/models"
	"kasplex-executor/storage"
	"net/http"
	"strconv"
)

func GetAddressBalances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendResponse(w, http.StatusMethodNotAllowed, false, nil, "Method not allowed")
		return
	}

	address := r.URL.Query().Get("address")
	if address == "" {
		sendResponse(w, http.StatusBadRequest, false, nil, "Address parameter is required")
		return
	}

	// Query storage for address balances
	balances, err := storage.GetAddressBalances(address)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch address balances")
		return
	}

	sendResponse(w, http.StatusOK, true, balances, "")
}

func GetTopHolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendResponse(w, http.StatusMethodNotAllowed, false, nil, "Method not allowed")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 || pageSize > 2000 {
		pageSize = 2000
	}

	holders, total, err := storage.GetTopHoldersByTokenCount(page, pageSize)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch top holders: "+err.Error())
		return
	}

	// Convert storage types to model types
	modelHolders := make([]models.HolderPortfolio, len(holders))
	for i, holder := range holders {
		modelHoldings := make([]models.PortfolioHolding, len(holder.Holdings))
		for j, holding := range holder.Holdings {
			modelHoldings[j] = models.PortfolioHolding{
				Tick:    holding.Tick,
				Balance: holding.Balance,
				Locked:  holding.Locked,
				Dec:     holding.Dec,
			}
		}

		modelHolders[i] = models.HolderPortfolio{
			Address:    holder.Address,
			TokenCount: holder.TokenCount,
			TotalValue: holder.TotalValue,
			Holdings:   modelHoldings,
		}
	}

	totalPages := (total + pageSize - 1) / pageSize
	paginationInfo := &models.PaginationInfo{
		CurrentPage:  page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: total,
	}

	sendPaginatedResponse(w, http.StatusOK, true, modelHolders, paginationInfo, "")
}

// GetAllAddresses returns all addresses with their balances with pagination support
func GetAllAddresses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendResponse(w, http.StatusMethodNotAllowed, false, nil, "Method not allowed")
		return
	}

	// Parse pagination parameters
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 || pageSize > 5000 {
		pageSize = 100
	}

	// Parse lastAddress if provided for cursor-based pagination
	lastAddress := r.URL.Query().Get("lastAddress")

	// Get all addresses with pagination
	addresses, hasMore, err := storage.GetAllAddressesPaginated(lastAddress, pageSize)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, false, nil, "Failed to fetch addresses: "+err.Error())
		return
	}

	// Create pagination info
	paginationInfo := &models.PaginationInfo{
		PageSize: pageSize,
		HasMore:  hasMore,
	}

	sendPaginatedResponse(w, http.StatusOK, true, addresses, paginationInfo, "")
}
