package sqlitecli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type DB struct {
	Path    string
	hookMu  sync.RWMutex
	onQuery func(QueryEvent)
}

type Result struct {
	Changes       int64
	LastInsertRow int64
}

type Statement struct {
	Query string
	Args  []any
}

type QueryEvent struct {
	SQL        string
	DurationMs float64
	RowCount   int
	Error      string
}

func Open(path string) *DB {
	return &DB{Path: path}
}

func (db *DB) SetQueryHook(hook func(QueryEvent)) {
	db.hookMu.Lock()
	defer db.hookMu.Unlock()
	db.onQuery = hook
}

func (db *DB) Exec(query string, args ...any) (Result, error) {
	expanded, err := expandQuery(query, args)
	if err != nil {
		return Result{}, err
	}
	wrapper := "BEGIN; " + expanded + "; SELECT changes() AS changes, last_insert_rowid() AS last_insert_rowid; COMMIT;"
	rows, err := db.queryJSON(wrapper)
	if err != nil {
		return Result{}, err
	}
	if len(rows) == 0 {
		return Result{}, nil
	}
	last := rows[len(rows)-1]
	return Result{
		Changes:       toInt64(last["changes"]),
		LastInsertRow: toInt64(last["last_insert_rowid"]),
	}, nil
}

func (db *DB) QueryRows(query string, args ...any) ([]map[string]any, error) {
	expanded, err := expandQuery(query, args)
	if err != nil {
		return nil, err
	}
	return db.queryJSON(expanded)
}

func (db *DB) QueryRow(query string, args ...any) (map[string]any, bool, error) {
	rows, err := db.QueryRows(query, args...)
	if err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	return rows[0], true, nil
}

func (db *DB) ExecTx(statements []Statement) error {
	if len(statements) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("BEGIN IMMEDIATE; ")
	for i, stmt := range statements {
		expanded, err := expandQuery(stmt.Query, stmt.Args)
		if err != nil {
			return fmt.Errorf("statement %d: %w", i+1, err)
		}
		b.WriteString(expanded)
		b.WriteString("; ")
	}
	b.WriteString("COMMIT;")
	_, err := db.queryJSON(b.String())
	return err
}

func (db *DB) queryJSON(sqlText string) ([]map[string]any, error) {
	startedAt := time.Now()
	cmd := exec.Command("sqlite3", "-json", db.Path, sqlText)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		db.emitQueryEvent(QueryEvent{
			SQL:        sqlText,
			DurationMs: time.Since(startedAt).Seconds() * 1000,
			RowCount:   0,
			Error:      msg,
		})
		return nil, fmt.Errorf("sqlite3: %s", msg)
	}
	raw := strings.TrimSpace(stdout.String())
	if raw == "" {
		db.emitQueryEvent(QueryEvent{
			SQL:        sqlText,
			DurationMs: time.Since(startedAt).Seconds() * 1000,
			RowCount:   0,
			Error:      "",
		})
		return []map[string]any{}, nil
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		db.emitQueryEvent(QueryEvent{
			SQL:        sqlText,
			DurationMs: time.Since(startedAt).Seconds() * 1000,
			RowCount:   0,
			Error:      fmt.Sprintf("decode sqlite json: %v", err),
		})
		return nil, fmt.Errorf("decode sqlite json: %w", err)
	}
	db.emitQueryEvent(QueryEvent{
		SQL:        sqlText,
		DurationMs: time.Since(startedAt).Seconds() * 1000,
		RowCount:   len(rows),
		Error:      "",
	})
	return rows, nil
}

func (db *DB) emitQueryEvent(event QueryEvent) {
	db.hookMu.RLock()
	hook := db.onQuery
	db.hookMu.RUnlock()
	if hook != nil {
		hook(event)
	}
}

func expandQuery(query string, args []any) (string, error) {
	parts := strings.Split(query, "?")
	if len(parts)-1 != len(args) {
		return "", fmt.Errorf("placeholder count mismatch: %d placeholders for %d args", len(parts)-1, len(args))
	}
	var b strings.Builder
	for i := 0; i < len(args); i++ {
		b.WriteString(parts[i])
		b.WriteString(sqlLiteral(args[i]))
	}
	b.WriteString(parts[len(parts)-1])
	return b.String(), nil
}

func sqlLiteral(v any) string {
	switch t := v.(type) {
	case nil:
		return "NULL"
	case bool:
		if t {
			return "1"
		}
		return "0"
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case float64:
		return fmt.Sprintf("%g", t)
	case float32:
		return fmt.Sprintf("%g", t)
	case string:
		return "'" + strings.ReplaceAll(t, "'", "''") + "'"
	default:
		return "'" + strings.ReplaceAll(fmt.Sprintf("%v", t), "'", "''") + "'"
	}
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	default:
		return 0
	}
}
