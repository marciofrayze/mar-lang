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

// BackupFile describes an existing backup file on disk.
type BackupFile struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	SizeBytes   int64  `json:"sizeBytes"`
	CreatedAtMs int64  `json:"createdAtMs"`
	CreatedAt   string `json:"createdAt"`
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

// ListSQLiteBackups lists existing backup files for a SQLite database, newest first.
func ListSQLiteBackups(databasePath string, limit int) ([]BackupFile, error) {
	baseName := filepath.Base(databasePath)
	prefix := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	if prefix == "" {
		prefix = "database"
	}
	backupDir := filepath.Join(filepath.Dir(databasePath), "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []BackupFile{}, nil
		}
		return nil, err
	}

	pattern := prefix + "-"
	files := make([]BackupFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, pattern) || !strings.HasSuffix(name, ".db") {
			continue
		}

		fullPath := filepath.Join(backupDir, name)
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}

		timestamp := backupFileTimestamp(name, info.ModTime())
		files = append(files, BackupFile{
			Path:        fullPath,
			Name:        name,
			SizeBytes:   info.Size(),
			CreatedAtMs: timestamp.UnixMilli(),
			CreatedAt:   timestamp.Format("2006-01-02 15:04:05"),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].CreatedAtMs > files[j].CreatedAtMs
	})
	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}
	return files, nil
}

func backupFileTimestamp(fileName string, fallback time.Time) time.Time {
	trimmed := strings.TrimSuffix(fileName, ".db")
	parts := strings.Split(trimmed, "-")
	if len(parts) == 0 {
		return fallback
	}
	raw := parts[len(parts)-1]
	if ts, err := time.Parse("20060102T150405Z", raw); err == nil {
		return ts.Local()
	}
	return fallback
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
		return []string{}, nil
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
