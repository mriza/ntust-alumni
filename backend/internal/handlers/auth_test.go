package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"ntust-alumni/backend/internal/db"
	"ntust-alumni/backend/internal/models"
)

// fakeDB implements enough of db.CouchDB for auth handler tests
type fakeDB struct {
	alumni  map[string]*models.Alumni
	byEmail map[string]*models.Alumni
	nextID  int
}

func newFakeDB() *fakeDB {
	return &fakeDB{
		alumni:  make(map[string]*models.Alumni),
		byEmail: make(map[string]*models.Alumni),
	}
}

func (f *fakeDB) GetAlumni(ctx context.Context, id string) (*models.Alumni, error) {
	if a, ok := f.alumni[id]; ok {
		return a, nil
	}
	return nil, &errNotFound{}
}

func (f *fakeDB) SaveAlumni(ctx context.Context, a *models.Alumni) error {
	if a.ID == "" {
		f.nextID++
		a.ID = string(rune('a' + f.nextID))
	}
	f.alumni[a.ID] = a
	f.byEmail[a.Email] = a
	return nil
}

func (f *fakeDB) UpdateAlumni(ctx context.Context, a *models.Alumni) error {
	if _, ok := f.alumni[a.ID]; !ok {
		return &errNotFound{}
	}
	f.alumni[a.ID] = a
	f.byEmail[a.Email] = a
	return nil
}

func (f *fakeDB) ListAlumni(ctx context.Context) ([]models.Alumni, error) {
	var result []models.Alumni
	for _, a := range f.alumni {
		result = append(result, *a)
	}
	return result, nil
}

func (f *fakeDB) FindAlumniByEmail(ctx context.Context, email string) (*models.Alumni, error) {
	if a, ok := f.byEmail[email]; ok {
		return a, nil
	}
	return nil, nil
}

func (f *fakeDB) GetSubgroup(ctx context.Context, id string) (*models.Subgroup, error) { return nil, nil }
func (f *fakeDB) SaveSubgroup(ctx context.Context, s *models.Subgroup) error            { return nil }
func (f *fakeDB) UpdateSubgroup(ctx context.Context, s *models.Subgroup) error         { return nil }
func (f *fakeDB) DeleteSubgroup(ctx context.Context, id, rev string) error             { return nil }
func (f *fakeDB) ListSubgroups(ctx context.Context) ([]models.Subgroup, error)         { return nil, nil }
func (f *fakeDB) SaveForumPost(ctx context.Context, p *models.ForumPost) error         { return nil }
func (f *fakeDB) ListForumPosts(ctx context.Context) ([]models.ForumPost, error)       { return nil, nil }
func (f *fakeDB) InitDatabase(ctx context.Context) error                               { return nil }

// Compile-time check that fakeDB satisfies db.Store interface
var _ db.Store = (*fakeDB)(nil)

type errNotFound struct{}
func (e *errNotFound) Error() string { return "not found" }

func TestRegisterCreatesPendingAlumni(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	h := NewAuthHandler(database)

	router := gin.New()
	router.POST("/register", h.Register)

	body := `{"nama":"Budi","email":"budi@gmail.com","password":"secret123","nim":"12345","jurusan":"Teknik","tahun_lulus":2020,"wilayah":"Jakarta","telepon":"08123456789"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	if len(database.alumni) != 1 {
		t.Fatalf("expected 1 alumni, got %d", len(database.alumni))
	}
	for _, a := range database.alumni {
		if a.Status != "pending" {
			t.Errorf("expected status pending, got %s", a.Status)
		}
		if a.Role != "member" {
			t.Errorf("expected role member, got %s", a.Role)
		}
		if a.PasswordHash == "" {
			t.Error("expected password hash to be set")
		}
		if a.Email != "budi@gmail.com" {
			t.Errorf("expected email budi@gmail.com, got %s", a.Email)
		}
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	h := NewAuthHandler(database)

	database.SaveAlumni(context.Background(), &models.Alumni{Email: "dup@example.com", Status: "pending", Role: "member"})

	router := gin.New()
	router.POST("/register", h.Register)

	body := `{"nama":"Dup","email":"dup@example.com","password":"secret123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestLoginSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	SetJWTKey([]byte("test-secret"))

	hashed, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:           "u1",
		Nama:         "Budi",
		Email:        "budi@example.com",
		PasswordHash: string(hashed),
		Role:         "member",
		Status:       "verified",
	})

	h := NewAuthHandler(database)
	router := gin.New()
	router.POST("/login", h.Login)

	body := `{"email":"budi@example.com","password":"secret123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.TokenResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Token == "" {
		t.Error("expected token to be set")
	}
	if resp.Role != "member" {
		t.Errorf("expected role member, got %s", resp.Role)
	}
	if resp.ID != "u1" {
		t.Errorf("expected ID u1, got %s", resp.ID)
	}
	if resp.Nama != "Budi" {
		t.Errorf("expected nama Budi, got %s", resp.Nama)
	}

	parsed, err := jwt.Parse(resp.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	if err != nil || !parsed.Valid {
		t.Errorf("token is invalid: %v", err)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	SetJWTKey([]byte("test-secret"))

	hashed, _ := bcrypt.GenerateFromPassword([]byte("correctpass"), bcrypt.DefaultCost)
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:           "u1",
		Email:        "budi@example.com",
		PasswordHash: string(hashed),
		Role:         "member",
		Status:       "verified",
	})

	h := NewAuthHandler(database)
	router := gin.New()
	router.POST("/login", h.Login)

	body := `{"email":"budi@example.com","password":"wrongpass"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLoginRejectsUnknownEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	SetJWTKey([]byte("test-secret"))

	h := NewAuthHandler(database)
	router := gin.New()
	router.POST("/login", h.Login)

	body := `{"email":"unknown@example.com","password":"secret123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
