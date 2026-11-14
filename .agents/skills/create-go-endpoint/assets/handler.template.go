// Package handlers implements HTTP request handlers for the NutriSync Go API service.
// Replace <DomainAction> and payload fields with your specific endpoint logic.
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

// DomainActionRequestPayload defines the expected JSON body for POST /api/health/<resource>.
type DomainActionRequestPayload struct {
	RecordedValue float64 `json:"recordedValue"`
	Unit          string  `json:"unit"`
}

// DomainActionResponsePayload defines the JSON response returned on success.
type DomainActionResponsePayload struct {
	ID              string  `json:"id"`
	NormalizedValue float64 `json:"normalizedValue"`
}

// HandleDomainActionRequest validates the session cookie, normalizes input bounds,
// persists the user-scoped record, and writes the JSON response.
func (h *HealthProfileHandler) HandleDomainActionRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	authenticatedUserID, authErr := h.authenticateRequestUserID(r)
	if authErr != nil || strings.TrimSpace(authenticatedUserID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
		return
	}

	var requestPayload DomainActionRequestPayload
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decodeErr := decoder.Decode(&requestPayload); decodeErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON request payload"})
		return
	}

	if requestPayload.RecordedValue <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "recordedValue must be greater than zero"})
		return
	}

	// Invoke user-scoped repository method using authenticatedUserID...
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(DomainActionResponsePayload{
		ID:              "generated-record-id",
		NormalizedValue: requestPayload.RecordedValue,
	})
}
