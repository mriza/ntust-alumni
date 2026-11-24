package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"ntust-alumni/backend/internal/middleware"
	"ntust-alumni/backend/internal/models"
)

func TestCreateForumPostRejectsPending(t *testing.T) {
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

	h := NewForumHandler(database)

	router := gin.New()
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.POST("/forum", h.CreateForumPost)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  "u1",
		"email":    "pending@example.com",
		"role":     "member",
		"status":   "pending",
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	body := `{"judul":"Test","isi":"Test"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/forum", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for pending user, got %d", w.Code)
	}
}

func TestCreateForumPostAsVerified(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	// Create verified alumni
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:           "u1",
		Nama:         "Verified",
		Email:        "verified@example.com",
		Role:         "member",
		Status:       "verified",
		PasswordHash: "hash",
		Wilayah:      "Jakarta",
	})

	h := NewForumHandler(database)

	router := gin.New()
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.POST("/forum", h.CreateForumPost)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  "u1",
		"email":    "verified@example.com",
		"role":     "member",
		"status":   "verified",
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	body := `{"judul":"Baru saja pulang kerja","isi":"Hari ini sangat berat tapi puas karena proyek selesai","wilayah":""}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/forum", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify post was created
	posts, _ := database.ListForumPosts(context.Background())
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].Judul != "Baru saja pulang kerja" {
		t.Errorf("expected judul 'Baru saja pulang kerja', got %s", posts[0].Judul)
	}
	if posts[0].PenulisNama != "Verified" {
		t.Errorf("expected penulis_nama Verified, got %s", posts[0].PenulisNama)
	}
	if posts[0].PenulisID != "u1" {
		t.Errorf("expected penulis_id u1, got %s", posts[0].PenulisID)
	}
	if posts[0].Wilayah != "Jakarta" {
		t.Errorf("expected wilayah Jakarta, got %s", posts[0].Wilayah)
	}
}

func TestListForumPosts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()

	// Create forum post directly
	database.SaveForumPost(context.Background(), &models.ForumPost{
		Judul:       "Pertanyaan umum",
		Isi:         "Ada yang bisa bantu?",
		PenulisID:   "u1",
		PenulisNama: "Budi",
	})

	h := NewForumHandler(database)

	router := gin.New()
	router.GET("/forum", h.ListForumPosts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/forum", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}
