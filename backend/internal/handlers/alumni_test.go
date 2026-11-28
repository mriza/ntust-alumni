package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"ntust-alumni/backend/internal/middleware"
	"ntust-alumni/backend/internal/models"
)

func TestListAlumniRejectsPending(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	// Create a pending alumni
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:           "u1",
		Nama:         "Pending",
		Email:        "pending@example.com",
		Role:         "member",
		Status:       "pending",
		PasswordHash: "hash",
	})

	// Create a verified alumni
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:           "u2",
		Nama:         "Verified",
		Email:        "verified@example.com",
		Role:         "member",
		Status:       "verified",
		PasswordHash: "hash",
	})

	h := NewAlumniHandler(database)

	router := gin.New()
	verified := router.Group("/")
	verified.Use(middleware.AuthMiddleware())
	verified.Use(middleware.RequireVerified())
	verified.GET("/alumni", h.ListAlumni)
	verified.GET("/alumni/:id", h.GetAlumniByID)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "u1",
		"email":   "pending@example.com",
		"role":    "member",
		"status":  "pending",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/alumni", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for pending alumni, got %d", w.Code)
	}
}

func TestListAlumniReturnsOnlyVerified(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	// Create a verified alumni
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:           "u2",
		Nama:         "Verified",
		Email:        "verified@example.com",
		Role:         "member",
		Status:       "verified",
		PasswordHash: "hash",
		Nim:          "12345",
		Sosmed:       &models.Sosmed{Instagram: "@verified"},
	})

	h := NewAlumniHandler(database)

	router := gin.New()
	verified := router.Group("/")
	verified.Use(middleware.AuthMiddleware())
	verified.Use(middleware.RequireVerified())
	verified.GET("/alumni", h.ListAlumni)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "u2",
		"email":   "verified@example.com",
		"role":    "member",
		"status":  "verified",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/alumni", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Should contain only verified alumni
	body := w.Body.String()
	if !strings.Contains(body, "verified@example.com") {
		t.Errorf("expected verified alumni in response, got %s", body)
	}
	if !strings.Contains(body, "sosmed") {
		t.Errorf("expected sosmed in response, got %s", body)
	}
}

func TestUpdateAlumniSelfUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:           "u1",
		Nama:         "Old Name",
		Email:        "alumni@example.com",
		Role:         "member",
		Status:       "verified",
		PasswordHash: "hash",
	})

	h := NewAlumniHandler(database)

	router := gin.New()
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.PUT("/alumni/:id", h.UpdateAlumni)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "u1",
		"email":   "alumni@example.com",
		"role":    "member",
		"status":  "verified",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	body := `{"nama":"New Name","no_wa":"08123456789","sosmed":{"instagram":"@newprofile"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/alumni/u1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	alumni, _ := database.GetAlumni(context.Background(), "u1")
	if alumni.Nama != "New Name" {
		t.Errorf("expected nama New Name, got %s", alumni.Nama)
	}
	if alumni.NoWA != "08123456789" {
		t.Errorf("expected no_wa 08123456789, got %s", alumni.NoWA)
	}
	if alumni.Sosmed == nil || alumni.Sosmed.Instagram != "@newprofile" {
		t.Errorf("expected sosmed.instagram @newprofile, got %v", alumni.Sosmed)
	}
}

func TestUpdateAlumniRejectsOtherUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:           "u1",
		Nama:         "User 1",
		Email:        "user1@example.com",
		Role:         "member",
		Status:       "verified",
		PasswordHash: "hash",
	})
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:           "u2",
		Nama:         "User 2",
		Email:        "user2@example.com",
		Role:         "member",
		Status:       "verified",
		PasswordHash: "hash",
	})

	h := NewAlumniHandler(database)

	router := gin.New()
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.PUT("/alumni/:id", h.UpdateAlumni)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "u1",
		"email":   "user1@example.com",
		"role":    "member",
		"status":  "verified",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	body := `{"nama":"Hacked"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/alumni/u2", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestListAlumniSearchFilterSortPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveAlumni(context.Background(), &models.Alumni{ID: "p1", Nama: "Pending", Email: "p@x.com", Status: "pending", Wilayah: "Jakarta", Jurusan: "CSIE"})
	database.SaveAlumni(context.Background(), &models.Alumni{ID: "v1", Nama: "Andi", Email: "andi@x.com", Status: "verified", Wilayah: "Jakarta", Jurusan: "CSIE"})
	database.SaveAlumni(context.Background(), &models.Alumni{ID: "v2", Nama: "Zaki", Email: "zaki@x.com", Status: "verified", Wilayah: "Jakarta", Jurusan: "CSIE"})
	database.SaveAlumni(context.Background(), &models.Alumni{ID: "v3", Nama: "Budi", Email: "budi@x.com", Status: "verified", Wilayah: "Bandung", Jurusan: "ECE"})

	h := NewAlumniHandler(database)
	router := gin.New()
	router.Use(middleware.AuthMiddleware(), middleware.RequireVerified())
	router.GET("/alumni", h.ListAlumni)

	tok, _ := GenerateJWT(&models.Alumni{ID: "v1", Email: "andi@x.com", Role: "member", Status: "verified"})

	// Filter wilayah=Jakarta, sort=nama_desc, page=1, limit=1 -> total 2, first item Zaki
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/alumni?wilayah=Jakarta&jurusan=CSIE&sort=nama_desc&page=1&limit=1", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Alumni []AlumniPublic `json:"alumni"`
			Page   int            `json:"page"`
			Limit  int            `json:"limit"`
			Total  int            `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Data.Total != 2 || len(resp.Data.Alumni) != 1 || resp.Data.Alumni[0].Nama != "Zaki" {
		t.Errorf("unexpected filtered/sorted/paginated result: %+v", resp.Data)
	}

	// Search q=budi
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/alumni?q=budi", nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	router.ServeHTTP(w2, req2)
	var resp2 struct {
		Data struct {
			Alumni []AlumniPublic `json:"alumni"`
			Total  int            `json:"total"`
		} `json:"data"`
	}
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.Data.Total != 1 || len(resp2.Data.Alumni) != 1 || resp2.Data.Alumni[0].Nama != "Budi" {
		t.Errorf("unexpected search result: %+v", resp2.Data)
	}
}

func TestUpdateAlumniPhotoValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveAlumni(context.Background(), &models.Alumni{ID: "u1", Nama: "User", Email: "u@x.com", Role: "member", Status: "verified"})
	h := NewAlumniHandler(database)
	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.POST("/alumni/:id", h.UpdateAlumni)

	tok, _ := GenerateJWT(&models.Alumni{ID: "u1", Email: "u@x.com", Role: "member", Status: "verified"})

	sendPhoto := func(photo string) int {
		b, _ := json.Marshal(map[string]string{"foto_profil": photo})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/alumni/u1", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)
		router.ServeHTTP(w, req)
		return w.Code
	}

	// Valid 1x1 PNG
	validPNG := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
	if code := sendPhoto(validPNG); code != http.StatusOK {
		t.Errorf("expected 200 for valid PNG, got %d", code)
	}
	// Reject lowercase SVG and uppercase SVG bypass
	svg := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte("<svg/>"))
	if code := sendPhoto(svg); code != http.StatusBadRequest {
		t.Errorf("expected 400 for SVG, got %d", code)
	}
	svgUpper := "data:image/SVG+xml;base64," + base64.StdEncoding.EncodeToString([]byte("<svg/>"))
	if code := sendPhoto(svgUpper); code != http.StatusBadRequest {
		t.Errorf("expected 400 for uppercase SVG, got %d", code)
	}
	// Reject spoofed MIME header with non-PNG bytes
	fakePNG := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("<svg onload=alert(1)/>"))
	if code := sendPhoto(fakePNG); code != http.StatusBadRequest {
		t.Errorf("expected 400 for spoofed PNG bytes, got %d", code)
	}
	// Reject non-data-URL
	if code := sendPhoto("https://evil.com/x.png"); code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-data URL, got %d", code)
	}
	// Reject >2MB
	big := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(make([]byte, maxPhotoBytes+1))
	if code := sendPhoto(big); code != http.StatusBadRequest {
		t.Errorf("expected 400 for >2MB photo, got %d", code)
	}
}

func TestGetAlumniByIDHidesPendingFromOtherUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveAlumni(context.Background(), &models.Alumni{ID: "p1", Nama: "Pending User", Email: "p@x.com", Status: "pending", Role: "member"})
	database.SaveAlumni(context.Background(), &models.Alumni{ID: "v1", Nama: "Verified User", Email: "v@x.com", Status: "verified", Role: "member"})

	h := NewAlumniHandler(database)
	router := gin.New()
	router.Use(middleware.AuthMiddleware(), middleware.RequireVerified())
	router.GET("/alumni/:id", h.GetAlumniByID)

	verTok, _ := GenerateJWT(&models.Alumni{ID: "v1", Email: "v@x.com", Role: "member", Status: "verified"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/alumni/p1", nil)
	req.Header.Set("Authorization", "Bearer "+verTok)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when verified member queries pending profile by ID, got %d", w.Code)
	}
}

func TestUpdateAlumniRejectsInvalidLinkedInAndOversizedInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveAlumni(context.Background(), &models.Alumni{ID: "u1", Nama: "User", Email: "u@x.com", Role: "member", Status: "verified"})
	h := NewAlumniHandler(database)
	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.PUT("/alumni/:id", h.UpdateAlumni)

	tok, _ := GenerateJWT(&models.Alumni{ID: "u1", Email: "u@x.com", Role: "member", Status: "verified"})

	wBadURL := httptest.NewRecorder()
	reqBadURL, _ := http.NewRequest(http.MethodPut, "/alumni/u1", bytes.NewBufferString(`{"linkedin":"javascript:alert(1)"}`))
	reqBadURL.Header.Set("Content-Type", "application/json")
	reqBadURL.Header.Set("Authorization", "Bearer "+tok)
	router.ServeHTTP(wBadURL, reqBadURL)
	if wBadURL.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for javascript: LinkedIn URL, got %d", wBadURL.Code)
	}

	oversized := `{"nama":"` + strings.Repeat("A", 205) + `"}`
	wBig := httptest.NewRecorder()
	reqBig, _ := http.NewRequest(http.MethodPut, "/alumni/u1", bytes.NewBufferString(oversized))
	reqBig.Header.Set("Content-Type", "application/json")
	reqBig.Header.Set("Authorization", "Bearer "+tok)
	router.ServeHTTP(wBig, reqBig)
	if wBig.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized nama, got %d", wBig.Code)
	}
}
