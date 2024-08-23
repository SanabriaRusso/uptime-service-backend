package delegation_backend

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock App structure with IsReady flag
type MockApp struct {
	IsReady bool
}

// TestHealthEndpointBeforeReady tests the /health endpoint before the application is ready.
func TestHealthEndpointBeforeReady(t *testing.T) {
	app := &MockApp{IsReady: false}

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := HealthHandler(func() bool { return app.IsReady })

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
	}

}

// TestHealthEndpointAfterReady tests the /health endpoint after the application is ready.
func TestHealthEndpointAfterReady(t *testing.T) {
	app := &MockApp{IsReady: true}

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := HealthHandler(func() bool { return app.IsReady })

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

}

// TestHealthEndpointTransition tests the /health endpoint during the transition from not ready to ready.
func TestHealthEndpointTransition(t *testing.T) {
	app := &MockApp{IsReady: false}

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := HealthHandler(func() bool { return app.IsReady })

	// Initially not ready
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
	}

	// Simulate the application becoming ready
	app.IsReady = true
	rr = httptest.NewRecorder() // Reset the recorder
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

}
