package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"ntust-alumni/backend/internal/db"
	"ntust-alumni/backend/internal/models"
)

type AuthHandler struct {
	database    db.Store
	rateLimiter *RateLimiter
}

func NewAuthHandler(database db.Store) *AuthHandler {
	return &AuthHandler{database: database, rateLimiter: NewRateLimiter()}
}

// Register pendaftaran alumni baru (email bebas, status pending)
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid: " + err.Error()})
		return
	}

	// Cek apakah email sudah terdaftar — kembalikan generic response agar
	// tidak bisa diakses dari luar untuk enumerasi email
	existing, _ := h.database.FindAlumniByEmail(c.Request.Context(), req.Email)
	if existing != nil {
		c.JSON(http.StatusOK, models.SuccessResponse{Message: "Registration request will be processed. Please check your inbox."})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal hash password"})
		return
	}

	alumni := &models.Alumni{
		Nama:          req.Nama,
		Email:         req.Email,
		Nim:           req.Nim,
		Jurusan:       req.Jurusan,
		Status:        "pending",
		Role:          "member",
		PasswordHash:  string(hashedPassword),
	}

	if err := h.database.SaveAlumni(c.Request.Context(), alumni); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal menyimpan data: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse{
		Message: "Pendaftaran berhasil. Silakan login.",
		Data: gin.H{
			"id":    alumni.ID,
			"email": alumni.Email,
			"role":  alumni.Role,
		},
	})
}

// Login untuk alumni dan admin
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid"})
		return
	}

	// Rate limit: 5 per email/IP per 10 min, mengembalikan 429 setelah limit
	ip := c.ClientIP()
	if allowed, _ := h.rateLimiter.CheckAndIncrement("login-ip:"+ip, 10); !allowed {
		c.JSON(http.StatusTooManyRequests, models.ErrorResponse{Error: "Terlalu banyak login. Coba lagi dalam 10 menit"})
		return
	}
	if allowed, _ := h.rateLimiter.CheckAndIncrement("login-email:"+req.Email, 5); !allowed {
		c.JSON(http.StatusTooManyRequests, models.ErrorResponse{Error: "Akun terlalu sering login. Coba lagi dalam 5 menit"})
		return
	}

	alumni, err := h.database.FindAlumniByEmail(c.Request.Context(), req.Email)
	if err != nil || alumni == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Email atau password salah"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(alumni.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Email atau password salah"})
		return
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  alumni.ID,
		"email":    alumni.Email,
		"role":     alumni.Role,
		"status":   alumni.Status,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal generate token"})
		return
	}

	c.JSON(http.StatusOK, models.TokenResponse{
		Token: tokenString,
		Role:  alumni.Role,
		ID:    alumni.ID,
		Nama:  alumni.Nama,
	})
}

// Me mengembalikan data user yang sedang login
func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	alumni, err := h.database.GetAlumni(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Alumni tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          alumni.ID,
		"no_telp":     alumni.NoTelp,
		"nama":        alumni.Nama,
		"email":       alumni.Email,
		"nim":         alumni.Nim,
		"jurusan":     alumni.Jurusan,
		"wilayah":     alumni.Wilayah,
		"role":        alumni.Role,
	})
}

var jwtKey []byte

func SetJWTKey(key []byte) {
	jwtKey = key
}
