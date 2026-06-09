package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ducklogs/internal/sqlsafe"
)

func GenerateSQLPlan(ctx context.Context, client *OpenRouterClient, userPrompt string) (*SQLPlan, error) {
	content, err := client.Chat(ctx, []ChatMessage{
		{Role: "system", Content: SQLSystemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return nil, err
	}

	plan, err := parseSQLPlan(content)
	if err != nil {
		return nil, err
	}

	if plan.NeedsClarification {
		return plan, nil
	}
	if err := sqlsafe.ValidateReadOnlySQL(plan.SQL); err != nil {
		return nil, err
	}

	return plan, nil
}

func RepairSQLPlan(ctx context.Context, client *OpenRouterClient, userPrompt, badSQL string, duckErr error) (*SQLPlan, error) {
	repairPrompt := fmt.Sprintf(`The generated DuckDB SQL failed.

User request:
%s

Bad SQL:
%s

DuckDB error:
%s

Fix the SQL using only the app_logs schema.
Return the same JSON shape.`, userPrompt, badSQL, duckErr.Error())

	return GenerateSQLPlan(ctx, client, repairPrompt)
}

func parseSQLPlan(content string) (*SQLPlan, error) {
	clean := strings.TrimSpace(content)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	var plan SQLPlan
	if err := json.Unmarshal([]byte(clean), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse AI JSON: %w\nraw: %s", err, content)
	}

	return &plan, nil
}
