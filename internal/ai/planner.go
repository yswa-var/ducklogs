package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"ducklogs/internal/sqlsafe"
)

var sqlFenceRe = regexp.MustCompile("(?is)```sql\\s*(.*?)\\s*```")

func GenerateSQLPlan(ctx context.Context, client *OpenRouterClient, userPrompt string) (*SQLPlan, error) {
	content, err := client.ChatJSON(ctx, []ChatMessage{
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
	clean := stripMarkdownFence(strings.TrimSpace(content))

	var plan SQLPlan
	if err := json.Unmarshal([]byte(clean), &plan); err == nil {
		return &plan, nil
	}

	if jsonBlock, ok := extractJSONObject(content); ok {
		if err := json.Unmarshal([]byte(jsonBlock), &plan); err == nil {
			return &plan, nil
		}
	}

	if sql, ok := extractSQLFence(content); ok {
		return &SQLPlan{
			NeedsClarification: false,
			Intent:             "log_analysis_query",
			SQL:                sql,
			Explanation:        "Generated from the SQL block returned by the model.",
			ReportTitle:        "DuckLog Query Results",
			ColumnsExpected:    []string{},
		}, nil
	}

	return nil, fmt.Errorf("failed to parse AI response as SQLPlan JSON\nraw: %s", content)
}

func stripMarkdownFence(content string) string {
	clean := strings.TrimSpace(content)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	return strings.TrimSpace(clean)
}

func extractSQLFence(content string) (string, bool) {
	matches := sqlFenceRe.FindStringSubmatch(content)
	if matches == nil {
		return "", false
	}
	sql := strings.TrimSpace(matches[1])
	if sql == "" {
		return "", false
	}
	return sql, true
}

func extractJSONObject(content string) (string, bool) {
	start := strings.Index(content, "{")
	if start == -1 {
		return "", false
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(content); i++ {
		ch := content[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return content[start : i+1], true
			}
		}
	}

	return "", false
}
