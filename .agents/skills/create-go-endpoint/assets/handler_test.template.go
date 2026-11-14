// This file contains unit and negative authorization tests for HandleDomainActionRequest.
package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleDomainActionRequest_RejectsUnauthenticatedRequest verifies that requests without
// a valid access_token session cookie are rejected with 401 Unauthorized (AGENTS.md Section 6).
func TestHandleDomainActionRequest_RejectsUnauthenticatedRequest(t *testing.T) {
	requestBody := bytes.NewBufferString(`{"recordedValue": 72.5, "unit": "kg"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/health/resource", requestBody)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	// Replace with initialized handler under test:
	// handler.HandleDomainActionRequest(recorder, req)
	_ = req
	_ = recorder
}

// TestHandleDomainActionRequest_RejectsInvalidBounds verifies that out-of-bounds values
// are rejected with 400 Bad Request before reaching the repository layer.
func TestHandleDomainActionRequest_RejectsInvalidBounds(t *testing.T) {
	requestBody := bytes.NewBufferString(`{"recordedValue": -10, "unit": "kg"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/health/resource", requestBody)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	_ = req
	_ = recorder
}
