package report

import (
	"fmt"
	"os"
	"strings"

	"ducklogs/internal/ai"
	"ducklogs/internal/db"
)

type MarkdownReport struct {
	Title       string
	UserPrompt  string
	SQL         string
	Summary     string
	Findings    []string
	Rows        *db.QueryResult
	Limitations []string
}

func WriteMarkdownReport(path, prompt string, plan *ai.SQLPlan, result *db.QueryResult, summary string, maxRows int) error {
	var b strings.Builder

	title := plan.ReportTitle
	if title == "" {
		title = "DuckLog Report"
	}

	b.WriteString("# " + title + "\n\n")
	b.WriteString("## User Prompt\n\n")
	b.WriteString("```text\n" + prompt + "\n```\n\n")
	b.WriteString("## Generated SQL\n\n")
	b.WriteString("```sql\n" + plan.SQL + "\n```\n\n")
	b.WriteString("## Summary\n\n")
	b.WriteString(summary + "\n\n")
	b.WriteString("## Result Count\n\n")
	b.WriteString(fmt.Sprintf("%d rows\n\n", result.RowCount))
	b.WriteString("## Results\n\n")
	b.WriteString(MarkdownTable(result, maxRows))
	b.WriteString("\n\n")
	b.WriteString("## Limitations\n\n")
	b.WriteString("- This report is generated from the local DuckDB data only.\n")
	b.WriteString("- The AI summary is based only on the query result shown above.\n")
	b.WriteString("- Raw logs should be checked before making production decisions.\n")

	return os.WriteFile(path, []byte(b.String()), 0644)
}

func MarkdownTable(result *db.QueryResult, limit int) string {
	if len(result.Columns) == 0 {
		return "_No columns returned._\n"
	}

	max := result.RowCount
	if max > limit {
		max = limit
	}

	var b strings.Builder
	b.WriteString("|")
	for _, col := range result.Columns {
		b.WriteString(" " + escapeCell(col) + " |")
	}
	b.WriteString("\n|")
	for range result.Columns {
		b.WriteString(" --- |")
	}
	b.WriteString("\n")

	for i := 0; i < max; i++ {
		b.WriteString("|")
		for _, col := range result.Columns {
			val := fmt.Sprintf("%v", result.Rows[i][col])
			if result.Rows[i][col] == nil {
				val = ""
			}
			if len(val) > 160 {
				val = val[:160] + "..."
			}
			b.WriteString(" " + escapeCell(val) + " |")
		}
		b.WriteString("\n")
	}

	if result.RowCount > limit {
		b.WriteString(fmt.Sprintf("\n_Showing %d of %d rows._\n", limit, result.RowCount))
	}

	return b.String()
}

func escapeCell(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}
