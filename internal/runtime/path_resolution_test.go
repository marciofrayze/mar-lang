package runtime

import (
	"path/filepath"
	"testing"
)

func TestResolvePathRelativeToExecutableKeepsAbsolutePath(t *testing.T) {
	absolutePath := filepath.Join(t.TempDir(), "todo.db")
	if got := ResolvePathRelativeToExecutable(absolutePath); got != absolutePath {
		t.Fatalf("expected absolute path to stay unchanged, got %q", got)
	}
}

