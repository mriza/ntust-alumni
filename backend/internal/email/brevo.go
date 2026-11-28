package email

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ResolveAPIKey mengekstrak Brevo api-key dari raw key (xkeysib-...) atau MCP token base64 JSON.
func ResolveAPIKey(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.HasPrefix(trimmed, "xkeysib-") {
		return trimmed
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding} {
		if decoded, err := enc.DecodeString(trimmed); err == nil {
			var payload struct {
				APIKey string `json:"api_key"`
			}
			if json.Unmarshal(decoded, &payload) == nil && strings.TrimSpace(payload.APIKey) != "" {
				return strings.TrimSpace(payload.APIKey)
			}
		}
	}
	return trimmed
}

// BrevoClient mengirim email, SMS, dan WhatsApp verifikasi melalui Brevo API v3.
type BrevoClient struct {
	apiKey             string
	fromEmail          string
	fromName           string
	smsSender          string
	whatsappSender     string
	whatsappTemplateID int64
	baseURL            string
	client             *http.Client
}

func NewBrevoClient(apiKey, fromEmail, fromName string) *BrevoClient {
	fromEmail = strings.TrimSpace(fromEmail)
	if fromEmail == "" {
		fromEmail = "ntust-alumni-verification@cerdas.club"
	}
	fromName = strings.TrimSpace(fromName)
	if fromName == "" {
		fromName = "Alumni NTUST Indonesia"
	}
	return &BrevoClient{
		apiKey:    ResolveAPIKey(apiKey),
		fromEmail: fromEmail,
		fromName:  fromName,
		baseURL:   "https://api.brevo.com",
		client: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// ConfigureMessaging mengatur pengirim SMS dan WhatsApp Brevo.
func (c *BrevoClient) ConfigureMessaging(smsSender, whatsappSender string, whatsappTemplateID int64) {
	c.smsSender = strings.TrimSpace(smsSender)
	c.whatsappSender = strings.TrimSpace(whatsappSender)
	c.whatsappTemplateID = whatsappTemplateID
}

type sendEmailRequest struct {
	To          []map[string]string `json:"to"`
	Sender      map[string]string   `json:"sender"`
	Subject     string              `json:"subject"`
	HtmlContent string              `json:"htmlContent"`
}

type sendEmailResponse struct {
	MessageID string `json:"messageId"`
}

func (c *BrevoClient) postBrevoJSON(path string, payload interface{}) error {
	if c.apiKey == "" {
		return errors.New("BREVO_API_KEY belum dikonfigurasi")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gagal marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("gagal membuat request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("gagal mengirim pesan via Brevo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Brevo API error: status %d, body %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SendVerificationOTP mengirim email berisi OTP dan link unik verifikasi ke alamat tujuan.
func (c *BrevoClient) SendVerificationOTP(to, otp, token string) error {
	frontendURL := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/")
	if frontendURL == "" {
		frontendURL = "http://localhost:8081"
	}
	verifyURL := fmt.Sprintf("%s/#/verify/confirm?token=%s&channel=email", frontendURL, url.QueryEscape(token))
	safeOTP := html.EscapeString(otp)
	safeVerifyURL := html.EscapeString(verifyURL)

	htmlContent := fmt.Sprintf(`
		<h2>Verifikasi Akun Alumni NTUST</h2>
		<p>Kode OTP Anda: <strong>%s</strong></p>
		<p>Tautan dan kode ini berlaku selama 15 menit.</p>
		<p>Klik tautan unik berikut untuk membuka halaman konfirmasi dan memasukkan kode OTP: <a href="%s">Konfirmasi Verifikasi Akun</a></p>
		<p>Atau salin tautan ini ke browser: %s</p>
	`, safeOTP, safeVerifyURL, safeVerifyURL)

	reqBody := sendEmailRequest{
		To: []map[string]string{
			{"email": to},
		},
		Sender: map[string]string{
			"email": c.fromEmail,
			"name":  c.fromName,
		},
		Subject:     "Tautan & Kode OTP Verifikasi Alumni NTUST Indonesia",
		HtmlContent: htmlContent,
	}

	if err := c.postBrevoJSON("/v3/smtp/email", reqBody); err != nil {
		return err
	}
	return nil
}

// SendVerificationSMS mengirim OTP verifikasi melalui Brevo Transactional SMS API.
func (c *BrevoClient) SendVerificationSMS(phone, otp string) error {
	sender := c.smsSender
	if sender == "" {
		sender = "NTUSTAlumni"
	}
	payload := map[string]interface{}{
		"sender":         sender,
		"recipient":      phone,
		"content":        fmt.Sprintf("Kode OTP verifikasi Alumni NTUST Anda: %s. Berlaku 15 menit.", otp),
		"type":           "transactional",
		"unicodeEnabled": true,
	}
	return c.postBrevoJSON("/v3/transactionalSMS/send", payload)
}

// SendVerificationWhatsApp mengirim OTP verifikasi melalui Brevo WhatsApp API.
func (c *BrevoClient) SendVerificationWhatsApp(phone, otp string) error {
	if c.whatsappSender == "" {
		return errors.New("BREVO_WHATSAPP_SENDER belum dikonfigurasi")
	}
	var payload map[string]interface{}
	if c.whatsappTemplateID > 0 {
		payload = map[string]interface{}{
			"senderNumber":   c.whatsappSender,
			"contactNumbers": []string{phone},
			"templateId":     c.whatsappTemplateID,
			"params":         map[string]string{"OTP": otp},
		}
	} else {
		payload = map[string]interface{}{
			"senderNumber":   c.whatsappSender,
			"contactNumbers": []string{phone},
			"text":           fmt.Sprintf("Kode OTP verifikasi Alumni NTUST Indonesia Anda: %s. Berlaku selama 15 menit.", otp),
		}
	}
	return c.postBrevoJSON("/v3/whatsapp/sendMessage", payload)
}
