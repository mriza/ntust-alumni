package handlers

import (
	"log"
	"net/http"
	"net/mail"
	"strings"
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

func isValidEmailAddress(raw string) bool {
	if raw == "" || strings.ContainsAny(raw, "\r\n<> ") {
		return false
	}
	parsed, err := mail.ParseAddress(raw)
	if err != nil || !strings.EqualFold(parsed.Address, raw) {
		return false
	}
	at := strings.IndexByte(parsed.Address, '@')
	if at <= 0 {
		return false
	}
	domain := parsed.Address[at+1:]
	return strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}

// Register pendaftaran alumni baru (email bebas, status pending)
func (h *AuthHandler) Register(c *gin.Context) {
	ip := c.ClientIP()
	if allowed, _ := h.rateLimiter.CheckAndIncrement("register-ip:"+ip, 10); !allowed {
		c.JSON(http.StatusTooManyRequests, models.ErrorResponse{Error: "Terlalu banyak percobaan pendaftaran. Coba lagi nanti"})
		return
	}

	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data pendaftaran tidak valid"})
		return
	}

	req.Nama = strings.TrimSpace(req.Nama)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Nama == "" || !isValidEmailAddress(req.Email) || len(req.Password) < 6 || len(req.Password) > 72 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Nama wajib diisi, email harus valid, dan password harus antara 6 hingga 72 karakter"})
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
		Nama:         req.Nama,
		Email:        req.Email,
		Nim:          strings.TrimSpace(req.Nim),
		Jurusan:      strings.TrimSpace(req.Jurusan),
		Wilayah:      strings.TrimSpace(req.Wilayah),
		NoTelp:       strings.TrimSpace(req.Telepon),
		Status:       "pending",
		Role:         "member",
		PasswordHash: string(hashedPassword),
	}
	if req.TahunLulus > 0 {
		alumni.PendidikanNTUST = &models.PendidikanNTUST{
			TahunLulus: req.TahunLulus,
			Jurusan:    strings.TrimSpace(req.Jurusan),
		}
	}

	if err := h.database.SaveAlumni(c.Request.Context(), alumni); err != nil {
		log.Printf("SaveAlumni error on register: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal menyimpan data pendaftaran"})
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
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Rate limit: 5 per email / 10 per IP per 15 min window, mengembalikan 429 setelah limit
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

	tokenString, err := GenerateJWT(alumni)
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
	userIDVal, _ := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Sesi tidak valid"})
		return
	}
	alumni, err := h.database.GetAlumni(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Alumni tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Profil berhasil diambil",
		Data:    AlumniPublicFromAlumni(alumni),
	})
}

var jwtKey []byte

func SetJWTKey(key []byte) {
	jwtKey = key
}

// GenerateJWT membuat token JWT HS256 untuk alumni.
func GenerateJWT(alumni *models.Alumni) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   alumni.ID,
		"email":     alumni.Email,
		"role":      alumni.Role,
		"status":    alumni.Status,
		"token_use": "session",
		"iat":       now.Unix(),
		"exp":       now.Add(24 * time.Hour).Unix(),
	})
	return token.SignedString(jwtKey)
}
