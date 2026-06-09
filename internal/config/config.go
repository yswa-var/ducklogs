package config

import (
	"os"
	"strconv"
)

const (
	DefaultDatabase         = "./ducklogs.duckdb"
	DefaultOpenRouterURL    = "https://openrouter.ai/api/v1/chat/completions"
	DefaultOpenRouterModel  = "qwen/qwen-2.5-coder-32b-instruct"
	DefaultTemperature      = 0.1
	DefaultMaxRowsForReport = 50
	DefaultReportPath       = "ducklog-report.md"
)

type Config struct {
	Database         string
	OpenRouterAPIKey string
	OpenRouterURL    string
	OpenRouterModel  string
	Temperature      float64
	MaxRowsForReport int
	ReportPath       string
	HTTPReferer      string
	AppTitle         string
}

func Load() Config {
	return Config{
		Database:         envString("DUCKLOG_DATABASE", DefaultDatabase),
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterURL:    envString("OPENROUTER_BASE_URL", DefaultOpenRouterURL),
		OpenRouterModel:  envString("OPENROUTER_MODEL", DefaultOpenRouterModel),
		Temperature:      envFloat("OPENROUTER_TEMPERATURE", DefaultTemperature),
		MaxRowsForReport: envInt("DUCKLOG_MAX_ROWS_FOR_REPORT", DefaultMaxRowsForReport),
		ReportPath:       envString("DUCKLOG_REPORT_PATH", DefaultReportPath),
		HTTPReferer:      envString("OPENROUTER_HTTP_REFERER", "https://github.com/yourname/ducklog"),
		AppTitle:         envString("OPENROUTER_APP_TITLE", "ducklog"),
	}
}

func envString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
