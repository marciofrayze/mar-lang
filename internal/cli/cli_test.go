package cli

import (
	"strings"
	"testing"
)

func TestUnknownCommandErrorSuggestsDevForMarFile(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	err := unknownCommandError("mar", "examples/store.mar")
	if err == nil {
		t.Fatal("expected unknownCommandError to return an error")
	}

	msg := err.Error()
	if !strings.Contains(msg, `unknown command "examples/store.mar"`) {
		t.Fatalf("expected unknown command message, got %q", msg)
	}
	if !strings.Contains(msg, "Looks like you want to open this .mar app in development mode.") {
		t.Fatalf("expected friendly .mar hint, got %q", msg)
	}
	if !strings.Contains(msg, "Try: mar dev examples/store.mar") {
		t.Fatalf("expected dev command hint, got %q", msg)
	}
}
