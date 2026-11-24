package db

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ntust-alumni/backend/internal/models"
)

func TestNewCouchDBStoresBaseURL(t *testing.T) {
	// Construct directly to avoid requiring a live CouchDB for ping
	c := &CouchDB{
		baseURL: "http://admin:pass@localhost:5984",
		client:  &http.Client{},
	}
	if c.baseURL != "http://admin:pass@localhost:5984" {
		t.Fatalf("expected baseURL %q, got %q", "http://admin:pass@localhost:5984", c.baseURL)
	}
	if c.client == nil {
		t.Fatal("expected http client to be set")
	}
}

func TestSaveAlumniSerializesCorrectly(t *testing.T) {
	// Use a mock HTTP server to verify the request body
	var receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/alumni" {
			t.Errorf("expected path /alumni, got %s", r.URL.Path)
		}
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		receivedBody = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "abc123", "rev": "1-xyz"})
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	a := &models.Alumni{
		Nama:     "Budi",
		Email:    "budi@example.com",
		Role:     "member",
		Status:   "pending",
		Wilayah:  "Jakarta",
	}

	if err := c.SaveAlumni(context.Background(), a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.ID != "abc123" {
		t.Errorf("expected ID abc123, got %s", a.ID)
	}
	if a.Rev != "1-xyz" {
		t.Errorf("expected rev 1-xyz, got %s", a.Rev)
	}

	if !strings.Contains(receivedBody, `"nama":"Budi"`) {
		t.Errorf("expected body to contain nama Budi, got %s", receivedBody)
	}
	if !strings.Contains(receivedBody, `"email":"budi@example.com"`) {
		t.Errorf("expected body to contain email, got %s", receivedBody)
	}
	if !strings.Contains(receivedBody, `"role":"member"`) {
		t.Errorf("expected body to contain role member, got %s", receivedBody)
	}
	if !strings.Contains(receivedBody, `"status":"pending"`) {
		t.Errorf("expected body to contain status pending, got %s", receivedBody)
	}
}

func TestGetAlumniReturnsNotFoundError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	_, err := c.GetAlumni(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent alumni")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got %v", err)
	}
}

func TestListAlumniParsesRows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rows": []interface{}{
				map[string]interface{}{
					"doc": map[string]interface{}{
						"_id":   "id1",
						"_rev":  "rev1",
						"nama":  "Alice",
						"email": "alice@example.com",
						"role":  "member",
						"status": "verified",
					},
				},
				map[string]interface{}{
					"doc": map[string]interface{}{
						"_id":   "id2",
						"_rev":  "rev2",
						"nama":  "Bob",
						"email": "bob@example.com",
						"role":  "member",
						"status": "verified",
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	alumni, err := c.ListAlumni(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alumni) != 2 {
		t.Fatalf("expected 2 alumni, got %d", len(alumni))
	}
	if alumni[0].Nama != "Alice" {
		t.Errorf("expected first nama Alice, got %s", alumni[0].Nama)
	}
	if alumni[1].Nama != "Bob" {
		t.Errorf("expected second nama Bob, got %s", alumni[1].Nama)
	}
}

func TestFindAlumniByEmailReturnsNilWhenNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"docs": []interface{}{},
		})
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	result, err := c.FindAlumniByEmail(context.Background(), "nonexistent@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestFindAlumniByEmailReturnsResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"docs": []interface{}{
				map[string]interface{}{
					"_id":   "id1",
					"_rev":  "rev1",
					"nama":  "Charlie",
					"email": "charlie@example.com",
					"role":  "member",
					"status": "verified",
				},
			},
		})
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	result, err := c.FindAlumniByEmail(context.Background(), "charlie@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Nama != "Charlie" {
		t.Errorf("expected nama Charlie, got %s", result.Nama)
	}
}

func TestSaveSubgroup(t *testing.T) {
	var receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subgroups" {
			t.Errorf("expected path /subgroups, got %s", r.URL.Path)
		}
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		receivedBody = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "sub1", "rev": "1-abc"})
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	s := &models.Subgroup{
		Nama:     "IKATAN ALUMNI NTUST WILAYAH JAKARTA",
		Wilayah:  "Jakarta",
		AdminID:  "alumni123",
	}

	if err := c.SaveSubgroup(context.Background(), s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ID != "sub1" {
		t.Errorf("expected ID sub1, got %s", s.ID)
	}
	if !strings.Contains(receivedBody, `"nama":"IKATAN ALUMNI NTUST WILAYAH JAKARTA"`) {
		t.Errorf("expected body to contain nama, got %s", receivedBody)
	}
}

func TestDeleteSubgroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/subgroups/") {
			t.Errorf("expected path to contain /subgroups/, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("rev") != "rev123" {
			t.Errorf("expected rev query param rev123, got %s", r.URL.Query().Get("rev"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	if err := c.DeleteSubgroup(context.Background(), "sub1", "rev123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveForumPost(t *testing.T) {
	var receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/forum_posts" {
			t.Errorf("expected path /forum_posts, got %s", r.URL.Path)
		}
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		receivedBody = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "fp1", "rev": "1-xyz"})
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	p := &models.ForumPost{
		Judul:       "Halo semua",
		Isi:         "Ini postingan pertama",
		PenulisID:   "alumni1",
		PenulisNama: "Budi",
		Wilayah:     "Jakarta",
	}

	if err := c.SaveForumPost(context.Background(), p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID != "fp1" {
		t.Errorf("expected ID fp1, got %s", p.ID)
	}
	if !strings.Contains(receivedBody, `"judul":"Halo semua"`) {
		t.Errorf("expected body to contain judul, got %s", receivedBody)
	}
	if !strings.Contains(receivedBody, `"penulis_nama":"Budi"`) {
		t.Errorf("expected body to contain penulis_nama, got %s", receivedBody)
	}
}

func TestAlumniModelJSONExcludesPassword(t *testing.T) {
	a := models.Alumni{
		ID:           "id1",
		Nama:         "Test",
		Email:        "test@example.com",
		PasswordHash: "secret-hash",
		Role:         "member",
		Status:       "pending",
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := string(data)
	if strings.Contains(body, "PasswordHash") || strings.Contains(body, "secret-hash") {
		t.Error("PasswordHash should be excluded from JSON (json:\"-\")")
	}
	if !strings.Contains(body, `"nama":"Test"`) {
		t.Errorf("expected nama in JSON, got %s", body)
	}
}

func TestForumPostListDescending(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "descending=true") {
			t.Errorf("expected descending=true in query, got %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rows": []interface{}{},
		})
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	_, err := c.ListForumPosts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
