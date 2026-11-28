package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	InitJWT("test-secret")

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Token tidak ditemukan") {
		t.Errorf("expected 'Token tidak ditemukan' in body, got %s", w.Body.String())
	}
}

func TestAuthMiddlewareRejectsInvalidFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	InitJWT("test-secret")

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat token123")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddlewareAcceptsValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	InitJWT("test-secret")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user1",
		"email":   "test@example.com",
		"role":    "member",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("user_role")
		userEmail, _ := c.Get("user_email")
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"role":    userRole,
			"email":   userEmail,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "user1") {
		t.Errorf("expected user_id in response, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "member") {
		t.Errorf("expected role member in response, got %s", w.Body.String())
	}
}

func TestRequireRoleAllowsAllowedRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	InitJWT("test-secret")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "admin1",
		"role":    "super_admin",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	router := gin.New()
	router.Use(AuthMiddleware())
	router.Use(RequireRole("super_admin"))
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequireRoleRejectsDisallowedRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	InitJWT("test-secret")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "member1",
		"role":    "member",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	router := gin.New()
	router.Use(AuthMiddleware())
	router.Use(RequireRole("super_admin"))
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestAuthMiddlewareRejectsOAuthAccessAndMissingExp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	InitJWT("test-secret")

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	oauthTok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   "admin1",
		"token_use": "oauth_access",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	s1, _ := oauthTok.SignedString([]byte("test-secret"))
	w1 := httptest.NewRecorder()
	r1, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r1.Header.Set("Authorization", "Bearer "+s1)
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for oauth_access token, got %d", w1.Code)
	}

	noExpTok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "admin1",
		"role":    "super_admin",
	})
	s2, _ := noExpTok.SignedString([]byte("test-secret"))
	w2 := httptest.NewRecorder()
	r2, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r2.Header.Set("Authorization", "Bearer "+s2)
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing exp claim, got %d", w2.Code)
	}
}

func TestRequireVerifiedHandlesNonStringStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_status", 12345)
		c.Next()
	}, RequireVerified())
	router.GET("/v", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/v", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without panic, got %d", w.Code)
	}
}
