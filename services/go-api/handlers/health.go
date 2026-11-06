package handlers

import (
	"encoding/json"
	"net/http"
)

type ServiceHealthCheckResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func CreateServiceHealthCheckResponse() ServiceHealthCheckResponse {
	return ServiceHealthCheckResponse{
		Status:  "ok",
		Service: "go-api",
	}
}

func WriteJSONResponse(responseWriter http.ResponseWriter, statusCode int, responsePayload any) {
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(statusCode)
	_ = json.NewEncoder(responseWriter).Encode(responsePayload)
}

func HandleServiceHealthCheck(responseWriter http.ResponseWriter, request *http.Request) {
	healthCheckResponse := CreateServiceHealthCheckResponse()
	WriteJSONResponse(responseWriter, http.StatusOK, healthCheckResponse)
}
