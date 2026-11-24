package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check response is valid JSON with status ok
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v, body: %s", err, w.Body.String())
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status=ok, got %v", resp["status"])
	}
	if couchdb, ok := resp["couchdb"].(map[string]interface{}); !ok || couchdb["status"] != "ok" {
		t.Errorf("expected couchdb status=ok")
	}
}

// TestValidateJWTConfigConfirmsFailure - JWT_SECRET must be required
func TestValidateJWTConfigConfirmsFailure(t *testing.T) {
	os.Setenv("JWT_SECRET", "")
	defer os.Unsetenv("JWT_SECRET")
	os.Unsetenv("JWT_SECRET") // Explicitly unset

	if err := validateConfig(); err == nil {
		t.Fatal("expected validateConfig to fail when JWT_SECRET is empty")
	}
}

func TestValidateJWTConfigPassesWithSecret(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-test")
	defer os.Unsetenv("JWT_SECRET")

	if err := validateConfig(); err != nil {
		t.Fatalf("expected validateConfig to pass with JWT_SECRET set, got: %v", err)
	}
}
