package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BrevoClient mengirim email verifikasi melalui Brevo API v3.
type BrevoClient struct {
	apiKey    string
	fromEmail string
	fromName  string
	baseURL   string
	client    *http.Client
}

func NewBrevoClient(apiKey, fromEmail, fromName string) *BrevoClient {
	return &BrevoClient{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		fromName:  fromName,
		baseURL:   "https://api.brevo.com",
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

type sendEmailRequest struct {
	To      []map[string]string `json:"to"`
	Sender  map[string]string   `json:"sender"`
	Subject string              `json:"subject"`
	HtmlContent string           `json:"htmlContent"`
}

type sendEmailResponse struct {
	MessageID string `json:"messageId"`
}

// SendVerificationOTP mengirim email berisi OTP dan link verifikasi ke alamat tujuan.
func (c *BrevoClient) SendVerificationOTP(to, otp, token string) error {
	verifyURL := fmt.Sprintf("%s/verify?token=%s", "https://alumni.ntust-indonesia.id", token)

	htmlContent := fmt.Sprintf(`
		<h2>Verifikasi Akun Alumni NTUST</h2>
		<p>Kode OTP Anda: <strong>%s</strong></p>
		<p>Kode ini berlaku selama 15 menit.</p>
		<p>Atau kunjungi halaman verifikasi: <a href="%s">Verifikasi Sekarang</a></p>
		<p>Di halaman verifikasi, masukkan OTP dan NIM Anda.</p>
	`, otp, verifyURL)

	reqBody := sendEmailRequest{
		To: []map[string]string{
			{"email": to},
		},
		Sender: map[string]string{
			"email": c.fromEmail,
			"name":  c.fromName,
		},
		Subject:       "Kode Verifikasi Alumni NTUST Indonesia",
		HtmlContent:   htmlContent,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("gagal marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/v3/smtp/email", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("gagal membuat request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("gagal mengirim email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Brevo API error: status %d, body %s", resp.StatusCode, string(respBody))
	}

	var result sendEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("gagal decode response: %w", err)
	}

	_ = result.MessageID
	return nil
}
