package runtime

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthCodesAndSessionsAreStoredAsHashes(t *testing.T) {
	requireSQLite3(t)

	r := mustNewAuthRuntime(t, filepath.Join(t.TempDir(), "auth-hashing.db"))
	email := "hashed@example.com"

	loginCode := requestCodeAndUseKnownCode(t, r, email)
	codeRow, ok, err := queryRow(r.DB, `SELECT code FROM mar_auth_codes WHERE email = ? ORDER BY id DESC LIMIT 1`, email)
	if err != nil {
		t.Fatalf("load auth code failed: %v", err)
	}
	if !ok {
		t.Fatal("expected auth code row")
	}
	storedCode, _ := codeRow["code"].(string)
	if storedCode == strings.TrimSpace(loginCode) {
		t.Fatalf("expected auth code to be stored hashed, got raw value %q", storedCode)
	}
	if !strings.HasPrefix(storedCode, "sha256:") {
		t.Fatalf("expected auth code hash prefix, got %q", storedCode)
	}

	token := loginWithCodeAndReadToken(t, r, email, loginCode)
	sessionRow, ok, err := queryRow(r.DB, `SELECT token FROM mar_sessions WHERE email = ? ORDER BY created_at DESC LIMIT 1`, email)
	if err != nil {
		t.Fatalf("load session failed: %v", err)
	}
	if !ok {
		t.Fatal("expected session row")
	}
	storedToken, _ := sessionRow["token"].(string)
	if storedToken == strings.TrimSpace(token) {
		t.Fatalf("expected session token to be stored hashed, got raw value %q", storedToken)
	}
	if !strings.HasPrefix(storedToken, "sha256:") {
		t.Fatalf("expected session token hash prefix, got %q", storedToken)
	}
}
