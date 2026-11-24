package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
		"user_id":  "u1",
		"email":    "pending@example.com",
		"role":     "member",
		"status":   "pending",
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
		"user_id":  "u2",
		"email":    "verified@example.com",
		"role":     "member",
		"status":   "verified",
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
		"user_id":  "u1",
		"email":    "alumni@example.com",
		"role":     "member",
		"status":   "verified",
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
		"user_id":  "u1",
		"email":    "user1@example.com",
		"role":     "member",
		"status":   "verified",
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
