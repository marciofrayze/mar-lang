package sqlitecli

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	Path    string
	sqlDB   *sql.DB
	openErr error

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
	sqlDB, err := sql.Open("sqlite", path)
	db := &DB{
		Path:    path,
		sqlDB:   sqlDB,
		openErr: err,
	}
	if err != nil {
		return db
	}

	// Keep a single connection to preserve SQLite transactional behavior and
	// avoid surprising lock interactions in this runtime.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)

	// Ensure the database is reachable early, so failures are clearer.
	if pingErr := sqlDB.Ping(); pingErr != nil {
		db.openErr = pingErr
	}
	return db
}

func (db *DB) Close() error {
	if db == nil || db.sqlDB == nil {
		return nil
	}
	return db.sqlDB.Close()
}

func (db *DB) SetQueryHook(hook func(QueryEvent)) {
	db.hookMu.Lock()
	defer db.hookMu.Unlock()
	db.onQuery = hook
}

func (db *DB) Exec(query string, args ...any) (Result, error) {
	if err := db.ensureOpen(); err != nil {
		db.emitQueryEvent(QueryEvent{
			SQL:        query,
			DurationMs: 0,
			RowCount:   0,
			Error:      err.Error(),
		})
		return Result{}, err
	}

	startedAt := time.Now()
	res, err := db.sqlDB.Exec(query, args...)
	if err != nil {
		db.emitQueryEvent(QueryEvent{
			SQL:        query,
			DurationMs: elapsedMs(startedAt),
			RowCount:   0,
			Error:      err.Error(),
		})
		return Result{}, err
	}

	changes, _ := res.RowsAffected()
	lastInsertRow, _ := res.LastInsertId()
	db.emitQueryEvent(QueryEvent{
		SQL:        query,
		DurationMs: elapsedMs(startedAt),
		RowCount:   0,
		Error:      "",
	})
	return Result{
		Changes:       changes,
		LastInsertRow: lastInsertRow,
	}, nil
}

func (db *DB) QueryRows(query string, args ...any) ([]map[string]any, error) {
	if err := db.ensureOpen(); err != nil {
		db.emitQueryEvent(QueryEvent{
			SQL:        query,
			DurationMs: 0,
			RowCount:   0,
			Error:      err.Error(),
		})
		return nil, err
	}

	startedAt := time.Now()
	rows, err := db.sqlDB.Query(query, args...)
	if err != nil {
		db.emitQueryEvent(QueryEvent{
			SQL:        query,
			DurationMs: elapsedMs(startedAt),
			RowCount:   0,
			Error:      err.Error(),
		})
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		db.emitQueryEvent(QueryEvent{
			SQL:        query,
			DurationMs: elapsedMs(startedAt),
			RowCount:   0,
			Error:      err.Error(),
		})
		return nil, err
	}

	result := make([]map[string]any, 0, 16)
	for rows.Next() {
		rawValues := make([]any, len(columns))
		scanTargets := make([]any, len(columns))
		for i := range rawValues {
			scanTargets[i] = &rawValues[i]
		}

		if err := rows.Scan(scanTargets...); err != nil {
			db.emitQueryEvent(QueryEvent{
				SQL:        query,
				DurationMs: elapsedMs(startedAt),
				RowCount:   len(result),
				Error:      err.Error(),
			})
			return nil, err
		}

		record := make(map[string]any, len(columns))
		for i, col := range columns {
			record[col] = normalizeColumnValue(rawValues[i])
		}
		result = append(result, record)
	}

	if err := rows.Err(); err != nil {
		db.emitQueryEvent(QueryEvent{
			SQL:        query,
			DurationMs: elapsedMs(startedAt),
			RowCount:   len(result),
			Error:      err.Error(),
		})
		return nil, err
	}

	db.emitQueryEvent(QueryEvent{
		SQL:        query,
		DurationMs: elapsedMs(startedAt),
		RowCount:   len(result),
		Error:      "",
	})
	return result, nil
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
	if err := db.ensureOpen(); err != nil {
		db.emitQueryEvent(QueryEvent{
			SQL:        txStatementSummary(statements),
			DurationMs: 0,
			RowCount:   0,
			Error:      err.Error(),
		})
		return err
	}

	startedAt := time.Now()
	tx, err := db.sqlDB.BeginTx(context.Background(), nil)
	if err != nil {
		db.emitQueryEvent(QueryEvent{
			SQL:        txStatementSummary(statements),
			DurationMs: elapsedMs(startedAt),
			RowCount:   0,
			Error:      err.Error(),
		})
		return err
	}

	for i, stmt := range statements {
		if _, err := tx.Exec(stmt.Query, stmt.Args...); err != nil {
			_ = tx.Rollback()
			db.emitQueryEvent(QueryEvent{
				SQL:        txStatementSummary(statements),
				DurationMs: elapsedMs(startedAt),
				RowCount:   0,
				Error:      fmt.Sprintf("statement %d: %v", i+1, err),
			})
			return fmt.Errorf("statement %d: %w", i+1, err)
		}
	}

	if err := tx.Commit(); err != nil {
		db.emitQueryEvent(QueryEvent{
			SQL:        txStatementSummary(statements),
			DurationMs: elapsedMs(startedAt),
			RowCount:   0,
			Error:      err.Error(),
		})
		return err
	}

	db.emitQueryEvent(QueryEvent{
		SQL:        txStatementSummary(statements),
		DurationMs: elapsedMs(startedAt),
		RowCount:   0,
		Error:      "",
	})
	return nil
}

func (db *DB) emitQueryEvent(event QueryEvent) {
	db.hookMu.RLock()
	hook := db.onQuery
	db.hookMu.RUnlock()
	if hook != nil {
		hook(event)
	}
}

func (db *DB) ensureOpen() error {
	if db == nil {
		return fmt.Errorf("sqlite database handle is nil")
	}
	if db.openErr != nil {
		return db.openErr
	}
	if db.sqlDB == nil {
		return fmt.Errorf("sqlite database connection is not initialized")
	}
	return nil
}

func elapsedMs(start time.Time) float64 {
	return time.Since(start).Seconds() * 1000
}

func normalizeColumnValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(typed)
	case bool:
		if typed {
			return int64(1)
		}
		return int64(0)
	default:
		return typed
	}
}

func txStatementSummary(statements []Statement) string {
	if len(statements) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("BEGIN; ")
	for i, stmt := range statements {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(stmt.Query)
	}
	b.WriteString("; COMMIT;")
	return b.String()
}
