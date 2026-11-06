package main

import (
	"encoding/json"
	"log"
	"net/http"
	"nutrysync/go-api/handlers"
)

func handleServiceHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	healthCheckResponse := handlers.ServiceHealthCheckResponse{
		Status:  "ok",
		Service: "go-api",
	}

	_ = json.NewEncoder(w).Encode(healthCheckResponse)
}

func main() {
	http.HandleFunc("/health", handleServiceHealthCheck)

	log.Println("Server is running on http://localhost:8080")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
