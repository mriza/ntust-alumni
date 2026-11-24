package email

import (
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
