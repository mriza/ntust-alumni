package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"ntust-alumni/backend/internal/db"
	"ntust-alumni/backend/internal/models"
)

// EmailSender adalah interface untuk mengirim email verifikasi OTP.
type EmailSender interface {
	SendVerificationOTP(to, otp, token string) error
}

// VerificationHandler menangani alur verifikasi alumni via OTP email.
type VerificationHandler struct {
	database db.Store
	email    EmailSender
}

func NewVerificationHandler(database db.Store, email EmailSender) *VerificationHandler {
	return &VerificationHandler{
		database: database,
		email:    email,
	}
}

// RequestVerification godoc
// @Summary Meminta verifikasi alumni
// @Description Alumni yang status pending meminta verifikasi dengan email @mail.ntust.edu.tw. Sistem generate OTP dan kirim via email.
// @Tags verification
// @Accept json
// @Produce json
// @Param request body requestVerificationRequest true "Email NTUST"
// @Success 200 {object} models.SuccessResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /request-verification [post]
func (h *VerificationHandler) RequestVerification(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req requestVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid: " + err.Error()})
		return
	}

	// Validasi email domain @mail.ntust.edu.tw
	if !strings.HasSuffix(strings.ToLower(req.EmailNtust), "@mail.ntust.edu.tw") {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Email verifikasi harus berdomain @mail.ntust.edu.tw"})
		return
	}

	alumni, err := h.database.GetAlumni(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Alumni tidak ditemukan"})
		return
	}

	// Generate OTP 6 digit
	otp := generateOTP()

	// Generate verification token
	token := generateToken()

	// Set expiry 15 menit
	expiresAt := time.Now().Add(15 * time.Minute)

	// Hash OTP
	otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal generate OTP"})
		return
	}

	// Update alumni record
	alumni.EmailNtust = req.EmailNtust
	alumni.OtpHash = string(otpHash)
	alumni.OtpExpiresAt = expiresAt
	alumni.VerificationToken = token
	alumni.TokenExpiresAt = expiresAt

	if err := h.database.UpdateAlumni(c.Request.Context(), alumni); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal menyimpan data verifikasi"})
		return
	}

	// Kirim email via email client
	if err := h.email.SendVerificationOTP(req.EmailNtust, otp, token); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal mengirim email verifikasi"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Kode verifikasi telah dikirim ke email Anda",
		Data: gin.H{
			"email":      req.EmailNtust,
			"expires_in": "15 menit",
		},
	})
}

// Verify godoc
// @Summary Verifikasi alumni dengan OTP dan NIM
// @Description Memverifikasi alumni dengan token, OTP, dan NIM. Jika berhasil, status berubah menjadi verified.
// @Tags verification
// @Accept json
// @Produce json
// @Param request body verifyRequest true "Token, OTP, dan NIM"
// @Success 200 {object} models.SuccessResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /verify [post]
func (h *VerificationHandler) Verify(c *gin.Context) {
	var req verifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid: " + err.Error()})
		return
	}

	alumni, err := h.database.FindAlumniByVerificationToken(c.Request.Context(), req.Token)
	if err != nil || alumni == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Token verifikasi tidak valid"})
		return
	}

	if time.Now().After(alumni.TokenExpiresAt) {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Token verifikasi telah kadaluarsa"})
		return
	}

	if time.Now().After(alumni.OtpExpiresAt) {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Kode OTP telah kadaluarsa"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(alumni.OtpHash), []byte(req.OTP)); err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Kode OTP salah"})
		return
	}

	alumni.Status = "verified"
	alumni.Nim = req.NIM
	alumni.OtpHash = ""
	alumni.OtpExpiresAt = time.Time{}
	alumni.VerificationToken = ""
	alumni.TokenExpiresAt = time.Time{}

	if err := h.database.UpdateAlumni(c.Request.Context(), alumni); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal memperbarui status verifikasi"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Verifikasi berhasil. Akun Anda sudah aktif.",
		Data: gin.H{
			"status": "verified",
		},
	})
}

func generateOTP() string {
	buf := make([]byte, 6)
	rand.Read(buf)
	otp := ""
	for _, b := range buf {
		otp += string(rune('0' + int(b)%10))
	}
	return otp
}

func generateToken() string {
	buf := make([]byte, 32)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}

type requestVerificationRequest struct {
	EmailNtust string `json:"email_ntust"`
}

type verifyRequest struct {
	Token string `json:"token"`
	OTP   string `json:"otp"`
	NIM   string `json:"nim"`
}
