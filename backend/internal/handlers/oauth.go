package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"ntust-alumni/backend/internal/db"
	"ntust-alumni/backend/internal/middleware"
	"ntust-alumni/backend/internal/models"
)

// OAuthConfig menyimpan konfigurasi OAuth2 provider untuk integrasi Apache Answer SSO.
type OAuthConfig struct {
	ClientID              string
	ClientSecret          string
	AllowedRedirectPrefix string
}

type authCodeEntry struct {
	UserID      string
	ClientID    string
	RedirectURI string
	ExpiresAt   time.Time
}

const maxActiveOAuthCodes = 1000

// OAuthHandler menyediakan endpoint OAuth2 Authorization Code Flow untuk Apache Answer connector-basic.
type OAuthHandler struct {
	database    db.Store
	cfg         OAuthConfig
	rateLimiter *RateLimiter
	mu          sync.Mutex
	codes       map[string]authCodeEntry
}

func NewOAuthHandler(database db.Store, cfg OAuthConfig) *OAuthHandler {
	if strings.TrimSpace(cfg.AllowedRedirectPrefix) == "" {
		cfg.AllowedRedirectPrefix = "http://localhost:9080/answer/api/v1/connector/"
	}
	return &OAuthHandler{
		database:    database,
		cfg:         cfg,
		rateLimiter: NewRateLimiter(),
		codes:       make(map[string]authCodeEntry),
	}
}

func isAllowedRedirectURI(rawRedirect, allowedPrefix string) bool {
	u, err := url.Parse(strings.TrimSpace(rawRedirect))
	p, errP := url.Parse(strings.TrimSpace(allowedPrefix))
	if err != nil || errP != nil || u.Scheme == "" || u.Host == "" || u.User != nil {
		return false
	}
	if u.Scheme != p.Scheme || u.Host != p.Host {
		return false
	}
	if strings.Contains(u.Path, "..") {
		return false
	}
	return strings.HasPrefix(u.Path, p.Path)
}

func constantTimeEqual(a, b string) bool {
	return len(a) > 0 && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

type oauthAuthorizeRequest struct {
	ClientID     string `json:"client_id"`
	RedirectURI  string `json:"redirect_uri"`
	State        string `json:"state"`
	ResponseType string `json:"response_type"`
	Scope        string `json:"scope"`
}

// Authorize menerbitkan authorization code sekali pakai (TTL 2 menit) bagi alumni verified.
func (h *OAuthHandler) Authorize(c *gin.Context) {
	if h.cfg.ClientID == "" || h.cfg.ClientSecret == "" {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "OAuth2 SSO belum dikonfigurasi"})
		return
	}

	var req oauthAuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data otorisasi tidak valid"})
		return
	}

	if !constantTimeEqual(strings.TrimSpace(req.ClientID), h.cfg.ClientID) {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "client_id tidak dikenal"})
		return
	}
	if req.ResponseType != "" && req.ResponseType != "code" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "response_type harus code"})
		return
	}
	if !isAllowedRedirectURI(req.RedirectURI, h.cfg.AllowedRedirectPrefix) {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "redirect_uri tidak diizinkan"})
		return
	}

	userIDVal, _ := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Sesi tidak valid"})
		return
	}
	if allowed, _ := h.rateLimiter.CheckAndIncrement("oauth-auth:"+userID, 30); !allowed {
		c.JSON(http.StatusTooManyRequests, models.ErrorResponse{Error: "Terlalu banyak permintaan otorisasi SSO"})
		return
	}
	alumni, err := h.database.GetAlumni(c.Request.Context(), userID)
	if err != nil || alumni == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Alumni tidak ditemukan"})
		return
	}
	if alumni.Status != "verified" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Akun belum terverifikasi"})
		return
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal membuat kode otorisasi"})
		return
	}
	code := hex.EncodeToString(buf)

	now := time.Now()
	h.mu.Lock()
	for k, v := range h.codes {
		if now.After(v.ExpiresAt) {
			delete(h.codes, k)
		}
	}
	if len(h.codes) >= maxActiveOAuthCodes {
		h.mu.Unlock()
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{Error: "Antrean otorisasi SSO penuh, coba lagi sebentar"})
		return
	}
	h.codes[code] = authCodeEntry{
		UserID:      alumni.ID,
		ClientID:    h.cfg.ClientID,
		RedirectURI: strings.TrimSpace(req.RedirectURI),
		ExpiresAt:   now.Add(2 * time.Minute),
	}
	h.mu.Unlock()

	parsedRedirect, _ := url.Parse(strings.TrimSpace(req.RedirectURI))
	q := parsedRedirect.Query()
	q.Set("code", code)
	if req.State != "" {
		q.Set("state", req.State)
	}
	parsedRedirect.RawQuery = q.Encode()

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Otorisasi SSO berhasil",
		Data: gin.H{
			"code":         code,
			"state":        req.State,
			"redirect_url": parsedRedirect.String(),
		},
	})
}

// Token menukar authorization code menjadi scoped access token JWT (berlaku 5 menit).
func (h *OAuthHandler) Token(c *gin.Context) {
	if allowed, _ := h.rateLimiter.CheckAndIncrement("oauth-token-ip:"+c.ClientIP(), 30); !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limit_exceeded"})
		return
	}
	if h.cfg.ClientID == "" || h.cfg.ClientSecret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	var grantType, code, redirectURI, clientID, clientSecret string
	if strings.Contains(strings.ToLower(c.ContentType()), "application/json") {
		var body struct {
			GrantType    string `json:"grant_type"`
			Code         string `json:"code"`
			RedirectURI  string `json:"redirect_uri"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}
		grantType, code, redirectURI = body.GrantType, body.Code, body.RedirectURI
		clientID, clientSecret = body.ClientID, body.ClientSecret
	} else {
		grantType = c.PostForm("grant_type")
		code = c.PostForm("code")
		redirectURI = c.PostForm("redirect_uri")
		clientID = c.PostForm("client_id")
		clientSecret = c.PostForm("client_secret")
	}

	if basicID, basicSecret, ok := c.Request.BasicAuth(); ok {
		clientID = basicID
		clientSecret = basicSecret
	}

	if !constantTimeEqual(clientID, h.cfg.ClientID) || !constantTimeEqual(clientSecret, h.cfg.ClientSecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	if grantType != "authorization_code" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}

	code = strings.TrimSpace(code)
	h.mu.Lock()
	entry, ok := h.codes[code]
	if ok {
		delete(h.codes, code)
	}
	h.mu.Unlock()

	if !ok || time.Now().After(entry.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
		return
	}
	if entry.RedirectURI != strings.TrimSpace(redirectURI) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
		return
	}

	alumni, err := h.database.GetAlumni(c.Request.Context(), entry.UserID)
	if err != nil || alumni == nil || alumni.Status != "verified" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access_denied"})
		return
	}

	signingKey := jwtKey
	if len(signingKey) == 0 {
		signingKey = middleware.JwtSecret
	}

	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   alumni.ID,
		"token_use": "oauth_access",
		"iat":       now.Unix(),
		"exp":       now.Add(5 * time.Minute).Unix(),
	})
	accessToken, err := tok.SignedString(signingKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   300,
		"scope":        "openid profile email",
	})
}

func makeAnswerUsername(emailAddr, id string) string {
	local := strings.SplitN(strings.ToLower(strings.TrimSpace(emailAddr)), "@", 2)[0]
	var b strings.Builder
	for _, r := range local {
		if (r >= 'a' && r <= 'z') || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		} else if r == '.' || r == '-' {
			b.WriteRune('_')
		}
	}
	base := strings.Trim(b.String(), "_")
	if base == "" {
		base = "alumni"
	}
	suffix := id
	if len(suffix) > 4 {
		suffix = suffix[len(suffix)-4:]
	}
	if len(base) > 25 {
		base = base[:25]
	}
	if suffix != "" {
		return base + "_" + suffix
	}
	return base
}

// UserInfo mengembalikan profil datar JSON untuk Apache Answer connector-basic.
func (h *OAuthHandler) UserInfo(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	signingKey := jwtKey
	if len(signingKey) == 0 {
		signingKey = middleware.JwtSecret
	}

	parsed, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return signingKey, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil || !parsed.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || claims["token_use"] != "oauth_access" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	userID, _ := claims["user_id"].(string)
	alumni, err := h.database.GetAlumni(c.Request.Context(), userID)
	if err != nil || alumni == nil || alumni.Status != "verified" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access_denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           alumni.ID,
		"email":        alumni.Email,
		"username":     makeAnswerUsername(alumni.Email, alumni.ID),
		"display_name": alumni.Nama,
		"avatar":       "",
	})
}
