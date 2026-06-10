package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
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
}

func Load() Config {
	loadDotEnv(".env")

	return Config{
		Database:         envString("DUCKLOG_DATABASE", DefaultDatabase),
		OpenRouterAPIKey: normalizeAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		OpenRouterURL:    envString("OPENROUTER_BASE_URL", DefaultOpenRouterURL),
		OpenRouterModel:  envString("OPENROUTER_MODEL", DefaultOpenRouterModel),
		Temperature:      envFloat("OPENROUTER_TEMPERATURE", DefaultTemperature),
		MaxRowsForReport: envInt("DUCKLOG_MAX_ROWS_FOR_REPORT", DefaultMaxRowsForReport),
		ReportPath:       envString("DUCKLOG_REPORT_PATH", DefaultReportPath),
	}
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}

		_ = os.Setenv(key, value)
	}
}

func normalizeAPIKey(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "Bearer ")
	return strings.TrimSpace(value)
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
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
