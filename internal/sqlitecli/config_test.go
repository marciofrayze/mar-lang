package sqlitecli

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenWithConfigAppliesPragmas(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "config.db")
	db := OpenWithConfig(dbPath, Config{
		JournalMode:       "delete",
		Synchronous:       "full",
		ForeignKeys:       false,
		BusyTimeoutMs:     3210,
		WALAutoCheckpoint: 222,
		JournalSizeLimitB: 1048576,
	})
	defer db.Close()

	journalMode := pragmaValue(t, db, "journal_mode")
	if !strings.EqualFold(journalMode, "delete") {
		t.Fatalf("unexpected journal_mode: %q", journalMode)
	}

	if pragmaValue(t, db, "foreign_keys") != "0" {
		t.Fatalf("unexpected foreign_keys value: %q", pragmaValue(t, db, "foreign_keys"))
	}
	if pragmaValue(t, db, "busy_timeout") != "3210" {
		t.Fatalf("unexpected busy_timeout value: %q", pragmaValue(t, db, "busy_timeout"))
	}
	if pragmaValue(t, db, "wal_autocheckpoint") != "222" {
		t.Fatalf("unexpected wal_autocheckpoint value: %q", pragmaValue(t, db, "wal_autocheckpoint"))
	}
	if pragmaValue(t, db, "journal_size_limit") != "1048576" {
		t.Fatalf("unexpected journal_size_limit value: %q", pragmaValue(t, db, "journal_size_limit"))
	}
}

func TestOpenWithConfigRejectsInvalidConfig(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "invalid.db")
	db := OpenWithConfig(dbPath, Config{
		JournalMode:       "invalid",
		Synchronous:       "normal",
		ForeignKeys:       true,
		BusyTimeoutMs:     1000,
		WALAutoCheckpoint: 1000,
		JournalSizeLimitB: 1024,
	})
	defer db.Close()

	_, err := db.Exec(`SELECT 1`)
	if err == nil {
		t.Fatal("expected database open error for invalid sqlite config")
	}
	if !strings.Contains(err.Error(), "invalid sqlite journal mode") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func pragmaValue(t *testing.T, db *DB, name string) string {
	t.Helper()
	row, ok, err := db.QueryRow("PRAGMA " + name)
	if err != nil {
		t.Fatalf("PRAGMA %s failed: %v", name, err)
	}
	if !ok {
		t.Fatalf("PRAGMA %s returned no rows", name)
	}
	for _, v := range row {
		return strings.TrimSpace(strings.ToLower(toString(v)))
	}
	t.Fatalf("PRAGMA %s returned empty row", name)
	return ""
}

func toString(v any) string {
	switch raw := v.(type) {
	case nil:
		return ""
	case string:
		return raw
	case []byte:
		return string(raw)
	default:
		return fmt.Sprintf("%v", raw)
	}
}
