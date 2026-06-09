package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"ducklogs/internal/app"
	"ducklogs/internal/config"
	"ducklogs/internal/ducklogai"
	"ducklogs/internal/report"
)

func Run() error {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "tui" {
		return app.Run()
	}

	switch args[0] {
	case "ask":
		return runAsk(args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runAsk(args []string) error {
	cfg := config.Load()
	flags := flag.NewFlagSet("ask", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	dryRun := flags.Bool("dry-run", false, "generate and validate SQL without running it")
	runQuery := flags.Bool("run", false, "run the generated SQL")
	reportPath := flags.String("report", "", "write a Markdown report to this path")
	dbPath := flags.String("db", cfg.Database, "DuckDB database path")

	if err := flags.Parse(args); err != nil {
		return err
	}

	prompt := strings.TrimSpace(strings.Join(flags.Args(), " "))
	if prompt == "" {
		return fmt.Errorf("ask requires a prompt")
	}

	cfg.Database = *dbPath
	result, err := ducklogai.Ask(context.Background(), cfg, ducklogai.Options{
		Prompt: prompt,
		DryRun: *dryRun,
		Run:    *runQuery || *reportPath != "",
		Report: *reportPath,
	})
	if err != nil {
		return err
	}

	if result.Plan.NeedsClarification {
		fmt.Printf("Clarification needed:\n%s\n", result.Plan.ClarifyingQuestion)
		return nil
	}

	fmt.Printf("Intent: %s\n\n", result.Plan.Intent)
	fmt.Printf("Generated SQL:\n%s\n\n", result.Plan.SQL)
	if result.Plan.Explanation != "" {
		fmt.Printf("Explanation:\n%s\n\n", result.Plan.Explanation)
	}

	if result.Rows != nil {
		fmt.Printf("Rows: %d\n\n", result.Rows.RowCount)
		fmt.Print(report.MarkdownTable(result.Rows, cfg.MaxRowsForReport))
		fmt.Println()
	}

	if result.ReportPath != "" {
		fmt.Printf("\nReport written:\n%s\n", result.ReportPath)
	}

	return nil
}

func printUsage() {
	fmt.Println(`ducklog

Usage:
  ducklog tui
  ducklog ask "find vendor not found where tracking id is ..." --dry-run
  ducklog ask "find vendor not found where tracking id is ..." --run
  ducklog ask "find vendor not found where tracking id is ..." --report vendor-not-found.md

Environment:
  OPENROUTER_API_KEY is required for ask/report flows.
  DUCKLOG_DATABASE defaults to ./ducklogs.duckdb.`)
}
