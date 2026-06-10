package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"ducklogs/internal/ai"
	"ducklogs/internal/config"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := &ai.OpenRouterClient{
		APIKey:      cfg.OpenRouterAPIKey,
		BaseURL:     cfg.OpenRouterURL,
		Model:       cfg.OpenRouterModel,
		Temperature: 0,
	}

	content, err := client.Chat(ctx, []ai.ChatMessage{
		{Role: "system", Content: "Reply with exactly: ok"},
		{Role: "user", Content: "credential check"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "OpenRouter credential check failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("OpenRouter credential check passed.")
	fmt.Printf("Model: %s\n", cfg.OpenRouterModel)
	fmt.Printf("Response: %s\n", strings.TrimSpace(content))
}
