package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

type QueryResult struct {
	Columns  []string
	Rows     []map[string]any
	RowCount int
}

func Open(path string) (*sql.DB, error) {
	return sql.Open("duckdb", path)
}

func RunDuckQuery(ctx context.Context, database *sql.DB, query string) (*QueryResult, error) {
	rows, err := database.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := &QueryResult{
		Columns: cols,
		Rows:    []map[string]any{},
	}

	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}

		row := map[string]any{}
		for i, col := range cols {
			row[col] = normalizeValue(values[i])
		}
		result.Rows = append(result.Rows, row)
	}

	result.RowCount = len(result.Rows)
	return result, rows.Err()
}

func normalizeValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(typed)
	case time.Time:
		return typed.Format("2006-01-02 15:04:05.000")
	default:
		return fmt.Sprint(typed)
	}
}
