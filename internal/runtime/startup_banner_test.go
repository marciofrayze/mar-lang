package runtime

import (
	"path/filepath"
	"testing"
)

func TestResolveDatabaseDisplayPathUsesLaunchCWDForRelativeDatabase(t *testing.T) {
	launchCWD := t.TempDir()
	processCWD := filepath.Join(launchCWD, "build", "todo")

	got := resolveDatabaseDisplayPath("todo.db", processCWD, launchCWD)
	want := filepath.Join("build", "todo", "todo.db")
	if got != want {
		t.Fatalf("unexpected database path display:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestResolveDatabaseDisplayPathFallsBackToLiteralWithoutLaunchCWD(t *testing.T) {
	got := resolveDatabaseDisplayPath("todo.db", "/tmp/build/todo", "")
	if got != "todo.db" {
		t.Fatalf("expected literal relative database path, got %q", got)
	}
}

func TestResolveDatabaseDisplayPathKeepsAbsoluteDatabasePath(t *testing.T) {
	got := resolveDatabaseDisplayPath("/tmp/data/todo.db", "/tmp/build/todo", "/tmp/project")
	if got != "/tmp/data/todo.db" {
		t.Fatalf("expected absolute database path, got %q", got)
	}
}

func TestResolveDatabaseDisplayPathRelativizesAbsolutePathToLaunchCWD(t *testing.T) {
	launchCWD := t.TempDir()
	absolutePath := filepath.Join(launchCWD, "build", "todo", "todo.db")

	got := resolveDatabaseDisplayPath(absolutePath, filepath.Join(launchCWD, "build", "todo"), launchCWD)
	want := filepath.Join("build", "todo", "todo.db")
	if got != want {
		t.Fatalf("unexpected relative absolute database path display:\nwant: %q\ngot:  %q", want, got)
	}
}
