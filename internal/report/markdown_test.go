package report

import (
	"strings"
	"testing"

	"ducklogs/internal/db"
)

func TestMarkdownTableEscapesAndLimitsRows(t *testing.T) {
	t.Parallel()

	result := &db.QueryResult{
		Columns: []string{"message"},
		Rows: []map[string]any{
			{"message": "vendor | not found\nline two"},
			{"message": "second"},
		},
		RowCount: 2,
	}

	table := MarkdownTable(result, 1)

	if !strings.Contains(table, "vendor \\| not found line two") {
		t.Fatalf("table did not escape cell: %s", table)
	}
	if strings.Contains(table, "second") {
		t.Fatalf("table should only include first row: %s", table)
	}
	if !strings.Contains(table, "Showing 1 of 2 rows") {
		t.Fatalf("table did not include limit notice: %s", table)
	}
}
