package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSOriginAllowList(t *testing.T) {
	t.Setenv("FERSCALC_CORS_ORIGINS", "https://example.com")
	server := NewServer("0")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta/states", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("expected allow-origin header, got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/meta/states", nil)
	req.Header.Set("Origin", "https://not-allowed.example")
	rec = httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow-origin header, got %q", got)
	}
}

func TestMaxBodyBytes(t *testing.T) {
	t.Setenv("FERSCALC_MAX_BODY_BYTES", "20")
	server := NewServer("0")

	payload := `{"data":"` + string(bytes.Repeat([]byte("a"), 100)) + `"}`
	body := []byte(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}
