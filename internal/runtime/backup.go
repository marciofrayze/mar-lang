package runtime

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupResult describes the generated backup file and rotated files.
type BackupResult struct {
	Path       string
	BackupDir  string
	Removed    []string
	KeptLast   int
	Database   string
	OccurredAt int64
}

// CreateSQLiteBackup creates a timestamped SQLite snapshot and rotates old backups.
func CreateSQLiteBackup(databasePath string, keepLast int) (BackupResult, error) {
	if keepLast <= 0 {
		keepLast = 20
	}
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return BackupResult{}, errors.New("sqlite3 is required to create backups")
	}

	baseName := filepath.Base(databasePath)
	prefix := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	if prefix == "" {
		prefix = "database"
	}

	backupDir := filepath.Join(filepath.Dir(databasePath), "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return BackupResult{}, err
	}

	timestamp := time.Now().UTC().Format("20060102T150405Z")
	backupPath := filepath.Join(backupDir, prefix+"-"+timestamp+".db")
	quotedPath := strings.ReplaceAll(backupPath, "'", "''")

	cmd := exec.Command("sqlite3", databasePath, "VACUUM INTO '"+quotedPath+"';")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return BackupResult{}, fmt.Errorf("backup failed: %s", msg)
	}

	removed, err := rotateBackups(backupDir, prefix, keepLast)
	if err != nil {
		return BackupResult{}, err
	}

	return BackupResult{
		Path:       backupPath,
		BackupDir:  backupDir,
		Removed:    removed,
		KeptLast:   keepLast,
		Database:   databasePath,
		OccurredAt: time.Now().UnixMilli(),
	}, nil
}

func rotateBackups(backupDir, prefix string, keepLast int) ([]string, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}

	pattern := prefix + "-"
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, pattern) && strings.HasSuffix(name, ".db") {
			files = append(files, name)
		}
	}
	if len(files) <= keepLast {
		return nil, nil
	}

	sort.Strings(files)
	removeCount := len(files) - keepLast
	removed := make([]string, 0, removeCount)
	for i := 0; i < removeCount; i++ {
		path := filepath.Join(backupDir, files[i])
		if err := os.Remove(path); err != nil {
			return removed, err
		}
		removed = append(removed, path)
	}
	return removed, nil
}
