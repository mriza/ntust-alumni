package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"ntust-alumni/backend/internal/middleware"
	"ntust-alumni/backend/internal/models"
)

func setupOAuthTestRouter(database *fakeDB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	middleware.InitJWT("test-oauth-secret")
	SetJWTKey([]byte("test-oauth-secret"))

	h := NewOAuthHandler(database, OAuthConfig{
		ClientID:              "ntust-client",
		ClientSecret:          "ntust-client-secret",
		AllowedRedirectPrefix: "http://localhost:9080/answer/api/v1/connector/",
	})

	r := gin.New()
	r.POST("/api/oauth/token", h.Token)
	r.GET("/api/oauth/userinfo", h.UserInfo)

	verified := r.Group("/api")
	verified.Use(middleware.AuthMiddleware(), middleware.RequireVerified())
	verified.POST("/oauth/authorize", h.Authorize)
	return r
}

func TestOAuthAuthorizeRejectsPendingAndBadRedirect(t *testing.T) {
	database := newFakeDB()
	database.SaveAlumni(context.Background(), &models.Alumni{ID: "u-pend", Email: "p@gmail.com", Role: "member", Status: "pending"})
	database.SaveAlumni(context.Background(), &models.Alumni{ID: "u-ver", Email: "v@gmail.com", Role: "member", Status: "verified", Nama: "Verified User"})
	router := setupOAuthTestRouter(database)

	pendTok, _ := GenerateJWT(&models.Alumni{ID: "u-pend", Email: "p@gmail.com", Role: "member", Status: "pending"})
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/oauth/authorize", bytes.NewBufferString(`{"client_id":"ntust-client","redirect_uri":"http://localhost:9080/answer/api/v1/connector/redirect/basic"}`))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+pendTok)
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for pending user, got %d", w1.Code)
	}

	verTok, _ := GenerateJWT(&models.Alumni{ID: "u-ver", Email: "v@gmail.com", Role: "member", Status: "verified"})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/oauth/authorize", bytes.NewBufferString(`{"client_id":"ntust-client","redirect_uri":"https://evil.example.com/steal"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+verTok)
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for disallowed redirect_uri, got %d", w2.Code)
	}
}

func TestOAuthFullFlowAndReplayAndWrongSecret(t *testing.T) {
	database := newFakeDB()
	database.SaveAlumni(context.Background(), &models.Alumni{ID: "u-ver123", Email: "rina.kusuma@gmail.com", Role: "member", Status: "verified", Nama: "Rina Kusuma"})
	router := setupOAuthTestRouter(database)

	verTok, _ := GenerateJWT(&models.Alumni{ID: "u-ver123", Email: "rina.kusuma@gmail.com", Role: "member", Status: "verified"})
	redirectURI := "http://localhost:9080/answer/api/v1/connector/login/basic_connector"

	// 1. Authorize
	authBody, _ := json.Marshal(map[string]string{
		"client_id":     "ntust-client",
		"redirect_uri":  redirectURI,
		"state":         "state-xyz",
		"response_type": "code",
	})
	wAuth := httptest.NewRecorder()
	reqAuth, _ := http.NewRequest(http.MethodPost, "/api/oauth/authorize", bytes.NewReader(authBody))
	reqAuth.Header.Set("Content-Type", "application/json")
	reqAuth.Header.Set("Authorization", "Bearer "+verTok)
	router.ServeHTTP(wAuth, reqAuth)
	if wAuth.Code != http.StatusOK {
		t.Fatalf("expected 200 on authorize, got %d: %s", wAuth.Code, wAuth.Body.String())
	}
	var authResp struct {
		Data struct {
			Code        string `json:"code"`
			RedirectURL string `json:"redirect_url"`
		} `json:"data"`
	}
	json.Unmarshal(wAuth.Body.Bytes(), &authResp)
	if authResp.Data.Code == "" || !strings.Contains(authResp.Data.RedirectURL, "state=state-xyz") {
		t.Fatalf("unexpected authorize response: %+v", authResp.Data)
	}

	// 2. Wrong client_secret rejected 401
	formBad := url.Values{"grant_type": {"authorization_code"}, "code": {authResp.Data.Code}, "redirect_uri": {redirectURI}}
	wBad := httptest.NewRecorder()
	reqBad, _ := http.NewRequest(http.MethodPost, "/api/oauth/token", strings.NewReader(formBad.Encode()))
	reqBad.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqBad.SetBasicAuth("ntust-client", "wrong-secret")
	router.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong client_secret, got %d", wBad.Code)
	}

	// 3. Valid token exchange via form-urlencoded + Basic Auth
	wTok := httptest.NewRecorder()
	reqTok, _ := http.NewRequest(http.MethodPost, "/api/oauth/token", strings.NewReader(formBad.Encode()))
	reqTok.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqTok.SetBasicAuth("ntust-client", "ntust-client-secret")
	router.ServeHTTP(wTok, reqTok)
	if wTok.Code != http.StatusOK {
		t.Fatalf("expected 200 on token exchange, got %d: %s", wTok.Code, wTok.Body.String())
	}
	var tokResp map[string]interface{}
	json.Unmarshal(wTok.Body.Bytes(), &tokResp)
	accessToken, _ := tokResp["access_token"].(string)
	if accessToken == "" {
		t.Fatalf("expected access_token, got %+v", tokResp)
	}

	// 4. Replay of same code must fail 400
	wReplay := httptest.NewRecorder()
	reqReplay, _ := http.NewRequest(http.MethodPost, "/api/oauth/token", strings.NewReader(formBad.Encode()))
	reqReplay.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqReplay.SetBasicAuth("ntust-client", "ntust-client-secret")
	router.ServeHTTP(wReplay, reqReplay)
	if wReplay.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on code replay, got %d", wReplay.Code)
	}

	// 5. UserInfo returns flat JSON
	wInfo := httptest.NewRecorder()
	reqInfo, _ := http.NewRequest(http.MethodGet, "/api/oauth/userinfo", nil)
	reqInfo.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(wInfo, reqInfo)
	if wInfo.Code != http.StatusOK {
		t.Fatalf("expected 200 on userinfo, got %d: %s", wInfo.Code, wInfo.Body.String())
	}
	var info map[string]string
	json.Unmarshal(wInfo.Body.Bytes(), &info)
	if info["id"] != "u-ver123" || info["email"] != "rina.kusuma@gmail.com" || info["display_name"] != "Rina Kusuma" || info["username"] == "" {
		t.Errorf("unexpected userinfo payload: %+v", info)
	}
}

func TestOAuthAccessTokenRejectedByAuthMiddleware(t *testing.T) {
	database := newFakeDB()
	database.SaveAlumni(context.Background(), &models.Alumni{ID: "u-admin", Email: "admin@ntust-alumni.id", Role: "super_admin", Status: "verified", Nama: "Super Admin"})
	router := setupOAuthTestRouter(database)
	router.GET("/api/me", middleware.AuthMiddleware(), NewAuthHandler(database).Me)

	adminTok, _ := GenerateJWT(&models.Alumni{ID: "u-admin", Email: "admin@ntust-alumni.id", Role: "super_admin", Status: "verified"})
	redirectURI := "http://localhost:9080/answer/api/v1/connector/redirect/basic"
	authBody, _ := json.Marshal(map[string]string{"client_id": "ntust-client", "redirect_uri": redirectURI})
	wA := httptest.NewRecorder()
	reqA, _ := http.NewRequest(http.MethodPost, "/api/oauth/authorize", bytes.NewReader(authBody))
	reqA.Header.Set("Content-Type", "application/json")
	reqA.Header.Set("Authorization", "Bearer "+adminTok)
	router.ServeHTTP(wA, reqA)
	var aResp struct {
		Data struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	json.Unmarshal(wA.Body.Bytes(), &aResp)

	form := url.Values{"grant_type": {"authorization_code"}, "code": {aResp.Data.Code}, "redirect_uri": {redirectURI}}
	wT := httptest.NewRecorder()
	reqT, _ := http.NewRequest(http.MethodPost, "/api/oauth/token", strings.NewReader(form.Encode()))
	reqT.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqT.SetBasicAuth("ntust-client", "ntust-client-secret")
	router.ServeHTTP(wT, reqT)
	var tResp map[string]interface{}
	json.Unmarshal(wT.Body.Bytes(), &tResp)
	oauthToken, _ := tResp["access_token"].(string)

	// Using OAuth access token on portal API /api/me MUST fail with 401
	wMe := httptest.NewRecorder()
	reqMe, _ := http.NewRequest(http.MethodGet, "/api/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+oauthToken)
	router.ServeHTTP(wMe, reqMe)
	if wMe.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when using oauth_access token on portal API, got %d: %s", wMe.Code, wMe.Body.String())
	}

	// Omitting redirect_uri on /api/oauth/token MUST fail with 400
	wA2 := httptest.NewRecorder()
	reqA2, _ := http.NewRequest(http.MethodPost, "/api/oauth/authorize", bytes.NewReader(authBody))
	reqA2.Header.Set("Content-Type", "application/json")
	reqA2.Header.Set("Authorization", "Bearer "+adminTok)
	router.ServeHTTP(wA2, reqA2)
	var aResp2 struct {
		Data struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	json.Unmarshal(wA2.Body.Bytes(), &aResp2)

	formNoRedirect := url.Values{"grant_type": {"authorization_code"}, "code": {aResp2.Data.Code}}
	wNoR := httptest.NewRecorder()
	reqNoR, _ := http.NewRequest(http.MethodPost, "/api/oauth/token", strings.NewReader(formNoRedirect.Encode()))
	reqNoR.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqNoR.SetBasicAuth("ntust-client", "ntust-client-secret")
	router.ServeHTTP(wNoR, reqNoR)
	if wNoR.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when redirect_uri is omitted on /api/oauth/token, got %d", wNoR.Code)
	}
}
