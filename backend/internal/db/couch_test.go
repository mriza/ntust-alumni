package db

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ntust-alumni/backend/internal/models"
)

func TestNewCouchDBStoresBaseURL(t *testing.T) {
	var seenAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"couchdb":"Welcome"}`))
	}))
	defer srv.Close()

	// Insert credentials into URL and verify NewCouchDB strips them from baseURL while sending Basic Auth
	rawWithCreds := strings.Replace(srv.URL, "http://", "http://admin:SuperSecretPass@", 1)
	c, err := NewCouchDB(rawWithCreds)
	if err != nil {
		t.Fatalf("expected NewCouchDB to succeed, got %v", err)
	}
	if strings.Contains(c.baseURL, "SuperSecretPass") || strings.Contains(c.baseURL, "admin@") {
		t.Fatalf("expected credentials stripped from baseURL, got %q", c.baseURL)
	}
	if !strings.HasPrefix(seenAuth, "Basic ") {
		t.Errorf("expected Basic auth header sent to CouchDB, got %q", seenAuth)
	}
	// Verify unreachable host error does not leak password
	_, errFail := NewCouchDB("http://admin:SecretFailPass@127.0.0.1:1")
	if errFail == nil {
		t.Fatal("expected error for closed port")
	}
	if strings.Contains(errFail.Error(), "SecretFailPass") {
		t.Fatalf("expected password stripped from error, got %v", errFail)
	}
}

func TestCouchDBRejectsInvalidDocIDs(t *testing.T) {
	c := &CouchDB{baseURL: "http://127.0.0.1:1", client: &http.Client{}}
	for _, bad := range []string{"", "_all_docs", "_security", "../subgroups/1", "abc/def", "abc?rev=1", "abc#frag"} {
		if _, err := c.GetAlumni(context.Background(), bad); err == nil {
			t.Errorf("expected GetAlumni to reject doc ID %q", bad)
		}
		if err := c.DeleteSubgroup(context.Background(), bad, "1-rev"); err == nil {
			t.Errorf("expected DeleteSubgroup to reject doc ID %q", bad)
		}
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
	alumni, err := c.ListAlumni(context.Background(), "", "", "", 10, 0)
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
		ID:                "id1",
		Nama:              "Test",
		Email:             "test@example.com",
		PasswordHash:      "secret-hash",
		OtpHash:           "secret-otp",
		VerificationToken: "secret-tok",
		Role:              "member",
		Status:            "pending",
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := string(data)
	if strings.Contains(body, "PasswordHash") || strings.Contains(body, "secret-hash") {
		t.Error("PasswordHash should be excluded from JSON (json:\"-\")")
	}
	if strings.Contains(body, "secret-otp") || strings.Contains(body, "secret-tok") {
		t.Error("OtpHash and VerificationToken should be excluded from JSON (json:\"-\")")
	}
	if !strings.Contains(body, `"nama":"Test"`) {
		t.Errorf("expected nama in JSON, got %s", body)
	}
}

func TestSaveAndGetAlumniPersistsAuthAndVerificationFields(t *testing.T) {
	var savedBody string
	exp := time.Now().Add(15 * time.Minute).UTC().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/alumni" {
			b, _ := io.ReadAll(r.Body)
			savedBody = string(b)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": "alum1", "rev": "1-rev"})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/alumni/alum1" {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, savedBody)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	a := &models.Alumni{
		Nama:              "Budi",
		Email:             "budi@example.com",
		PasswordHash:      "hashed-pw",
		OtpHash:           "hashed-otp",
		OtpExpiresAt:      exp,
		VerificationToken: "tok-123",
		TokenExpiresAt:    exp,
		Role:              "member",
		Status:            "pending",
	}
	if err := c.SaveAlumni(context.Background(), a); err != nil {
		t.Fatalf("SaveAlumni error: %v", err)
	}
	if !strings.Contains(savedBody, `"password_hash":"hashed-pw"`) {
		t.Errorf("expected password_hash in CouchDB payload, got %s", savedBody)
	}
	if !strings.Contains(savedBody, `"otp_hash":"hashed-otp"`) || !strings.Contains(savedBody, `"verification_token":"tok-123"`) {
		t.Errorf("expected otp_hash and verification_token in CouchDB payload, got %s", savedBody)
	}

	got, err := c.GetAlumni(context.Background(), "alum1")
	if err != nil {
		t.Fatalf("GetAlumni error: %v", err)
	}
	if got.PasswordHash != "hashed-pw" || got.OtpHash != "hashed-otp" || got.VerificationToken != "tok-123" {
		t.Errorf("expected round-tripped secrets, got %+v", got)
	}
}

func TestGetAndListSubgroupDecodesIDAndRev(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/subgroups/sub1" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"_id":     "sub1",
				"_rev":    "2-rev",
				"nama":    "Jakarta",
				"wilayah": "Jakarta",
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rows": []interface{}{
				map[string]interface{}{
					"doc": map[string]interface{}{
						"_id":     "sub1",
						"_rev":    "2-rev",
						"nama":    "Jakarta",
						"wilayah": "Jakarta",
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	sg, err := c.GetSubgroup(context.Background(), "sub1")
	if err != nil {
		t.Fatalf("GetSubgroup error: %v", err)
	}
	if sg.ID != "sub1" || sg.Rev != "2-rev" {
		t.Errorf("expected ID=sub1 Rev=2-rev, got ID=%s Rev=%s", sg.ID, sg.Rev)
	}

	list, err := c.ListSubgroups(context.Background())
	if err != nil || len(list) != 1 || list[0].ID != "sub1" || list[0].Rev != "2-rev" {
		t.Errorf("unexpected ListSubgroups result: %+v, err=%v", list, err)
	}
}

func TestSubgroupPengurusRoundTrip(t *testing.T) {
	var savedDoc string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/subgroups" {
			b, _ := io.ReadAll(r.Body)
			savedDoc = string(b)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": "sg-jabo", "rev": "1-r"})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/subgroups/sg-jabo" {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, savedDoc)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/subgroups/_all_docs" {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"rows":[{"doc":`+savedDoc+`}]}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	sg := &models.Subgroup{
		Nama:      "Ikatan Alumni NTUST Wilayah Jabodetabek",
		Wilayah:   "Jabodetabek",
		Deskripsi: "Pengurus Jabodetabek",
		AdminID:   "u-admin",
		Pengurus: []models.PengurusWilayah{
			{Jabatan: "Ketua Wilayah", Nama: "Budi Pratama", AlumniID: "u-admin"},
			{Jabatan: "Sekretaris", Nama: "Andi Wijaya", AlumniID: "u-sek"},
		},
	}
	if err := c.SaveSubgroup(context.Background(), sg); err != nil {
		t.Fatalf("SaveSubgroup error: %v", err)
	}
	got, err := c.GetSubgroup(context.Background(), "sg-jabo")
	if err != nil {
		t.Fatalf("GetSubgroup error: %v", err)
	}
	if len(got.Pengurus) != 2 || got.Pengurus[0].Jabatan != "Ketua Wilayah" || got.Pengurus[1].Nama != "Andi Wijaya" {
		t.Errorf("expected 2 pengurus round-tripped, got %+v", got.Pengurus)
	}
	list, err := c.ListSubgroups(context.Background())
	if err != nil || len(list) != 1 || len(list[0].Pengurus) != 2 {
		t.Errorf("expected ListSubgroups to include 2 pengurus, got %+v, err=%v", list, err)
	}
}

func TestListForumPostsDecodesIDAndRev(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rows": []interface{}{
				map[string]interface{}{
					"doc": map[string]interface{}{
						"_id":          "fp1",
						"_rev":         "1-fp",
						"judul":        "Halo",
						"isi":          "Konten",
						"penulis_id":   "u1",
						"penulis_nama": "Budi",
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	posts, err := c.ListForumPosts(context.Background())
	if err != nil || len(posts) != 1 || posts[0].ID != "fp1" || posts[0].Rev != "1-fp" {
		t.Errorf("unexpected ListForumPosts result: %+v, err=%v", posts, err)
	}
}

func TestListAlumniSearchMatchesPekerjaan(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rows": []interface{}{
				map[string]interface{}{
					"doc": map[string]interface{}{
						"_id":    "id1",
						"nama":   "Alice",
						"email":  "alice@example.com",
						"status": "verified",
						"pekerjaan": map[string]interface{}{
							"institusi": "PT Teknologi Nusantara",
							"jabatan":   "Software Engineer",
						},
					},
				},
				map[string]interface{}{
					"doc": map[string]interface{}{
						"_id":    "id2",
						"nama":   "Bob",
						"email":  "bob@example.com",
						"status": "verified",
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := &CouchDB{baseURL: srv.URL, client: srv.Client()}
	got, err := c.ListAlumni(context.Background(), "teknologi", "", "", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Nama != "Alice" {
		t.Errorf("expected only Alice matching pekerjaan, got %+v", got)
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
