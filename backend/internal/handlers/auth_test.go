package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"ntust-alumni/backend/internal/db"
	"ntust-alumni/backend/internal/middleware"
	"ntust-alumni/backend/internal/models"
)

// fakeDB implements enough of db.CouchDB for auth handler tests
type fakeDB struct {
	alumni      map[string]*models.Alumni
	byEmail     map[string]*models.Alumni
	subgroups   map[string]*models.Subgroup
	forumPosts  map[string]*models.ForumPost
	nextID      int
}

func newFakeDB() *fakeDB {
	return &fakeDB{
		alumni:      make(map[string]*models.Alumni),
		byEmail:     make(map[string]*models.Alumni),
		subgroups:   make(map[string]*models.Subgroup),
		forumPosts:  make(map[string]*models.ForumPost),
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

func (f *fakeDB) ListAlumni(ctx context.Context, search, wilayah, jurusan string, limit, offset int) ([]models.Alumni, error) {
	q := strings.ToLower(search)
	var result []models.Alumni
	for _, a := range f.alumni {
		if wilayah != "" && a.Wilayah != wilayah {
			continue
		}
		if jurusan != "" && a.Jurusan != jurusan {
			continue
		}
		if q != "" {
			match := strings.Contains(strings.ToLower(a.Nama), q) ||
				strings.Contains(strings.ToLower(a.Email), q) ||
				strings.Contains(strings.ToLower(a.Nim), q) ||
				strings.Contains(strings.ToLower(a.Wilayah), q) ||
				strings.Contains(strings.ToLower(a.Jurusan), q)
			if !match && a.Pekerjaan != nil {
				match = strings.Contains(strings.ToLower(a.Pekerjaan.Institusi), q) ||
					strings.Contains(strings.ToLower(a.Pekerjaan.Jabatan), q)
			}
			if !match {
				continue
			}
		}
		result = append(result, *a)
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(result) {
		return []models.Alumni{}, nil
	}
	if limit > 0 {
		end := offset + limit
		if end > len(result) {
			end = len(result)
		}
		return result[offset:end], nil
	}
	return result[offset:], nil
}

func (f *fakeDB) FindAlumniByEmail(ctx context.Context, email string) (*models.Alumni, error) {
	if a, ok := f.byEmail[email]; ok {
		return a, nil
	}
	return nil, nil
}

func (f *fakeDB) FindAlumniByVerificationToken(ctx context.Context, token string) (*models.Alumni, error) {
	for _, a := range f.alumni {
		if a.VerificationToken == token {
			return a, nil
		}
	}
	return nil, nil
}

func (f *fakeDB) GetSubgroup(ctx context.Context, id string) (*models.Subgroup, error) {
	if s, ok := f.subgroups[id]; ok {
		return s, nil
	}
	return nil, &errNotFound{}
}

func (f *fakeDB) SaveSubgroup(ctx context.Context, s *models.Subgroup) error {
	f.nextID++
	s.ID = string(rune('s' + f.nextID))
	f.subgroups[s.ID] = s
	return nil
}

func (f *fakeDB) UpdateSubgroup(ctx context.Context, s *models.Subgroup) error {
	if _, ok := f.subgroups[s.ID]; !ok {
		return &errNotFound{}
	}
	f.subgroups[s.ID] = s
	return nil
}

func (f *fakeDB) DeleteSubgroup(ctx context.Context, id, rev string) error {
	delete(f.subgroups, id)
	return nil
}

func (f *fakeDB) ListSubgroups(ctx context.Context) ([]models.Subgroup, error) {
	var result []models.Subgroup
	for _, s := range f.subgroups {
		result = append(result, *s)
	}
	return result, nil
}

func (f *fakeDB) SaveForumPost(ctx context.Context, p *models.ForumPost) error {
	f.nextID++
	p.ID = string(rune('f' + f.nextID))
	f.forumPosts[p.ID] = p
	return nil
}

func (f *fakeDB) ListForumPosts(ctx context.Context) ([]models.ForumPost, error) {
	var result []models.ForumPost
	for _, p := range f.forumPosts {
		result = append(result, *p)
	}
	return result, nil
}
func (f *fakeDB) InitDatabase(ctx context.Context) error                               { return nil }

func (f *fakeDB) Ping() error                                                           { return nil }

// Compile-time check that fakeDB satisfies db.Store interface
var _ db.Store = (*fakeDB)(nil)

type errNotFound struct{}
func (e *errNotFound) Error() string { return "not found" }

// fakeEmailClient implements email, SMS & WhatsApp sending for verification handler tests
type fakeEmailClient struct {
	sentTo       string
	sentPhone    string
	sentChannel  string
	sentOTP      string
	sentToken    string
	sendError    error
	callCount    int
}

func (f *fakeEmailClient) SendVerificationOTP(to, otp, token string) error {
	f.callCount++
	f.sentChannel = "email"
	f.sentTo = to
	f.sentOTP = otp
	f.sentToken = token
	return f.sendError
}

func (f *fakeEmailClient) SendVerificationSMS(phone, otp string) error {
	f.callCount++
	f.sentChannel = "sms"
	f.sentPhone = phone
	f.sentOTP = otp
	return f.sendError
}

func (f *fakeEmailClient) SendVerificationWhatsApp(phone, otp string) error {
	f.callCount++
	f.sentChannel = "whatsapp"
	f.sentPhone = phone
	f.sentOTP = otp
	return f.sendError
}

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

	// Email sudah terdaftar: harus return 200 generik (anti-enumerasi)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for duplicate email (anti-enumeration), got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "check your inbox") {
		t.Errorf("expected generic message, got %s", w.Body.String())
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

func TestMeReturnsWrappedProfileWithStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:      "u1",
		Nama:    "Budi",
		Email:   "budi@example.com",
		Status:  "verified",
		Role:    "member",
		Wilayah: "Jakarta",
		Jurusan: "CSIE",
	})

	h := NewAuthHandler(database)
	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.GET("/me", h.Me)

	tok, _ := GenerateJWT(&models.Alumni{ID: "u1", Email: "budi@example.com", Role: "member", Status: "verified"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Message string       `json:"message"`
		Data    AlumniPublic `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Data.ID != "u1" || resp.Data.Status != "verified" || resp.Data.Jurusan != "CSIE" {
		t.Errorf("unexpected /me data: %+v", resp.Data)
	}
}

func TestRegisterPersistsWilayahTeleponTahunLulusAndValidates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	h := NewAuthHandler(database)

	router := gin.New()
	router.POST("/register", h.Register)

	// Invalid short password
	wBad := httptest.NewRecorder()
	reqBad, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"nama":"A","email":"a@b.com","password":"123"}`))
	reqBad.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short password, got %d", wBad.Code)
	}

	// Valid registration
	body := `{"nama":"Siti","email":"Siti@Example.com","password":"secret123","nim":"M109","jurusan":"ECE","tahun_lulus":2022,"wilayah":"Bandung","telepon":"0811111"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	saved, _ := database.FindAlumniByEmail(context.Background(), "siti@example.com")
	if saved == nil || saved.Wilayah != "Bandung" || saved.NoTelp != "0811111" || saved.PendidikanNTUST == nil || saved.PendidikanNTUST.TahunLulus != 2022 {
		t.Errorf("expected fields persisted on register, got %+v", saved)
	}
}

func TestLoginRateLimitReturns429(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	SetJWTKey([]byte("test-secret"))
	h := NewAuthHandler(database)

	router := gin.New()
	router.POST("/login", h.Login)

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"rate@example.com","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, w.Code)
		}
	}

	w6 := httptest.NewRecorder()
	req6, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"rate@example.com","password":"wrong"}`))
	req6.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w6, req6)
	if w6.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on 6th login attempt, got %d", w6.Code)
	}
}
