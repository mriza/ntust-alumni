package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"ntust-alumni/backend/internal/models"
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
	os.Setenv("COUCHDB_URL", "http://testuser:testpass@localhost:5984")
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("COUCHDB_URL")

	if err := validateConfig(); err != nil {
		t.Fatalf("expected validateConfig to pass with JWT_SECRET and COUCHDB_URL set, got: %v", err)
	}
}

func TestValidateConfigFailsWithoutCouchDBURL(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-test")
	os.Unsetenv("COUCHDB_URL")
	defer os.Unsetenv("JWT_SECRET")

	if err := validateConfig(); err == nil {
		t.Fatal("expected validateConfig to fail when COUCHDB_URL is empty")
	}
}

func TestSecurityHeadersAndCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Access-Control-Allow-Origin *, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Errorf("expected Access-Control-Allow-Credentials to be omitted with *, got %q", w.Header().Get("Access-Control-Allow-Credentials"))
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w2, req2)

	for _, hdr := range []string{"X-Content-Type-Options", "X-Frame-Options", "X-XSS-Protection", "Strict-Transport-Security", "Content-Security-Policy"} {
		if w2.Header().Get(hdr) == "" {
			t.Errorf("expected security header %s to be set", hdr)
		}
	}
}

type failingPingStore struct{}

func (f *failingPingStore) GetAlumni(context.Context, string) (*models.Alumni, error) { return nil, nil }
func (f *failingPingStore) SaveAlumni(context.Context, *models.Alumni) error         { return nil }
func (f *failingPingStore) UpdateAlumni(context.Context, *models.Alumni) error       { return nil }
func (f *failingPingStore) ListAlumni(context.Context, string, string, string, int, int) ([]models.Alumni, error) {
	return nil, nil
}
func (f *failingPingStore) FindAlumniByEmail(context.Context, string) (*models.Alumni, error) {
	return nil, nil
}
func (f *failingPingStore) GetSubgroup(context.Context, string) (*models.Subgroup, error) {
	return nil, nil
}
func (f *failingPingStore) SaveSubgroup(context.Context, *models.Subgroup) error   { return nil }
func (f *failingPingStore) UpdateSubgroup(context.Context, *models.Subgroup) error { return nil }
func (f *failingPingStore) DeleteSubgroup(context.Context, string, string) error   { return nil }
func (f *failingPingStore) ListSubgroups(context.Context) ([]models.Subgroup, error) {
	return nil, nil
}
func (f *failingPingStore) SaveForumPost(context.Context, *models.ForumPost) error { return nil }
func (f *failingPingStore) ListForumPosts(context.Context) ([]models.ForumPost, error) {
	return nil, nil
}
func (f *failingPingStore) FindAlumniByVerificationToken(context.Context, string) (*models.Alumni, error) {
	return nil, nil
}
func (f *failingPingStore) InitDatabase(context.Context) error { return nil }
func (f *failingPingStore) Ping() error                        { return errors.New("connection refused") }

func TestHealthEndpointReturns503WhenCouchDBDown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prev := dbStore
	dbStore = &failingPingStore{}
	defer func() { dbStore = prev }()

	router := SetupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when CouchDB down, got %d", w.Code)
	}
}
