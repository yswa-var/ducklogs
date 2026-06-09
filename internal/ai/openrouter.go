package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenRouterClient struct {
	APIKey      string
	BaseURL     string
	Model       string
	Temperature float64
	HTTPReferer string
	AppTitle    string
	HTTP        *http.Client
}

func (c *OpenRouterClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY is required")
	}

	body := ChatRequest{
		Model:       c.Model,
		Messages:    messages,
		Temperature: c.Temperature,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(b))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if c.HTTPReferer != "" {
		req.Header.Set("HTTP-Referer", c.HTTPReferer)
	}
	if c.AppTitle != "" {
		req.Header.Set("X-Title", c.AppTitle)
	}

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
