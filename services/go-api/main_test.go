package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleServiceHealthCheckReturnsExpectedJSONResponse(testingInstance *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	responseRecorder := httptest.NewRecorder()

	handleServiceHealthCheck(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected status %d, got %d", http.StatusOK, responseRecorder.Code)
	}

	expectedContentType := "application/json"
	actualContentType := responseRecorder.Header().Get("Content-Type")
	if actualContentType != expectedContentType {
		testingInstance.Fatalf("expected Content-Type %q, got %q", expectedContentType, actualContentType)
	}

	var responseBody map[string]string
	decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &responseBody)
	if decodeError != nil {
		testingInstance.Fatalf("failed to decode JSON response body: %v", decodeError)
	}

	if responseBody["status"] != "ok" {
		testingInstance.Errorf("expected status %q, got %q", "ok", responseBody["status"])
	}

	if responseBody["service"] != "go-api" {
		testingInstance.Errorf("expected service %q, got %q", "go-api", responseBody["service"])
	}
}
