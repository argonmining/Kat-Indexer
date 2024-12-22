package handlers

import (
	"encoding/json"
	"kasplex-executor/api/models"
	"net/http"
	"strings"
)

func sendResponse(w http.ResponseWriter, status int, success bool, data interface{}, errMsg string) {
	response := models.TokenResponse{
		Success: success,
		Error:   errMsg,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// validateTick ensures the ticker symbol is valid:
// - 4-6 characters
// - Only uppercase alphabetical characters (A-Z)
func validateTick(tick string) bool {
	// Check length (4-6 characters)
	length := len(tick)
	if length < 4 || length > 6 {
		return false
	}

	// Check characters (only A-Z)
	for _, r := range tick {
		if r < 'A' || r > 'Z' {
			return false
		}
	}

	return true
}

// Helper function to sanitize input strings
func sanitizeString(s string) string {
	return strings.TrimSpace(strings.ToUpper(s))
}
