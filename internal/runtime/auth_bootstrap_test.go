package runtime

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"belm/internal/parser"
)

func TestBootstrapAdminCreatesSingleAdminWhenNoUsers(t *testing.T) {
	requireSQLite3(t)

	r := mustNewAuthRuntime(t, filepath.Join(t.TempDir(), "bootstrap-empty.db"))

	rec := httptest.NewRecorder()
	if err := r.handleBootstrapAdmin(rec, map[string]any{"email": "owner@example.com"}); err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	countRow, ok, err := queryRow(r.DB, `SELECT COUNT(*) AS total FROM users`)
	if err != nil {
		t.Fatalf("count users failed: %v", err)
	}
	if !ok {
		t.Fatal("count users returned no rows")
	}
	totalUsers, _ := toInt64(countRow["total"])
	if totalUsers != 1 {
		t.Fatalf("expected 1 user after bootstrap, got %d", totalUsers)
	}

	user, found, err := r.loadAuthUserByEmail("owner@example.com")
	if err != nil {
		t.Fatalf("load user failed: %v", err)
	}
	if !found {
		t.Fatal("expected bootstrap user to exist")
	}
	role, _ := user["role"].(string)
	if strings.ToLower(strings.TrimSpace(role)) != "admin" {
		t.Fatalf("expected role admin, got %q", role)
	}
}

func TestBootstrapAdminBlockedWhenAnyUserAlreadyExists(t *testing.T) {
	requireSQLite3(t)

	r := mustNewAuthRuntime(t, filepath.Join(t.TempDir(), "bootstrap-blocked.db"))

	// First request-code auto-creates a regular user.
	requestRec := httptest.NewRecorder()
	if err := r.handleAuthRequestCode(requestRec, map[string]any{"email": "user@example.com"}); err != nil {
		t.Fatalf("request-code failed: %v", err)
	}

	// Bootstrap must now be blocked because there is already at least one user.
	bootstrapRec := httptest.NewRecorder()
	err := r.handleBootstrapAdmin(bootstrapRec, map[string]any{"email": "admin@example.com"})
	if err == nil {
		t.Fatal("expected bootstrap to be blocked when users already exist")
	}

	apiErr, ok := err.(*apiError)
	if !ok {
		t.Fatalf("expected apiError, got %T: %v", err, err)
	}
	if apiErr.Status != 409 {
		t.Fatalf("expected status 409, got %d", apiErr.Status)
	}
	if !strings.Contains(apiErr.Message, "no users") {
		t.Fatalf("unexpected error message: %q", apiErr.Message)
	}
}

func TestRequestCodeCreatesAdminWhenNoUsersExist(t *testing.T) {
	requireSQLite3(t)

	r := mustNewAuthRuntime(t, filepath.Join(t.TempDir(), "request-code-first-admin.db"))

	requestRec := httptest.NewRecorder()
	if err := r.handleAuthRequestCode(requestRec, map[string]any{"email": "first@example.com"}); err != nil {
		t.Fatalf("request-code failed: %v", err)
	}

	user, found, err := r.loadAuthUserByEmail("first@example.com")
	if err != nil {
		t.Fatalf("load user failed: %v", err)
	}
	if !found {
		t.Fatal("expected first user to be created")
	}
	role, _ := user["role"].(string)
	if strings.ToLower(strings.TrimSpace(role)) != "admin" {
		t.Fatalf("expected first user role to be admin, got %q", role)
	}

	countRow, ok, err := queryRow(r.DB, `SELECT COUNT(*) AS total FROM users`)
	if err != nil {
		t.Fatalf("count users failed: %v", err)
	}
	if !ok {
		t.Fatal("count users returned no rows")
	}
	totalUsers, _ := toInt64(countRow["total"])
	if totalUsers != 1 {
		t.Fatalf("expected exactly 1 user, got %d", totalUsers)
	}
}

func mustNewAuthRuntime(t *testing.T, dbPath string) *Runtime {
	t.Helper()
	app, err := parser.Parse(strings.TrimSpace(`
app AuthBootstrapApi

entity User {
  id: Int primary auto
  email: String
  role: String
}

entity Todo {
  id: Int primary auto
  title: String
}

auth {
  user_entity User
  email_field email
  role_field role
  email_transport console
  dev_expose_code true
}
`) + "\n")
	if err != nil {
		t.Fatalf("failed to parse app: %v", err)
	}
	app.Database = dbPath

	r, err := New(app)
	if err != nil {
		t.Fatalf("runtime.New failed: %v", err)
	}
	return r
}
