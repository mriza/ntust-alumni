package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"ntust-alumni/backend/internal/models"
	"ntust-alumni/backend/internal/middleware"
)

func TestVerificationRequestAndVerifyFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	// Create alumni
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:            "u1",
		Nama:          "Budi",
		Email:         "budi@gmail.com",
		Role:          "member",
		Status:        "pending",
		PasswordHash:  "hash",
	})

	emailClient := &fakeEmailClient{}
	h := NewVerificationHandler(database, emailClient)

	router := gin.New()
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.POST("/request-verification", h.RequestVerification)
	router.POST("/verify", h.Verify)

	// Step 1: Request verification
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  "u1",
		"email":    "budi@gmail.com",
		"role":     "member",
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	body := `{"email_ntust":"budi@mail.ntust.edu.tw"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/request-verification", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if emailClient.callCount != 1 {
		t.Fatalf("expected 1 email sent, got %d", emailClient.callCount)
	}
	if emailClient.sentTo != "budi@mail.ntust.edu.tw" {
		t.Errorf("expected email to budi@mail.ntust.edu.tw, got %s", emailClient.sentTo)
	}
	if emailClient.sentOTP == "" {
		t.Error("expected OTP to be generated")
	}
	if emailClient.sentToken == "" {
		t.Error("expected token to be generated")
	}

	// Step 2: Verify with OTP and NIM
	alumni, _ := database.GetAlumni(context.Background(), "u1")
	verifyBody := map[string]string{
		"token": alumni.VerificationToken,
		"otp":   emailClient.sentOTP,
		"nim":   "12345",
	}
	verifyBodyJSON, _ := json.Marshal(verifyBody)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewBuffer(verifyBodyJSON))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Verify status updated
	alumni2, err := database.GetAlumni(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alumni2.Status != "verified" {
		t.Errorf("expected status verified, got %s", alumni2.Status)
	}
	if alumni2.Nim != "12345" {
		t.Errorf("expected NIM 12345, got %s", alumni2.Nim)
	}
}

func TestVerificationRejectsNonNtustEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:            "u1",
		Nama:          "Budi",
		Email:         "budi@gmail.com",
		Role:          "member",
		Status:        "pending",
		PasswordHash:  "hash",
	})

	emailClient := &fakeEmailClient{}
	h := NewVerificationHandler(database, emailClient)

	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.POST("/request-verification", h.RequestVerification)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "u1",
		"email":   "budi@gmail.com",
		"role":    "member",
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	body := `{"email_ntust":"budi@gmail.com"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/request-verification", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if emailClient.callCount != 0 {
		t.Errorf("expected 0 emails sent, got %d", emailClient.callCount)
	}
}

func TestVerificationRejectsInvalidOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	SetJWTKey([]byte("test-secret"))

	otpHash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	expiresAt := time.Now().Add(15 * time.Minute)

	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:                 "u1",
		Nama:               "Budi",
		Email:              "budi@gmail.com",
		Role:               "member",
		Status:             "pending",
		PasswordHash:       "hash",
		EmailNtust:         "budi@mail.ntust.edu.tw",
		OtpHash:            string(otpHash),
		OtpExpiresAt:       expiresAt,
		VerificationToken:  "valid-token",
		TokenExpiresAt:     expiresAt,
	})

	h := NewVerificationHandler(database, &fakeEmailClient{})

	router := gin.New()
	router.POST("/verify", h.Verify)

	body := `{"token":"valid-token","otp":"wrong-otp","nim":"12345"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
