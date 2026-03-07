package runtime

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolvePathRelativeToExecutable anchors a relative app path at the executable directory.
func ResolvePathRelativeToExecutable(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return path
	}

	cleaned := filepath.Clean(trimmed)
	if filepath.IsAbs(cleaned) {
		return cleaned
	}

	exePath, err := os.Executable()
	if err != nil || strings.TrimSpace(exePath) == "" {
		return cleaned
	}
	return filepath.Join(filepath.Dir(exePath), cleaned)
}
