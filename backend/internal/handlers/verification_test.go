package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"ntust-alumni/backend/internal/middleware"
	"ntust-alumni/backend/internal/models"
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
		"user_id": "u1",
		"email":   "budi@gmail.com",
		"role":    "member",
		"exp":     time.Now().Add(time.Hour).Unix(),
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
		"exp":     time.Now().Add(time.Hour).Unix(),
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

func TestVerifyReturnsFreshJWTAndRequestIncludesToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:     "u1",
		Nama:   "Budi",
		Email:  "budi@gmail.com",
		Role:   "member",
		Status: "pending",
	})
	emailClient := &fakeEmailClient{}
	h := NewVerificationHandler(database, emailClient)

	router := gin.New()
	router.POST("/verify", h.Verify)
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.POST("/request-verification", h.RequestVerification)

	pendingTok, _ := GenerateJWT(&models.Alumni{ID: "u1", Email: "budi@gmail.com", Role: "member", Status: "pending"})
	wReq := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/request-verification", bytes.NewBufferString(`{"email_ntust":"budi@mail.ntust.edu.tw"}`))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+pendingTok)
	router.ServeHTTP(wReq, req1)

	if wReq.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wReq.Code)
	}
	var reqResp struct {
		Data map[string]string `json:"data"`
	}
	json.Unmarshal(wReq.Body.Bytes(), &reqResp)
	if reqResp.Data["verification_token"] == "" {
		t.Fatal("expected verification_token in response data")
	}

	verifyBody, _ := json.Marshal(map[string]string{
		"token": reqResp.Data["verification_token"],
		"otp":   emailClient.sentOTP,
		"nim":   "M110",
	})
	wVer := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewReader(verifyBody))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wVer, req2)

	if wVer.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wVer.Code, wVer.Body.String())
	}
	var verResp struct {
		Data map[string]string `json:"data"`
	}
	json.Unmarshal(wVer.Body.Bytes(), &verResp)
	if verResp.Data["token"] == "" {
		t.Fatal("expected fresh JWT token in /verify response")
	}
	parsed, err := jwt.Parse(verResp.Data["token"], func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("expected valid fresh JWT, got %v", err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["status"] != "verified" {
		t.Errorf("expected status=verified in new JWT, got %v", claims["status"])
	}
}

func TestVerifyOTPRateLimitReturns429(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	otpHash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	exp := time.Now().Add(15 * time.Minute)
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:                "u-rl",
		Email:             "rl@gmail.com",
		Status:            "pending",
		OtpHash:           string(otpHash),
		OtpExpiresAt:      exp,
		VerificationToken: "tok-rl",
		TokenExpiresAt:    exp,
	})
	h := NewVerificationHandler(database, &fakeEmailClient{})
	router := gin.New()
	router.POST("/verify", h.Verify)

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewBufferString(`{"token":"tok-rl","otp":"000000","nim":"1"}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, w.Code)
		}
	}
	w6 := httptest.NewRecorder()
	req6, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewBufferString(`{"token":"tok-rl","otp":"123456","nim":"1"}`))
	req6.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w6, req6)
	if w6.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on 6th OTP attempt, got %d", w6.Code)
	}
}

func TestVerifyRejectsExpiredOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	otpHash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:                "u-exp",
		Email:             "exp@gmail.com",
		Status:            "pending",
		OtpHash:           string(otpHash),
		OtpExpiresAt:      time.Now().Add(-1 * time.Minute),
		VerificationToken: "tok-exp",
		TokenExpiresAt:    time.Now().Add(10 * time.Minute),
	})
	h := NewVerificationHandler(database, &fakeEmailClient{})
	router := gin.New()
	router.POST("/verify", h.Verify)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewBufferString(`{"token":"tok-exp","otp":"123456","nim":"1"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired OTP, got %d", w.Code)
	}
}

func TestRequestVerificationRateLimitReturns429(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))
	database.SaveAlumni(context.Background(), &models.Alumni{ID: "u-rv", Email: "rv@gmail.com", Role: "member", Status: "pending"})
	h := NewVerificationHandler(database, &fakeEmailClient{})
	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.POST("/request-verification", h.RequestVerification)

	tok, _ := GenerateJWT(&models.Alumni{ID: "u-rv", Email: "rv@gmail.com", Role: "member", Status: "pending"})
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/request-verification", bytes.NewBufferString(`{"email_ntust":"rv@mail.ntust.edu.tw"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("attempt %d: expected 200, got %d", i+1, w.Code)
		}
	}
	w6 := httptest.NewRecorder()
	req6, _ := http.NewRequest(http.MethodPost, "/request-verification", bytes.NewBufferString(`{"email_ntust":"rv@mail.ntust.edu.tw"}`))
	req6.Header.Set("Content-Type", "application/json")
	req6.Header.Set("Authorization", "Bearer "+tok)
	router.ServeHTTP(w6, req6)
	if w6.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on 6th request-verification, got %d", w6.Code)
	}
}

func TestVerificationSMSFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:     "u-sms",
		Nama:   "Budi SMS",
		Email:  "sms@gmail.com",
		Role:   "member",
		Status: "pending",
	})

	emailClient := &fakeEmailClient{}
	h := NewVerificationHandler(database, emailClient)
	router := gin.New()
	router.GET("/verify/status", h.VerificationStatus)
	router.POST("/verify", h.Verify)
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.POST("/request-verification", h.RequestVerification)

	tok, _ := GenerateJWT(&models.Alumni{ID: "u-sms", Email: "sms@gmail.com", Role: "member", Status: "pending"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/request-verification", bytes.NewBufferString(`{"channel":"sms","phone":"0812-3456-789"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if emailClient.sentChannel != "sms" || emailClient.sentPhone != "628123456789" {
		t.Fatalf("expected sms to 628123456789, got channel=%s phone=%s", emailClient.sentChannel, emailClient.sentPhone)
	}

	var reqResp struct {
		Data map[string]string `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &reqResp)
	vToken := reqResp.Data["verification_token"]

	// Status endpoint returns active session for SMS
	wSt := httptest.NewRecorder()
	reqSt, _ := http.NewRequest(http.MethodGet, "/verify/status?token="+vToken, nil)
	router.ServeHTTP(wSt, reqSt)
	if wSt.Code != http.StatusOK {
		t.Fatalf("expected 200 on status, got %d: %s", wSt.Code, wSt.Body.String())
	}
	var stResp struct {
		Data struct {
			Valid   bool   `json:"valid"`
			Channel string `json:"channel"`
			Target  string `json:"target"`
		} `json:"data"`
	}
	json.Unmarshal(wSt.Body.Bytes(), &stResp)
	if !stResp.Data.Valid || stResp.Data.Channel != "sms" {
		t.Errorf("unexpected status response: %+v", stResp.Data)
	}

	// Verify OTP
	vBody, _ := json.Marshal(map[string]string{"token": vToken, "otp": emailClient.sentOTP, "nim": "M112"})
	wVer := httptest.NewRecorder()
	reqVer, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewReader(vBody))
	reqVer.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wVer, reqVer)
	if wVer.Code != http.StatusOK {
		t.Fatalf("expected 200 on verify, got %d: %s", wVer.Code, wVer.Body.String())
	}
}

func TestVerificationWhatsAppFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:     "u-wa",
		Nama:   "Budi WA",
		Email:  "wa@gmail.com",
		Nim:    "M109001",
		Role:   "member",
		Status: "pending",
	})

	emailClient := &fakeEmailClient{}
	h := NewVerificationHandler(database, emailClient)
	router := gin.New()
	router.GET("/verify/status", h.VerificationStatus)
	router.POST("/verify", h.Verify)
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.POST("/request-verification", h.RequestVerification)

	tok, _ := GenerateJWT(&models.Alumni{ID: "u-wa", Email: "wa@gmail.com", Role: "member", Status: "pending"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/request-verification", bytes.NewBufferString(`{"channel":"whatsapp","phone":"+62 812 9988 7766"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if emailClient.sentChannel != "whatsapp" || emailClient.sentPhone != "6281299887766" {
		t.Fatalf("expected whatsapp to 6281299887766, got channel=%s phone=%s", emailClient.sentChannel, emailClient.sentPhone)
	}

	var reqResp struct {
		Data map[string]string `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &reqResp)
	vToken := reqResp.Data["verification_token"]

	// Verify with empty NIM should preserve existing NIM
	vBody, _ := json.Marshal(map[string]string{"token": vToken, "otp": emailClient.sentOTP})
	wVer := httptest.NewRecorder()
	reqVer, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewReader(vBody))
	reqVer.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wVer, reqVer)
	if wVer.Code != http.StatusOK {
		t.Fatalf("expected 200 on verify, got %d: %s", wVer.Code, wVer.Body.String())
	}
	updated, _ := database.GetAlumni(context.Background(), "u-wa")
	if updated.Nim != "M109001" || updated.Status != "verified" {
		t.Errorf("expected preserved NIM M109001 and verified status, got %+v", updated)
	}
}

func TestVerificationStatusExpiredAndNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:                "u-gone",
		VerificationToken: "tok-gone",
		TokenExpiresAt:    time.Now().Add(-1 * time.Minute),
		OtpExpiresAt:      time.Now().Add(-1 * time.Minute),
	})

	h := NewVerificationHandler(database, &fakeEmailClient{})
	router := gin.New()
	router.GET("/verify/status", h.VerificationStatus)

	w404 := httptest.NewRecorder()
	req404, _ := http.NewRequest(http.MethodGet, "/verify/status?token=missing", nil)
	router.ServeHTTP(w404, req404)
	if w404.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown token, got %d", w404.Code)
	}

	w410 := httptest.NewRecorder()
	req410, _ := http.NewRequest(http.MethodGet, "/verify/status?token=tok-gone", nil)
	router.ServeHTTP(w410, req410)
	if w410.Code != http.StatusGone {
		t.Errorf("expected 410 for expired token, got %d", w410.Code)
	}
}

func TestVerificationRejectsCerdasClubEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:     "u-cerdas",
		Nama:   "Budi Cerdas",
		Email:  "budi@gmail.com",
		Role:   "member",
		Status: "pending",
	})

	emailClient := &fakeEmailClient{}
	h := NewVerificationHandler(database, emailClient)
	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.POST("/request-verification", h.RequestVerification)

	tok, _ := GenerateJWT(&models.Alumni{ID: "u-cerdas", Email: "budi@gmail.com", Role: "member", Status: "pending"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/request-verification", bytes.NewBufferString(`{"channel":"email","email_ntust":"budi@cerdas.club"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for @cerdas.club user email, got %d: %s", w.Code, w.Body.String())
	}
	if emailClient.callCount != 0 {
		t.Errorf("expected 0 emails sent, got %d", emailClient.callCount)
	}
}

type failingEmailClient struct{}

func (f *failingEmailClient) SendVerificationOTP(to, otp, token string) error {
	return errors.New("smtp connection timeout: internal-host")
}

func TestRequestVerificationRejectsAlreadyVerifiedAndRollsBackOnError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveAlumni(context.Background(), &models.Alumni{ID: "u-ver", Email: "v@gmail.com", Role: "super_admin", Status: "verified"})
	database.SaveAlumni(context.Background(), &models.Alumni{ID: "u-pend", Email: "p@gmail.com", Role: "member", Status: "pending"})

	h := NewVerificationHandler(database, &failingEmailClient{})
	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.POST("/request-verification", h.RequestVerification)

	// Already verified user cannot mint fresh verification token
	verTok, _ := GenerateJWT(&models.Alumni{ID: "u-ver", Email: "v@gmail.com", Role: "super_admin", Status: "verified"})
	w1 := httptest.NewRecorder()
	r1, _ := http.NewRequest(http.MethodPost, "/request-verification", bytes.NewBufferString(`{"channel":"email","email_ntust":"admin@mail.ntust.edu.tw"}`))
	r1.Header.Set("Content-Type", "application/json")
	r1.Header.Set("Authorization", "Bearer "+verTok)
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for already-verified account, got %d: %s", w1.Code, w1.Body.String())
	}

	// Failed email send rolls back OTP & token in DB and sanitizes error message
	pendTok, _ := GenerateJWT(&models.Alumni{ID: "u-pend", Email: "p@gmail.com", Role: "member", Status: "pending"})
	w2 := httptest.NewRecorder()
	r2, _ := http.NewRequest(http.MethodPost, "/request-verification", bytes.NewBufferString(`{"channel":"email","email_ntust":"pend@mail.ntust.edu.tw"}`))
	r2.Header.Set("Content-Type", "application/json")
	r2.Header.Set("Authorization", "Bearer "+pendTok)
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on delivery error, got %d", w2.Code)
	}
	if bytes.Contains(w2.Body.Bytes(), []byte("internal-host")) {
		t.Errorf("expected sanitized error message, got %s", w2.Body.String())
	}
	stored, _ := database.GetAlumni(context.Background(), "u-pend")
	if stored.VerificationToken != "" || stored.OtpHash != "" {
		t.Errorf("expected failed delivery to roll back verification token & OTP, got token=%q", stored.VerificationToken)
	}
}

func TestVerificationRejectsMalformedEmails(t *testing.T) {
	domains := []string{"@mail.ntust.edu.tw"}
	for _, bad := range []string{"@mail.ntust.edu.tw", "a@b@mail.ntust.edu.tw", "bad\r\n@mail.ntust.edu.tw", "Name <a@mail.ntust.edu.tw>"} {
		if isAllowedVerificationEmail(bad, domains) {
			t.Errorf("expected malformed verification email %q to be rejected", bad)
		}
	}
	if !isAllowedVerificationEmail("m11015001@mail.ntust.edu.tw", domains) {
		t.Error("expected valid @mail.ntust.edu.tw address to be accepted")
	}
}

func TestVerifyRejectsEmptyTokenOrOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	h := NewVerificationHandler(database, &fakeEmailClient{})
	router := gin.New()
	router.POST("/verify", h.Verify)

	for _, payload := range []string{`{"token":"","otp":"123456"}`, `{"token":"abc","otp":""}`} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for payload %s, got %d", payload, w.Code)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("expected Cache-Control: no-store, got %q", w.Header().Get("Cache-Control"))
		}
	}
}
