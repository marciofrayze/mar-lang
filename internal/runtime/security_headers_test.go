package runtime

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestDefaultSecurityHeaders(t *testing.T) {
	requireSQLite3(t)

	app := mustParseApp(t, `
app SecurityHeadersApi

entity Todo {
  title: String
}
`)
	app.Database = filepath.Join(t.TempDir(), "security-defaults.db")

	r, err := New(app)
	if err != nil {
		t.Fatalf("runtime.New failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.handleHTTP(rec, req)

	if rec.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Fatalf("unexpected X-Frame-Options: %q", rec.Header().Get("X-Frame-Options"))
	}
	if rec.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Fatalf("unexpected Referrer-Policy: %q", rec.Header().Get("Referrer-Policy"))
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("unexpected X-Content-Type-Options: %q", rec.Header().Get("X-Content-Type-Options"))
	}
}

func TestSystemSecurityHeadersOverride(t *testing.T) {
	requireSQLite3(t)

	app := mustParseApp(t, `
app SecurityHeadersApi

system {
  security_frame_policy deny
  security_referrer_policy no-referrer
  security_content_type_nosniff false
}

entity Todo {
  title: String
}
`)
	app.Database = filepath.Join(t.TempDir(), "security-overrides.db")

	r, err := New(app)
	if err != nil {
		t.Fatalf("runtime.New failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.handleHTTP(rec, req)

	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("unexpected X-Frame-Options: %q", rec.Header().Get("X-Frame-Options"))
	}
	if rec.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("unexpected Referrer-Policy: %q", rec.Header().Get("Referrer-Policy"))
	}
	if rec.Header().Get("X-Content-Type-Options") != "" {
		t.Fatalf("expected X-Content-Type-Options to be omitted, got %q", rec.Header().Get("X-Content-Type-Options"))
	}
}
