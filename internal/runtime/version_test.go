package runtime

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
)

func TestVersionEndpointPublicPayload(t *testing.T) {
	requireSQLite3(t)

	r := mustNewAuthRuntime(t, filepath.Join(t.TempDir(), "version-public.db"))
	r.SetVersionInfo(VersionInfo{
		MarVersion:   "v1.2.3",
		MarCommit:    "abc123",
		MarBuildTime: "2026-03-04T16:00:00Z",
		AppBuildTime:  "2026-03-04T16:10:00Z",
		ManifestHash:  "sha256:deadbeef",
	})

	rec := doRuntimeRequest(r, http.MethodGet, "/_mar/version", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /_mar/version, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode /_mar/version failed: %v body=%s", err, rec.Body.String())
	}
	if _, hasMar := payload["mar"]; hasMar {
		t.Fatalf("public version payload must not include mar details: %+v", payload)
	}
	app, ok := payload["app"].(map[string]any)
	if !ok {
		t.Fatalf("expected app object in payload: %+v", payload)
	}
	if app["name"] != "AuthBootstrapApi" {
		t.Fatalf("unexpected app.name: %+v", app)
	}
	if app["manifestHash"] != "sha256:deadbeef" {
		t.Fatalf("unexpected app.manifestHash: %+v", app)
	}
}

func TestVersionEndpointAdminRequiresAdminRole(t *testing.T) {
	requireSQLite3(t)

	r := mustNewAuthRuntime(t, filepath.Join(t.TempDir(), "version-admin.db"))
	r.SetVersionInfo(VersionInfo{
		MarVersion:   "v1.2.3",
		MarCommit:    "abc123",
		MarBuildTime: "2026-03-04T16:00:00Z",
		AppBuildTime:  "2026-03-04T16:10:00Z",
		ManifestHash:  "sha256:deadbeef",
	})

	unauth := doRuntimeRequest(r, http.MethodGet, "/_mar/version/admin", "", "")
	if unauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d body=%s", unauth.Code, unauth.Body.String())
	}

	devCode := requestCodeAndReadDevCode(t, r, "admin@example.com")
	token := loginWithCodeAndReadToken(t, r, "admin@example.com", devCode)

	rec := doRuntimeRequest(r, http.MethodGet, "/_mar/version/admin", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /_mar/version/admin, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		App struct {
			Name string `json:"name"`
		} `json:"app"`
		Mar struct {
			Version string `json:"version"`
			Commit  string `json:"commit"`
		} `json:"mar"`
		Runtime struct {
			GoVersion string `json:"goVersion"`
			Platform  string `json:"platform"`
		} `json:"runtime"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode /_mar/version/admin failed: %v body=%s", err, rec.Body.String())
	}
	if payload.Mar.Version != "v1.2.3" || payload.Mar.Commit != "abc123" {
		t.Fatalf("unexpected mar payload: %+v", payload.Mar)
	}
	if payload.Runtime.GoVersion == "" || payload.Runtime.Platform == "" {
		t.Fatalf("expected runtime payload fields, got: %+v", payload.Runtime)
	}
}
