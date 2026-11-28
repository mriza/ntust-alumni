package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"ntust-alumni/backend/internal/db"
	"ntust-alumni/backend/internal/models"
)

// EmailSender adalah interface untuk mengirim email verifikasi OTP.
type EmailSender interface {
	SendVerificationOTP(to, otp, token string) error
}

// SMSSender adalah interface opsional untuk mengirim OTP via SMS.
type SMSSender interface {
	SendVerificationSMS(phone, otp string) error
}

// WhatsAppSender adalah interface opsional untuk mengirim OTP via WhatsApp.
type WhatsAppSender interface {
	SendVerificationWhatsApp(phone, otp string) error
}

// VerificationHandler menangani alur verifikasi alumni via Email, SMS, atau WhatsApp OTP.
type VerificationHandler struct {
	database    db.Store
	email       EmailSender
	rateLimiter *RateLimiter
}

func NewVerificationHandler(database db.Store, email EmailSender) *VerificationHandler {
	return &VerificationHandler{
		database:    database,
		email:       email,
		rateLimiter: NewRateLimiter(),
	}
}

func normalizePhone(raw string) (string, error) {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if strings.HasPrefix(digits, "0") {
		digits = "62" + digits[1:]
	}
	if len(digits) < 9 || len(digits) > 15 {
		return "", errors.New("Nomor telepon/WhatsApp tidak valid (gunakan 9-15 digit dengan kode negara, mis. 0812... atau 62812...)")
	}
	return digits, nil
}

func maskTarget(channel, target string) string {
	if target == "" {
		return ""
	}
	if channel == "email" || strings.Contains(target, "@") {
		parts := strings.SplitN(target, "@", 2)
		if len(parts) != 2 {
			return target
		}
		local := parts[0]
		if len(local) <= 2 {
			return local[:1] + "***@" + parts[1]
		}
		return local[:2] + "***@" + parts[1]
	}
	if len(target) <= 6 {
		return "+" + target
	}
	return "+" + target[:4] + "****" + target[len(target)-3:]
}

func allowedVerificationDomains() []string {
	raw := strings.TrimSpace(os.Getenv("VERIFICATION_ALLOWED_DOMAINS"))
	if raw == "" {
		return []string{"@mail.ntust.edu.tw"}
	}
	var domains []string
	for _, part := range strings.Split(raw, ",") {
		d := strings.ToLower(strings.TrimSpace(part))
		if d == "" {
			continue
		}
		if !strings.HasPrefix(d, "@") {
			d = "@" + d
		}
		domains = append(domains, d)
	}
	if len(domains) == 0 {
		return []string{"@mail.ntust.edu.tw"}
	}
	return domains
}

func isAllowedVerificationEmail(emailAddr string, domains []string) bool {
	lower := strings.ToLower(strings.TrimSpace(emailAddr))
	for _, d := range domains {
		if strings.HasSuffix(lower, d) {
			return true
		}
	}
	return false
}

// RequestVerification meminta pengiriman tautan &/atau OTP verifikasi lewat email, sms, atau whatsapp.
func (h *VerificationHandler) RequestVerification(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// Rate limit: maksimal 5 permintaan verifikasi per user per 15 menit
	if allowed, _ := h.rateLimiter.CheckAndIncrement("req-verify:"+userID.(string), 5); !allowed {
		c.JSON(http.StatusTooManyRequests, models.ErrorResponse{Error: "Terlalu banyak permintaan verifikasi. Coba lagi dalam 15 menit"})
		return
	}

	var req requestVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid: " + err.Error()})
		return
	}

	channel := strings.ToLower(strings.TrimSpace(req.Channel))
	if channel == "" {
		channel = "email"
	}

	var target string
	switch channel {
	case "email":
		target = strings.TrimSpace(req.EmailNtust)
		domains := allowedVerificationDomains()
		if !isAllowedVerificationEmail(target, domains) {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Email verifikasi harus berdomain " + strings.Join(domains, " atau ")})
			return
		}
	case "sms", "whatsapp":
		normalized, err := normalizePhone(req.Phone)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
			return
		}
		target = normalized
	default:
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Channel verifikasi tidak valid (pilih: email, sms, whatsapp)"})
		return
	}

	alumni, err := h.database.GetAlumni(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Alumni tidak ditemukan"})
		return
	}

	otp := generateOTP()
	token := generateToken()
	expiresAt := time.Now().Add(15 * time.Minute)

	otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal generate OTP"})
		return
	}

	alumni.OtpHash = string(otpHash)
	alumni.OtpExpiresAt = expiresAt
	alumni.VerificationToken = token
	alumni.TokenExpiresAt = expiresAt
	alumni.VerificationChannel = channel
	alumni.VerificationTarget = target

	switch channel {
	case "email":
		alumni.EmailNtust = target
	case "sms":
		if alumni.NoTelp == "" {
			alumni.NoTelp = target
		}
	case "whatsapp":
		if alumni.NoWA == "" {
			alumni.NoWA = target
		}
	}

	if err := h.database.UpdateAlumni(c.Request.Context(), alumni); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal menyimpan data verifikasi"})
		return
	}

	var sendErr error
	var msg string
	switch channel {
	case "email":
		sendErr = h.email.SendVerificationOTP(target, otp, token)
		msg = "Tautan verifikasi unik dan kode OTP telah dikirim ke email NTUST Anda"
	case "sms":
		smsSender, ok := h.email.(SMSSender)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Pengirim SMS belum tersedia"})
			return
		}
		sendErr = smsSender.SendVerificationSMS(target, otp)
		msg = "Kode OTP verifikasi telah dikirim melalui SMS"
	case "whatsapp":
		waSender, ok := h.email.(WhatsAppSender)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Pengirim WhatsApp belum tersedia"})
			return
		}
		sendErr = waSender.SendVerificationWhatsApp(target, otp)
		msg = "Kode OTP verifikasi telah dikirim melalui WhatsApp"
	}

	if sendErr != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal mengirim verifikasi: " + sendErr.Error()})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: msg,
		Data: gin.H{
			"channel":            channel,
			"target":             maskTarget(channel, target),
			"email":              alumni.EmailNtust,
			"expires_in":         "15 menit",
			"expires_at":         expiresAt.UTC().Format(time.RFC3339),
			"verification_token": token,
		},
	})
}

// VerificationStatus memeriksa validitas dan masa berlaku tautan/sesi verifikasi.
func (h *VerificationHandler) VerificationStatus(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Token verifikasi wajib disertakan"})
		return
	}

	alumni, err := h.database.FindAlumniByVerificationToken(c.Request.Context(), token)
	if err != nil || alumni == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Tautan atau sesi verifikasi tidak ditemukan"})
		return
	}

	if time.Now().After(alumni.TokenExpiresAt) || time.Now().After(alumni.OtpExpiresAt) {
		c.JSON(http.StatusGone, models.ErrorResponse{Error: "Tautan atau kode OTP verifikasi telah kadaluarsa"})
		return
	}

	channel := alumni.VerificationChannel
	if channel == "" {
		channel = "email"
	}
	target := alumni.VerificationTarget
	if target == "" {
		target = alumni.EmailNtust
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Sesi verifikasi aktif",
		Data: gin.H{
			"valid":      true,
			"channel":    channel,
			"target":     maskTarget(channel, target),
			"expires_at": alumni.TokenExpiresAt.UTC().Format(time.RFC3339),
		},
	})
}

// Verify memverifikasi alumni dengan token, OTP, dan NIM (opsional jika NIM sudah ada).
func (h *VerificationHandler) Verify(c *gin.Context) {
	var req verifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid: " + err.Error()})
		return
	}

	alumni, err := h.database.FindAlumniByVerificationToken(c.Request.Context(), strings.TrimSpace(req.Token))
	if err != nil || alumni == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Token verifikasi tidak valid"})
		return
	}

	// Rate limit: maksimal 5 attempt per token, lalu locked 15 menit
	key := "verify-token:" + alumni.ID
	allowed, lockUntil := h.rateLimiter.CheckAndIncrement(key, 5)
	if !allowed {
		returnLockDuration := time.Until(lockUntil).Minutes()
		c.JSON(http.StatusTooManyRequests, models.ErrorResponse{Error: "Terlalu banyak percobaan. Coba lagi dalam " + fmt.Sprintf("%.0f", returnLockDuration) + " menit"})
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

	if err := bcrypt.CompareHashAndPassword([]byte(alumni.OtpHash), []byte(strings.TrimSpace(req.OTP))); err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Kode OTP salah"})
		return
	}

	alumni.Status = "verified"
	if nim := strings.TrimSpace(req.NIM); nim != "" {
		alumni.Nim = nim
	}
	alumni.OtpHash = ""
	alumni.OtpExpiresAt = time.Time{}
	alumni.VerificationToken = ""
	alumni.TokenExpiresAt = time.Time{}
	alumni.VerificationChannel = ""
	alumni.VerificationTarget = ""

	if err := h.database.UpdateAlumni(c.Request.Context(), alumni); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal memperbarui status verifikasi"})
		return
	}

	respData := gin.H{
		"status": "verified",
	}
	if len(jwtKey) > 0 {
		if newToken, err := GenerateJWT(alumni); err == nil {
			respData["token"] = newToken
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Verifikasi berhasil. Akun Anda sudah aktif.",
		Data:    respData,
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
	Channel    string `json:"channel"`
	EmailNtust string `json:"email_ntust"`
	Phone      string `json:"phone"`
}

type verifyRequest struct {
	Token string `json:"token"`
	OTP   string `json:"otp"`
	NIM   string `json:"nim"`
}
