package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakePinger struct {
	err error
}

func (pinger fakePinger) Ping(context.Context) error {
	return pinger.err
}

func testRouter(pinger fakePinger) http.Handler {
	return NewRouter(pinger, slog.New(slog.NewTextHandler(io.Discard, nil)),
		[]string{"https://www.zaminflow.uz"}, time.Second, 1024)
}

func TestHealthEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		pinger     fakePinger
		wantStatus int
		wantBody   string
	}{
		{name: "liveness", path: "/healthz", wantStatus: http.StatusOK, wantBody: `"status":"ok"`},
		{name: "ready", path: "/readyz", wantStatus: http.StatusOK, wantBody: `"status":"ready"`},
		{name: "not ready", path: "/readyz", pinger: fakePinger{err: errors.New("db down")}, wantStatus: http.StatusServiceUnavailable, wantBody: `"code":"dependency_unavailable"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			testRouter(test.pinger).ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if !strings.Contains(response.Body.String(), test.wantBody) {
				t.Fatalf("body %q does not contain %q", response.Body.String(), test.wantBody)
			}
			if response.Header().Get("X-Request-ID") == "" {
				t.Fatal("expected request ID response header")
			}
			if response.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("expected security headers")
			}
		})
	}
}

func TestCORSAllowlistAndPreflight(t *testing.T) {
	handler := testRouter(fakePinger{})
	request := httptest.NewRequest(http.MethodOptions, "/readyz", nil)
	request.Header.Set("Origin", "https://www.zaminflow.uz")
	request.Header.Set("Access-Control-Request-Method", "GET")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "https://www.zaminflow.uz" {
		t.Fatal("allowed origin was not reflected")
	}

	request = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("disallowed origin status = %d, want %d", response.Code, http.StatusForbidden)
	}
	var body errorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode forbidden response: %v", err)
	}
	if body.Error.Code != "cors_origin_denied" {
		t.Fatalf("error code = %q, want cors_origin_denied", body.Error.Code)
	}
}

func TestCompression(t *testing.T) {
	handler := testRouter(fakePinger{})
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("content encoding = %q, want gzip", response.Header().Get("Content-Encoding"))
	}
}

func TestUnknownRouteUsesErrorEnvelope(t *testing.T) {
	response := httptest.NewRecorder()
	testRouter(fakePinger{}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	var body errorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Error.Code != "not_found" || body.Error.RequestID == "" {
		t.Fatalf("error = %#v", body.Error)
	}
}

func TestMetricsRequiresTokenAndUsesRouteTemplates(t *testing.T) {
	handler := NewRouter(fakePinger{}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		[]string{"https://www.zaminflow.uz"}, time.Second, 1024,
		RouterOptions{MetricsToken: "a-valid-test-metrics-token"})
	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated metrics status = %d, want %d", response.Code, http.StatusUnauthorized)
	}

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	request.Header.Set("Authorization", "Bearer a-valid-test-metrics-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("authorized metrics status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `route="/healthz"`) ||
		strings.Contains(response.Body.String(), "a-valid-test-metrics-token") {
		t.Fatalf("unexpected metrics output: %s", response.Body.String())
	}
}
