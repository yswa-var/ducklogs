package ai

import (
	"strings"
	"testing"
)

func TestParseSQLPlanFromEmbeddedJSON(t *testing.T) {
	t.Parallel()

	content := `Here is the plan:

` + "```json" + `
{
  "needs_clarification": false,
  "clarifying_question": "",
  "intent": "find_text_occurrences",
  "sql": "SELECT app_ts, message FROM app_logs WHERE lower(message) LIKE '%vendor not found%' ORDER BY app_ts LIMIT 200",
  "explanation": "Finds matching logs.",
  "report_title": "Vendor Not Found",
  "columns_expected": ["app_ts", "message"]
}
` + "```"

	plan, err := parseSQLPlan(content)
	if err != nil {
		t.Fatalf("parseSQLPlan returned error: %v", err)
	}

	if plan.Intent != "find_text_occurrences" {
		t.Fatalf("intent = %q", plan.Intent)
	}
	if !strings.Contains(plan.SQL, "vendor not found") {
		t.Fatalf("sql did not include expected filter: %q", plan.SQL)
	}
}

func TestParseSQLPlanFallsBackToSQLFence(t *testing.T) {
	t.Parallel()

	content := "```sql\nSELECT * FROM app_logs ORDER BY app_ts LIMIT 200;\n```\n\nJSON response:\n```json\n{\"sql\":\"broken"

	plan, err := parseSQLPlan(content)
	if err != nil {
		t.Fatalf("parseSQLPlan returned error: %v", err)
	}

	if plan.SQL != "SELECT * FROM app_logs ORDER BY app_ts LIMIT 200;" {
		t.Fatalf("sql = %q", plan.SQL)
	}
	if plan.ReportTitle == "" {
		t.Fatalf("report title should be populated")
	}
}
