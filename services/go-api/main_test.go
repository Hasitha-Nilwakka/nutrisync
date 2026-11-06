// This file contains unit tests for handleServiceHealthCheck and CreateApplicationMux in package main.
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nutrysync/go-api/handlers"
	"nutrysync/go-api/models"
)

type stubMainSessionRepository struct{}

func (stubMainSessionRepository) FindUserByEmailForLogin(ctx context.Context, email string) (*models.User, error) {
	return nil, nil
}
func (stubMainSessionRepository) CreateSessionWithInitialRefreshToken(ctx context.Context, session *models.Session, initialRefreshToken *models.RefreshToken) error {
	return nil
}
func (stubMainSessionRepository) FindRefreshTokenWithSessionByID(ctx context.Context, refreshTokenID string) (*models.RefreshToken, error) {
	return nil, nil
}
func (stubMainSessionRepository) RotateRefreshTokenInTransaction(ctx context.Context, currentTokenID string, newRefreshToken *models.RefreshToken) error {
	return nil
}
func (stubMainSessionRepository) RevokeSessionAndAllRefreshTokens(ctx context.Context, sessionID string) error {
	return nil
}
func (stubMainSessionRepository) FindSessionByID(ctx context.Context, sessionID string) (*models.Session, error) {
	return nil, nil
}

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

func TestCreateApplicationMuxRegistersHealthRegistrationAndSessionRoutes(testingInstance *testing.T) {
	registrationHandler := handlers.NewUserRegistrationHandler(nil, nil)
	sessionHandler, err := handlers.NewUserSessionHandler(stubMainSessionRepository{}, "test-secret", false)
	if err != nil {
		testingInstance.Fatalf("failed to initialize sessionHandler: %v", err)
	}

	mux := CreateApplicationMux(registrationHandler, sessionHandler)

	routesToCheck := []struct {
		method         string
		path           string
		expectedStatus int
	}{
		{method: http.MethodGet, path: "/health", expectedStatus: http.StatusOK},
		{method: http.MethodGet, path: "/api/auth/verify", expectedStatus: http.StatusBadRequest},
		{method: http.MethodPost, path: "/api/auth/logout", expectedStatus: http.StatusOK},
		{method: http.MethodGet, path: "/api/auth/session", expectedStatus: http.StatusUnauthorized},
	}

	for _, route := range routesToCheck {
		request := httptest.NewRequest(route.method, route.path, nil)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)

		if recorder.Code != route.expectedStatus {
			testingInstance.Errorf("expected route %s %s to return %d, got %d", route.method, route.path, route.expectedStatus, recorder.Code)
		}
	}
}
