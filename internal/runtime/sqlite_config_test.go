package runtime

import (
	"testing"

	"belm/internal/model"
	"belm/internal/sqlitecli"
)

func TestSQLiteConfigForAppDefaults(t *testing.T) {
	app := &model.App{}
	got := sqliteConfigForApp(app)
	want := sqlitecli.DefaultConfig()

	if got != want {
		t.Fatalf("unexpected default sqlite config:\nwant: %+v\ngot:  %+v", want, got)
	}
}

func TestSQLiteConfigForAppOverrides(t *testing.T) {
	journalMode := "delete"
	synchronous := "full"
	foreignKeys := false
	busyTimeoutMs := 12000
	walAutoCheckpoint := 250
	journalSizeLimitMB := 128

	app := &model.App{
		System: &model.SystemConfig{
			SQLiteJournalMode:        &journalMode,
			SQLiteSynchronous:        &synchronous,
			SQLiteForeignKeys:        &foreignKeys,
			SQLiteBusyTimeoutMs:      &busyTimeoutMs,
			SQLiteWALAutoCheckpoint:  &walAutoCheckpoint,
			SQLiteJournalSizeLimitMB: &journalSizeLimitMB,
		},
	}

	got := sqliteConfigForApp(app)
	if got.JournalMode != "delete" {
		t.Fatalf("unexpected journal mode: %q", got.JournalMode)
	}
	if got.Synchronous != "full" {
		t.Fatalf("unexpected synchronous: %q", got.Synchronous)
	}
	if got.ForeignKeys {
		t.Fatal("expected foreign keys disabled")
	}
	if got.BusyTimeoutMs != 12000 {
		t.Fatalf("unexpected busy timeout: %d", got.BusyTimeoutMs)
	}
	if got.WALAutoCheckpoint != 250 {
		t.Fatalf("unexpected wal autocheckpoint: %d", got.WALAutoCheckpoint)
	}
	if got.JournalSizeLimitB != 128*1024*1024 {
		t.Fatalf("unexpected journal size limit bytes: %d", got.JournalSizeLimitB)
	}
}
