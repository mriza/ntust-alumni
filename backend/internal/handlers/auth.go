package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"ntust-alumni/backend/internal/db"
	"ntust-alumni/backend/internal/models"
)

type AuthHandler struct {
	database *db.CouchDB
}

func NewAuthHandler(database *db.CouchDB) *AuthHandler {
	return &AuthHandler{database: database}
}

// Register pendaftaran alumni baru (hanya email @ntust.edu.tw)
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid: " + err.Error()})
		return
	}

	// Validasi email domain @ntust.edu.tw
	if !strings.HasSuffix(strings.ToLower(req.Email), "@ntust.edu.tw") {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Hanya email @ntust.edu.tw yang diperbolehkan"})
		return
	}

	// Cek apakah email sudah terdaftar
	existing, _ := h.database.FindAlumniByEmail(c.Request.Context(), req.Email)
	if existing != nil {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Email sudah terdaftar"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal hash password"})
		return
	}

	alumni := &models.Alumni{
		Nama:       req.Nama,
		Email:      req.Email,
		Nim:        req.Nim,
		Jurusan:    req.Jurusan,
		TahunLulus: req.TahunLulus,
		Wilayah:    req.Wilayah,
		Telepon:    req.Telepon,
		Role:       "member",
		Password:   string(hashedPassword),
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

	alumni, err := h.database.FindAlumniByEmail(c.Request.Context(), req.Email)
	if err != nil || alumni == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Email atau password salah"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(alumni.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Email atau password salah"})
		return
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  alumni.ID,
		"email":    alumni.Email,
		"role":     alumni.Role,
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
		"nama":        alumni.Nama,
		"email":       alumni.Email,
		"nim":         alumni.Nim,
		"jurusan":     alumni.Jurusan,
		"tahun_lulus": alumni.TahunLulus,
		"wilayah":     alumni.Wilayah,
		"telepon":     alumni.Telepon,
		"role":        alumni.Role,
	})
}

var jwtKey []byte

func SetJWTKey(key []byte) {
	jwtKey = key
}
