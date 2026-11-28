package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var JwtSecret []byte

func InitJWT(secret string) {
	JwtSecret = []byte(secret)
}

// AuthMiddleware memverifikasi JWT token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak ditemukan"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Format token tidak valid"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return JwtSecret, nil
		}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal parsing claims"})
			c.Abort()
			return
		}

		// Tolak token selain sesi portal (misalnya token_use=oauth_access tidak boleh mengakses API portal)
		if tokenUse, exists := claims["token_use"]; exists {
			if useStr, ok := tokenUse.(string); !ok || useStr != "session" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Jenis token tidak diizinkan"})
				c.Abort()
				return
			}
		}

		userID, ok := claims["user_id"].(string)
		if !ok || strings.TrimSpace(userID) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak memiliki identitas valid"})
			c.Abort()
			return
		}
		userRole, _ := claims["role"].(string)
		userEmail, _ := claims["email"].(string)
		userStatus, _ := claims["status"].(string)

		c.Set("user_id", userID)
		c.Set("user_role", userRole)
		c.Set("user_email", userEmail)
		c.Set("user_status", userStatus)
		c.Next()
	}
}

// RequireRole memeriksa apakah user memiliki role yang diizinkan
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role tidak ditemukan"})
			c.Abort()
			return
		}

		userRole, ok := role.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role tidak valid"})
			c.Abort()
			return
		}

		for _, allowed := range allowedRoles {
			if userRole == allowed {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak"})
		c.Abort()
	}
}
