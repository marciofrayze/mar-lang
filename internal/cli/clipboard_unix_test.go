//go:build !windows

package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSystemClipboardRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	clipboardFile := filepath.Join(tempDir, "clipboard.txt")

	copyCommand, pasteCommand := testClipboardCommandsForCurrentOS()
	writeClipboardScript(t, filepath.Join(tempDir, copyCommand), "#!/bin/sh\ncat > \"$MAR_TEST_CLIPBOARD_FILE\"\n")
	writeClipboardScript(t, filepath.Join(tempDir, pasteCommand), "#!/bin/sh\ncat \"$MAR_TEST_CLIPBOARD_FILE\"\n")

	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+originalPath)
	t.Setenv("MAR_TEST_CLIPBOARD_FILE", clipboardFile)
	if runtime.GOOS == "linux" {
		t.Setenv("WAYLAND_DISPLAY", "wayland-1")
	}

	if err := writeSystemClipboard("hello from mar edit"); err != nil {
		t.Fatalf("writeSystemClipboard returned error: %v", err)
	}

	got, err := readSystemClipboard()
	if err != nil {
		t.Fatalf("readSystemClipboard returned error: %v", err)
	}
	if got != "hello from mar edit" {
		t.Fatalf("unexpected clipboard text: got %q", got)
	}
}

func TestSystemClipboardUnavailableWithoutSupportedCommands(t *testing.T) {
	originalLookPath := clipboardLookPath
	t.Cleanup(func() {
		clipboardLookPath = originalLookPath
	})

	clipboardLookPath = func(string) (string, error) {
		return "", exec.ErrNotFound
	}

	if err := writeSystemClipboard("hello"); err == nil {
		t.Fatal("expected writeSystemClipboard to fail without clipboard commands")
	}

	if _, err := readSystemClipboard(); err == nil {
		t.Fatal("expected readSystemClipboard to fail without clipboard commands")
	}
}

func testClipboardCommandsForCurrentOS() (string, string) {
	switch runtime.GOOS {
	case "darwin":
		return "pbcopy", "pbpaste"
	case "linux":
		return "wl-copy", "wl-paste"
	default:
		panic("unsupported test platform: " + runtime.GOOS)
	}
}

func writeClipboardScript(t *testing.T, path string, body string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write script %s: %v", path, err)
	}
}

func TestClipboardToolByNamesRequiresBothCommands(t *testing.T) {
	t.Parallel()

	originalLookPath := clipboardLookPath
	t.Cleanup(func() {
		clipboardLookPath = originalLookPath
	})

	clipboardLookPath = func(name string) (string, error) {
		if name == "copy-tool" {
			return "/tmp/copy-tool", nil
		}
		return "", errors.New("missing")
	}

	if _, err := clipboardToolByNames(
		clipboardCommand{name: "copy-tool"},
		clipboardCommand{name: "paste-tool"},
	); err == nil {
		t.Fatal("expected clipboardToolByNames to fail when paste tool is missing")
	}
}
