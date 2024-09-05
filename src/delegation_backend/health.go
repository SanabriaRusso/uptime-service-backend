package delegation_backend

import (
	"encoding/json"
	"net/http"
)

// HealthStatus represents the JSON response structure for the /health endpoint
type HealthStatus struct {
	Status string `json:"status"`
}

// HealthHandler handles the /health endpoint, checking if the application is ready.
func HealthHandler(isReady func() bool) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if isReady() {
			rw.WriteHeader(http.StatusOK)
			json.NewEncoder(rw).Encode(HealthStatus{Status: "ok"})
		} else {
			rw.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(rw).Encode(HealthStatus{Status: "unavailable"})
		}
	}
}
