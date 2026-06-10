package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenRouterClient struct {
	APIKey      string
	BaseURL     string
	Model       string
	Temperature float64
	HTTP        *http.Client
}

func (c *OpenRouterClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.chat(ctx, messages, nil)
}

func (c *OpenRouterClient) ChatJSON(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.chat(ctx, messages, &ResponseFormat{Type: "json_object"})
}

func (c *OpenRouterClient) chat(ctx context.Context, messages []ChatMessage, responseFormat *ResponseFormat) (string, error) {
	apiKey := strings.TrimSpace(strings.TrimPrefix(c.APIKey, "Bearer "))
	if apiKey == "" || isPlaceholderAPIKey(apiKey) {
		return "", fmt.Errorf("OPENROUTER_API_KEY is required")
	}

	body := ChatRequest{
		Model:          c.Model,
		Messages:       messages,
		Temperature:    c.Temperature,
		ResponseFormat: responseFormat,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(b))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized {
			return "", fmt.Errorf("OpenRouter authentication failed (401). Check OPENROUTER_API_KEY in your environment or .env")
		}
		return "", fmt.Errorf("openrouter error: %s", string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", err
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty OpenRouter response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func isPlaceholderAPIKey(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "..." ||
		normalized == "your_api_key_here" ||
		normalized == "replace_me" ||
		normalized == "changeme"
}
