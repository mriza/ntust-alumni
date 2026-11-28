package email

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockBrevoServer struct {
	lastRequest *http.Request
	lastBody    string
}

func (m *mockBrevoServer) handler(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	m.lastRequest = r
	m.lastBody = string(bodyBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"messageId": "msg-123"})
}

func TestBrevoClientSendVerificationOTP(t *testing.T) {
	mock := &mockBrevoServer{}
	srv := httptest.NewServer(http.HandlerFunc(mock.handler))
	defer srv.Close()

	client := NewBrevoClient("test-api-key", "noreply@test.com", "Test Sender")
	// Override the base URL to point to mock
	client.baseURL = srv.URL

	err := client.SendVerificationOTP("alumni@example.com", "123456", "token-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastRequest == nil {
		t.Fatal("expected request to be made")
	}

	if mock.lastRequest.Method != http.MethodPost {
		t.Errorf("expected POST, got %s", mock.lastRequest.Method)
	}
	if mock.lastRequest.URL.Path != "/v3/smtp/email" {
		t.Errorf("expected path /v3/smtp/email, got %s", mock.lastRequest.URL.Path)
	}
	if mock.lastRequest.Header.Get("Accept") != "application/json" {
		t.Errorf("expected Accept header application/json, got %s", mock.lastRequest.Header.Get("Accept"))
	}
	if mock.lastRequest.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type header application/json, got %s", mock.lastRequest.Header.Get("Content-Type"))
	}
	if mock.lastRequest.Header.Get("api-key") != "test-api-key" {
		t.Errorf("expected api-key header test-api-key, got %s", mock.lastRequest.Header.Get("api-key"))
	}

	// Parse body
	var body map[string]interface{}
	json.Unmarshal([]byte(mock.lastBody), &body)

	to, _ := body["to"].([]interface{})
	if len(to) != 1 {
		t.Fatalf("expected 1 recipient, got %d", len(to))
	}
	toMap, _ := to[0].(map[string]interface{})
	if toMap["email"] != "alumni@example.com" {
		t.Errorf("expected email alumni@example.com, got %v", toMap["email"])
	}

	sender, _ := body["sender"].(map[string]interface{})
	if sender["email"] != "noreply@test.com" {
		t.Errorf("expected sender email noreply@test.com, got %v", sender["email"])
	}
	if sender["name"] != "Test Sender" {
		t.Errorf("expected sender name Test Sender, got %v", sender["name"])
	}

	if !strings.Contains(body["subject"].(string), "Verifikasi") {
		t.Errorf("expected subject to contain Verifikasi, got %v", body["subject"])
	}

	html := body["htmlContent"].(string)
	if !strings.Contains(html, "123456") {
		t.Error("expected html content to contain OTP 123456")
	}
	if !strings.Contains(html, "token-xyz") {
		t.Error("expected html content to contain token token-xyz")
	}
}

func TestBrevoClientHandlesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":"invalid API key"}`)
	}))
	defer srv.Close()

	client := NewBrevoClient("bad-key", "noreply@test.com", "Test Sender")
	client.baseURL = srv.URL

	err := client.SendVerificationOTP("alumni@example.com", "123456", "token-xyz")
	if err == nil {
		t.Fatal("expected error from Brevo API")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected 401 in error, got %v", err)
	}
}

func TestResolveAPIKey(t *testing.T) {
	if got := ResolveAPIKey("  xkeysib-direct-123  "); got != "xkeysib-direct-123" {
		t.Errorf("expected direct key, got %q", got)
	}
	mcp := base64.StdEncoding.EncodeToString([]byte(`{"api_key":"xkeysib-mcp-abc"}`))
	if got := ResolveAPIKey(mcp); got != "xkeysib-mcp-abc" {
		t.Errorf("expected extracted MCP key xkeysib-mcp-abc, got %q", got)
	}
}

func TestBrevoClientSendVerificationSMS(t *testing.T) {
	mock := &mockBrevoServer{}
	srv := httptest.NewServer(http.HandlerFunc(mock.handler))
	defer srv.Close()

	client := NewBrevoClient("xkeysib-sms", "noreply@test.com", "Test")
	client.baseURL = srv.URL
	client.ConfigureMessaging("NTUSTID", "", 0)

	if err := client.SendVerificationSMS("628123456789", "654321"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastRequest.URL.Path != "/v3/transactionalSMS/send" {
		t.Errorf("expected path /v3/transactionalSMS/send, got %s", mock.lastRequest.URL.Path)
	}
	if mock.lastRequest.Header.Get("api-key") != "xkeysib-sms" {
		t.Errorf("expected api-key header xkeysib-sms, got %s", mock.lastRequest.Header.Get("api-key"))
	}
	if !strings.Contains(mock.lastBody, `"recipient":"628123456789"`) || !strings.Contains(mock.lastBody, "654321") {
		t.Errorf("unexpected SMS body: %s", mock.lastBody)
	}
}

func TestBrevoClientSendVerificationWhatsApp(t *testing.T) {
	mock := &mockBrevoServer{}
	srv := httptest.NewServer(http.HandlerFunc(mock.handler))
	defer srv.Close()

	client := NewBrevoClient("xkeysib-wa", "noreply@test.com", "Test")
	client.baseURL = srv.URL

	// Missing sender fails clearly
	if err := client.SendVerificationWhatsApp("628123456789", "112233"); err == nil {
		t.Fatal("expected error when whatsappSender is empty")
	}

	// Plain text mode
	client.ConfigureMessaging("", "628111000222", 0)
	if err := client.SendVerificationWhatsApp("628123456789", "112233"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastRequest.URL.Path != "/v3/whatsapp/sendMessage" {
		t.Errorf("expected /v3/whatsapp/sendMessage, got %s", mock.lastRequest.URL.Path)
	}
	if !strings.Contains(mock.lastBody, `"senderNumber":"628111000222"`) || !strings.Contains(mock.lastBody, "112233") {
		t.Errorf("unexpected WA text body: %s", mock.lastBody)
	}

	// Template mode
	client.ConfigureMessaging("", "628111000222", 42)
	if err := client.SendVerificationWhatsApp("628123456789", "998877"); err != nil {
		t.Fatalf("unexpected template error: %v", err)
	}
	if !strings.Contains(mock.lastBody, `"templateId":42`) || !strings.Contains(mock.lastBody, `"OTP":"998877"`) {
		t.Errorf("unexpected WA template body: %s", mock.lastBody)
	}
}
