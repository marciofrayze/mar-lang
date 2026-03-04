package runtime

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"belm/internal/model"
	"belm/internal/parser"
	"belm/internal/sqlitecli"
)

func TestMigrationsCreateAndAddOptionalField(t *testing.T) {
	requireSQLite3(t)

	dbPath := filepath.Join(t.TempDir(), "migration-safe.db")

	appV1 := mustParseApp(t, `
app MigrationApi

entity Book {
  title: String
}
`)
	appV1.Database = dbPath

	if _, err := New(appV1); err != nil {
		t.Fatalf("runtime.New(v1) failed: %v", err)
	}

	appV2 := mustParseApp(t, `
app MigrationApi

entity Book {
  title: String
  notes: String optional
}
`)
	appV2.Database = dbPath

	if _, err := New(appV2); err != nil {
		t.Fatalf("runtime.New(v2) failed: %v", err)
	}

	db := sqlitecli.Open(dbPath)
	rows, err := db.QueryRows(`PRAGMA table_info("books")`)
	if err != nil {
		t.Fatalf("PRAGMA table_info failed: %v", err)
	}
	notes, ok := findColumn(rows, "notes")
	if !ok {
		t.Fatalf("expected notes column in books table, got rows: %+v", rows)
	}
	if got := strings.ToUpper(fmt.Sprintf("%v", notes["type"])); got != "TEXT" {
		t.Fatalf("unexpected notes column type: %s", got)
	}
	if got := int64Value(notes["notnull"]); got != 0 {
		t.Fatalf("expected notes to be nullable, got notnull=%d", got)
	}
}

func TestMigrationsBlockTypeChange(t *testing.T) {
	requireSQLite3(t)

	dbPath := filepath.Join(t.TempDir(), "migration-type-block.db")

	appV1 := mustParseApp(t, `
app MigrationApi

entity Book {
  title: String
  price: Float
}
`)
	appV1.Database = dbPath
	if _, err := New(appV1); err != nil {
		t.Fatalf("runtime.New(v1) failed: %v", err)
	}

	appV2 := mustParseApp(t, `
app MigrationApi

entity Book {
  title: String
  price: String
}
`)
	appV2.Database = dbPath
	_, err := New(appV2)
	if err == nil {
		t.Fatal("expected migration to block type change")
	}

	msg := err.Error()
	if !strings.Contains(msg, "migration blocked for Book.price") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if !strings.Contains(msg, "type changed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestMigrationsBlockAddingRequiredField(t *testing.T) {
	requireSQLite3(t)

	dbPath := filepath.Join(t.TempDir(), "migration-required-block.db")

	appV1 := mustParseApp(t, `
app MigrationApi

entity Book {
  title: String
}
`)
	appV1.Database = dbPath
	if _, err := New(appV1); err != nil {
		t.Fatalf("runtime.New(v1) failed: %v", err)
	}

	appV2 := mustParseApp(t, `
app MigrationApi

entity Book {
  title: String
  stock: Int
}
`)
	appV2.Database = dbPath
	_, err := New(appV2)
	if err == nil {
		t.Fatal("expected migration to block required field addition")
	}

	msg := err.Error()
	if !strings.Contains(msg, `cannot auto-add required field "stock"`) {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func requireSQLite3(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 binary not found in PATH")
	}
}

func mustParseApp(t *testing.T, src string) *model.App {
	t.Helper()
	app, err := parser.Parse(strings.TrimSpace(src) + "\n")
	if err != nil {
		t.Fatalf("failed to parse app source: %v", err)
	}
	return app
}

func findColumn(rows []map[string]any, name string) (map[string]any, bool) {
	for _, row := range rows {
		if strings.EqualFold(fmt.Sprintf("%v", row["name"]), name) {
			return row, true
		}
	}
	return nil, false
}

func int64Value(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}
