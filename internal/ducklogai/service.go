package ducklogai

import (
	"context"
	"fmt"

	"ducklogs/internal/ai"
	"ducklogs/internal/config"
	"ducklogs/internal/db"
	"ducklogs/internal/report"
)

const MaxQueryRetries = 2

type Options struct {
	Prompt string
	DryRun bool
	Run    bool
	Report string
}

type Result struct {
	Plan       *ai.SQLPlan
	Rows       *db.QueryResult
	Summary    string
	ReportPath string
}

func Ask(ctx context.Context, cfg config.Config, opts Options) (*Result, error) {
	if opts.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	client := &ai.OpenRouterClient{
		APIKey:      cfg.OpenRouterAPIKey,
		BaseURL:     cfg.OpenRouterURL,
		Model:       cfg.OpenRouterModel,
		Temperature: cfg.Temperature,
		HTTPReferer: cfg.HTTPReferer,
		AppTitle:    cfg.AppTitle,
	}

	plan, err := ai.GenerateSQLPlan(ctx, client, opts.Prompt)
	if err != nil {
		return nil, err
	}

	result := &Result{Plan: plan}
	if plan.NeedsClarification || opts.DryRun || (!opts.Run && opts.Report == "") {
		return result, nil
	}

	database, err := db.Open(cfg.Database)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	rows, err := db.RunDuckQuery(ctx, database, plan.SQL)
	for attempt := 0; err != nil && attempt < MaxQueryRetries; attempt++ {
		repairedPlan, repairErr := ai.RepairSQLPlan(ctx, client, opts.Prompt, plan.SQL, err)
		if repairErr != nil {
			return nil, repairErr
		}
		plan = repairedPlan
		result.Plan = repairedPlan
		rows, err = db.RunDuckQuery(ctx, database, repairedPlan.SQL)
	}
	if err != nil {
		return nil, err
	}

	result.Rows = rows

	if opts.Report != "" {
		summary, err := ai.GenerateSummary(ctx, client, opts.Prompt, plan, rows, cfg.MaxRowsForReport)
		if err != nil {
			return nil, err
		}
		result.Summary = summary

		if err := report.WriteMarkdownReport(opts.Report, opts.Prompt, plan, rows, summary, cfg.MaxRowsForReport); err != nil {
			return nil, err
		}
		result.ReportPath = opts.Report
	}

	return result, nil
}
