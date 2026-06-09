package logs

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

type DuckDBStore struct {
	db *sql.DB
}

func OpenDuckDB(ctx context.Context, path string) (*DuckDBStore, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, err
	}

	store := &DuckDBStore{db: db}
	if err := store.createSchema(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *DuckDBStore) Close() error {
	return s.db.Close()
}

func (s *DuckDBStore) createSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS app_logs (
	line_no BIGINT,
	cloudwatch_ts TEXT,
	log_stream TEXT,
	app_ts TIMESTAMP,
	level TEXT,
	app_name TEXT,
	process_id TEXT,
	thread_id TEXT,
	tracking_id TEXT,
	realm_id TEXT,
	source_location TEXT,
	module TEXT,
	function_name TEXT,
	line_number INTEGER,
	event_name TEXT,
	status TEXT,
	iteration_count INTEGER,
	message TEXT,
	attrs JSON,
	raw_line TEXT
);
`)
	return err
}

func (s *DuckDBStore) Insert(ctx context.Context, event LogEvent) error {
	return s.InsertBatch(ctx, []LogEvent{event})
}

func (s *DuckDBStore) InsertBatch(ctx context.Context, events []LogEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO app_logs (
	line_no,
	cloudwatch_ts,
	log_stream,
	app_ts,
	level,
	app_name,
	process_id,
	thread_id,
	tracking_id,
	realm_id,
	source_location,
	module,
	function_name,
	line_number,
	event_name,
	status,
	iteration_count,
	message,
	attrs,
	raw_line
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CAST(? AS JSON), ?);
`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, event := range events {
		if err := insertLogEvent(ctx, stmt, event); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func insertLogEvent(ctx context.Context, stmt *sql.Stmt, event LogEvent) error {
	_, err := stmt.ExecContext(ctx,
		event.LineNo,
		event.CloudwatchTS,
		event.LogStream,
		event.AppTS,
		event.Level,
		event.AppName,
		event.ProcessID,
		event.ThreadID,
		nullString(event.TrackingID),
		nullString(event.RealmID),
		event.SourceLocation,
		event.Module,
		nullString(event.FunctionName),
		nullInt(event.LineNumber),
		nullString(event.EventName),
		nullString(event.Status),
		nullInt(event.IterationCount),
		event.Message,
		event.AttrsJSON,
		event.RawLine,
	)
	if err != nil {
		return fmt.Errorf("insert log line %d: %w", event.LineNo, err)
	}

	return nil
}

func nullString(value string) sql.NullString {
	return sql.NullString{
		String: value,
		Valid:  value != "",
	}
}

func nullInt(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}

	return sql.NullInt64{
		Int64: int64(*value),
		Valid: true,
	}
}
