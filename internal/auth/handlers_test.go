package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zaminflow/zaminflow-api/internal/config"
)

func TestCSRFValidationRequiresMatchingSignedTokenAndAllowedOrigin(t *testing.T) {
	api := &handler{config: config.Config{
		AppSecret:          "csrf-handler-test-secret",
		CORSAllowedOrigins: []string{"https://www.zaminflow.uz"},
	}}
	token, err := newCSRFToken(api.config.AppSecret)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		origin     string
		cookie     string
		header     string
		wantValid  bool
		wantStatus int
	}{
		{name: "valid", origin: "https://www.zaminflow.uz", cookie: token, header: token, wantValid: true, wantStatus: http.StatusOK},
		{name: "mismatched token", origin: "https://www.zaminflow.uz", cookie: token, header: "attacker-value", wantStatus: http.StatusForbidden},
		{name: "unsigned cookie", origin: "https://www.zaminflow.uz", cookie: "forged-token", header: "forged-token", wantStatus: http.StatusForbidden},
		{name: "untrusted origin", origin: "https://attacker.example", cookie: token, header: token, wantStatus: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
			request.Header.Set("Origin", test.origin)
			request.Header.Set("X-CSRF-Token", test.header)
			request.AddCookie(&http.Cookie{Name: "zf_csrf", Value: test.cookie})
			response := httptest.NewRecorder()
			if got := api.validCSRF(response, request); got != test.wantValid {
				t.Fatalf("validCSRF() = %t, want %t", got, test.wantValid)
			}
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestRequestIPUsesProxyHeaderOnlyInProduction(t *testing.T) {
	tests := []struct {
		name        string
		appEnv      string
		proxyHeader string
		want        string
	}{
		{name: "production valid proxy address", appEnv: "production", proxyHeader: "203.0.113.8", want: "203.0.113.8"},
		{name: "production invalid proxy address", appEnv: "production", proxyHeader: "not-an-ip", want: "192.0.2.4"},
		{name: "development ignores proxy header", appEnv: "development", proxyHeader: "203.0.113.8", want: "192.0.2.4"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &handler{config: config.Config{AppEnv: test.appEnv}}
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			request.RemoteAddr = "192.0.2.4:54321"
			request.Header.Set("X-Real-IP", test.proxyHeader)
			if got := api.requestIP(request); got != test.want {
				t.Fatalf("requestIP() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestServiceErrorsPreserveQueryAndBodyStatusCodes(t *testing.T) {
	api := &handler{}
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid query", err: ErrInvalidQuery, wantStatus: http.StatusBadRequest, wantCode: `"code":"invalid_request"`},
		{name: "invalid body", err: ErrInvalidInput, wantStatus: http.StatusUnprocessableEntity, wantCode: `"code":"validation_failed"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			response := httptest.NewRecorder()
			api.writeServiceError(response, request, test.err)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if !strings.Contains(response.Body.String(), test.wantCode) {
				t.Fatalf("response body %q does not contain %q", response.Body.String(), test.wantCode)
			}
		})
	}
}
