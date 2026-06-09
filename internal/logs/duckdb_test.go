package logs

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDuckDBStoreInsert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := OpenDuckDB(ctx, filepath.Join(t.TempDir(), "ducklogs.duckdb"))
	if err != nil {
		t.Fatalf("OpenDuckDB returned error: %v", err)
	}
	defer store.Close()

	rawLine := "[2026-05-12T13:39:08+05:30] [ecs/Main/stream] 2026-05-12 08:09:08.284 | INFO | app | 1 | 139844682019584 | af01198314485493 | 9341456787191792 | calculate_orchestrator.langgraph_flows.bill.nodes:node_parse_and_validate:278 - [ITERATION_START] tracking_id=af01198314485493 status=INITIAL iteration_count=1"
	event, err := ParseLogLine(1, rawLine)
	if err != nil {
		t.Fatalf("ParseLogLine returned error: %v", err)
	}

	if err := store.Insert(ctx, event); err != nil {
		t.Fatalf("Insert returned error: %v", err)
	}

	var count int
	var module string
	var functionName string
	var lineNumber int
	if err := store.db.QueryRowContext(ctx, `
SELECT count(*), max(module), max(function_name), max(line_number)
FROM app_logs;
`).Scan(&count, &module, &functionName, &lineNumber); err != nil {
		t.Fatalf("query inserted log returned error: %v", err)
	}

	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if module != "calculate_orchestrator.langgraph_flows.bill.nodes" {
		t.Fatalf("module = %q", module)
	}
	if functionName != "node_parse_and_validate" {
		t.Fatalf("function name = %q", functionName)
	}
	if lineNumber != 278 {
		t.Fatalf("line number = %d", lineNumber)
	}
}
