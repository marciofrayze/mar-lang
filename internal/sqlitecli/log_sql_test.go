package sqlitecli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestQueryHookIncludesInterpolatedExecArgs(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "log-interpolation.db")
	db := Open(dbPath)
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, email TEXT, age INTEGER)`); err != nil {
		t.Fatalf("create table failed: %v", err)
	}

	var events []QueryEvent
	db.SetQueryHook(func(event QueryEvent) {
		events = append(events, event)
	})

	if _, err := db.Exec(`INSERT INTO users (email, age) VALUES (?, ?)`, "owner@example.com", 33); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected query event to be captured")
	}

	sqlText := events[len(events)-1].SQL
	if !strings.Contains(sqlText, "'owner@example.com'") {
		t.Fatalf("expected interpolated email in query log, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "33") {
		t.Fatalf("expected interpolated numeric arg in query log, got %q", sqlText)
	}
	if strings.Contains(sqlText, "?") {
		t.Fatalf("expected no bind placeholders in query log, got %q", sqlText)
	}
}

func TestInterpolateSQLForLogEscapesSingleQuotes(t *testing.T) {
	got := interpolateSQLForLog(`INSERT INTO books (title) VALUES (?)`, []any{"Marcio's Book"})
	if !strings.Contains(got, "'Marcio''s Book'") {
		t.Fatalf("expected escaped single quote in interpolated SQL, got %q", got)
	}
}
