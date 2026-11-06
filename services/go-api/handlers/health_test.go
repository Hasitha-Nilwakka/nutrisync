package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateServiceHealthCheckResponseReturnsExpectedStruct(testingInstance *testing.T) {
	healthCheckResponse := CreateServiceHealthCheckResponse()

	if healthCheckResponse.Status != "ok" {
		testingInstance.Errorf("expected Status %q, got %q", "ok", healthCheckResponse.Status)
	}

	if healthCheckResponse.Service != "go-api" {
		testingInstance.Errorf("expected Service %q, got %q", "go-api", healthCheckResponse.Service)
	}
}

func TestWriteJSONResponseWritesExpectedStatusHeaderAndPayload(testingInstance *testing.T) {
	responseRecorder := httptest.NewRecorder()
	payload := CreateServiceHealthCheckResponse()

	WriteJSONResponse(responseRecorder, http.StatusOK, payload)

	if responseRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected status %d, got %d", http.StatusOK, responseRecorder.Code)
	}

	if responseRecorder.Header().Get("Content-Type") != "application/json" {
		testingInstance.Fatalf("expected Content-Type %q, got %q", "application/json", responseRecorder.Header().Get("Content-Type"))
	}

	var decodedBody ServiceHealthCheckResponse
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &decodedBody); decodeError != nil {
		testingInstance.Fatalf("failed to decode response body: %v", decodeError)
	}

	if decodedBody != payload {
		testingInstance.Errorf("expected body %+v, got %+v", payload, decodedBody)
	}
}

func TestHandleServiceHealthCheckWritesExpectedHealthCheckResponse(testingInstance *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	responseRecorder := httptest.NewRecorder()

	HandleServiceHealthCheck(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected status %d, got %d", http.StatusOK, responseRecorder.Code)
	}

	var decodedBody ServiceHealthCheckResponse
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &decodedBody); decodeError != nil {
		testingInstance.Fatalf("failed to decode response body: %v", decodeError)
	}

	if decodedBody.Status != "ok" || decodedBody.Service != "go-api" {
		testingInstance.Errorf("unexpected response body: %+v", decodedBody)
	}
}
