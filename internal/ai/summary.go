package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"ducklogs/internal/db"
)

func GenerateSummary(ctx context.Context, client *OpenRouterClient, prompt string, plan *SQLPlan, result *db.QueryResult, maxRows int) (string, error) {
	rows := result.Rows
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}

	rowsJSON, _ := json.MarshalIndent(rows, "", "  ")
	content := fmt.Sprintf(`User request:
%s

SQL:
%s

Rows returned: %d

Rows JSON:
%s

Write a concise Markdown analysis.
Rules:
- Only use evidence from the rows.
- Mention if no rows were found.
- Do not invent causes.
- Include "Next checks" if useful.
`, prompt, plan.SQL, result.RowCount, string(rowsJSON))

	return client.Chat(ctx, []ChatMessage{
		{Role: "system", Content: SummarySystemPrompt},
		{Role: "user", Content: content},
	})
}
