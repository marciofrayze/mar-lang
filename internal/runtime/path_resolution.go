package runtime

import (
	"os"
	"path/filepath"
	"strings"
)

const databasePathOverrideEnv = "MAR_DATABASE_PATH"

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

// ResolveDatabasePath uses MAR_DATABASE_PATH when present, otherwise it anchors
// the app database path at the executable directory.
func ResolveDatabasePath(path string) string {
	if override := strings.TrimSpace(os.Getenv(databasePathOverrideEnv)); override != "" {
		return ResolvePathRelativeToExecutable(override)
	}
	return ResolvePathRelativeToExecutable(path)
}
