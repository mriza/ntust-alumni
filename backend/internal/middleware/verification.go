package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireVerified memeriksa apakah status alumni adalah "verified".
// Digunakan untuk endpoint yang hanya bisa diakses alumni terverifikasi.
func RequireVerified() gin.HandlerFunc {
	return func(c *gin.Context) {
		status, exists := c.Get("user_status")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Status alumni tidak ditemukan"})
			c.Abort()
			return
		}

		if status.(string) != "verified" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Akun belum terverifikasi. Silakan verifikasi email dan NIM terlebih dahulu"})
			c.Abort()
			return
		}

		c.Next()
	}
}
