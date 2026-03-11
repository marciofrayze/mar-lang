package runtime

import (
	"path/filepath"
	"testing"
)

func TestResolveDatabasePathUsesOverrideEnv(t *testing.T) {
	t.Setenv(databasePathOverrideEnv, "/data/todo.db")

	got := ResolveDatabasePath("todo.db")
	if got != "/data/todo.db" {
		t.Fatalf("expected database override path, got %q", got)
	}
}

func TestResolveDatabasePathFallsBackToExecutableRelativePath(t *testing.T) {
	got := ResolveDatabasePath("todo.db")
	if filepath.Base(got) != "todo.db" {
		t.Fatalf("expected resolved database basename todo.db, got %q", got)
	}
}
