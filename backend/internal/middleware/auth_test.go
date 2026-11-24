package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
		"user_id":  "user1",
		"email":    "test@example.com",
		"role":     "member",
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
