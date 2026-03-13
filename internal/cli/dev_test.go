package cli

import (
	"path/filepath"
	"testing"
)

func TestResolveDevDatabaseOverrideUsesMarDirectory(t *testing.T) {
	t.Parallel()

	projectDir := filepath.Join(t.TempDir(), "todo")
	sourcePath := filepath.Join(projectDir, "todo.mar")

	got := resolveDevDatabaseOverride(sourcePath, "todo.db")
	want := filepath.Join(projectDir, "todo.db")
	if got != want {
		t.Fatalf("unexpected database override path:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestResolveDevDatabaseOverrideKeepsNestedRelativeDatabaseUnderMarDirectory(t *testing.T) {
	t.Parallel()

	projectDir := filepath.Join(t.TempDir(), "todo")
	sourcePath := filepath.Join(projectDir, "todo.mar")

	got := resolveDevDatabaseOverride(sourcePath, filepath.Join("data", "todo.db"))
	want := filepath.Join(projectDir, "data", "todo.db")
	if got != want {
		t.Fatalf("unexpected nested database override path:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestResolveDevDatabaseOverrideSkipsAbsoluteDatabasePath(t *testing.T) {
	t.Parallel()

	sourcePath := filepath.Join(t.TempDir(), "todo.mar")
	absoluteDatabasePath := filepath.Join(t.TempDir(), "todo.db")

	got := resolveDevDatabaseOverride(sourcePath, absoluteDatabasePath)
	if got != "" {
		t.Fatalf("expected no override for absolute database path, got %q", got)
	}
}
