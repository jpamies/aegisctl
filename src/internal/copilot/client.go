// Package copilot provides a client for the GitHub Models API (GitHub Copilot)
// using only Go stdlib (net/http + encoding/json). It enables LLM-powered
// analysis, recommendations, and IaC generation in aegisctl.
//
// Authentication: requires a GITHUB_TOKEN environment variable with access
// to GitHub Models. Falls back to heuristic mode if unavailable.
package copilot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// DefaultEndpoint is the GitHub Models inference endpoint.
	DefaultEndpoint = "https://models.inference.ai.azure.com/chat/completions"

	// DefaultModel is the default model to use.
	DefaultModel = "gpt-4o"

	// DefaultTimeout for API calls.
	DefaultTimeout = 120 * time.Second

	// DefaultMaxTokens for responses.
	DefaultMaxTokens = 4096
)

// Client is a GitHub Models API client.
type Client struct {
	endpoint   string
	model      string
	token      string
	httpClient *http.Client
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest is the API request body.
type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature"`
}

// chatResponse is the API response body.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *apiError `json:"error,omitempty"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// Option configures the client.
type Option func(*Client)

// WithEndpoint sets a custom API endpoint.
func WithEndpoint(endpoint string) Option {
	return func(c *Client) { c.endpoint = endpoint }
}

// WithModel sets the model to use.
func WithModel(model string) Option {
	return func(c *Client) { c.model = model }
}

// WithToken sets the auth token explicitly (instead of reading from env).
func WithToken(token string) Option {
	return func(c *Client) { c.token = token }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// NewClient creates a new GitHub Models API client.
// Reads AEGIS_GITHUB_TOKEN (preferred) or GITHUB_TOKEN (fallback) from the
// environment unless provided explicitly via WithToken.
// Returns an error if no token is available.
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		endpoint: DefaultEndpoint,
		model:    DefaultModel,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.token == "" {
		c.token = resolveToken()
	}
	if c.token == "" {
		return nil, fmt.Errorf("no token found — set AEGIS_GITHUB_TOKEN (or GITHUB_TOKEN) or use WithToken()")
	}

	return c, nil
}

// resolveToken returns the first non-empty value among AEGIS_GITHUB_TOKEN and
// GITHUB_TOKEN. Using a dedicated variable avoids conflicts with the
// auto-generated GITHUB_TOKEN in GitHub Actions (which lacks Models access).
func resolveToken() string {
	if t := os.Getenv("AEGIS_GITHUB_TOKEN"); t != "" {
		return t
	}
	return os.Getenv("GITHUB_TOKEN")
}

// IsAvailable checks whether a usable token is configured.
func IsAvailable() bool {
	return resolveToken() != ""
}

// Chat sends a chat completion request and returns the response text.
func (c *Client) Chat(messages []Message) (string, error) {
	return c.ChatWithOptions(messages, DefaultMaxTokens, 0.2)
}

// ChatWithOptions sends a chat completion with custom parameters.
func (c *Client) ChatWithOptions(messages []Message, maxTokens int, temperature float64) (string, error) {
	reqBody := chatRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s (%s)", chatResp.Error.Message, chatResp.Error.Code)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API returned no choices")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

// ChatJSON sends a chat completion and parses the response as JSON into target.
// The prompt should instruct the model to respond with valid JSON only.
func (c *Client) ChatJSON(messages []Message, target interface{}) error {
	raw, err := c.Chat(messages)
	if err != nil {
		return err
	}

	// Strip markdown code fences if present
	cleaned := raw
	if idx := strings.Index(cleaned, "```json"); idx >= 0 {
		cleaned = cleaned[idx+7:]
	} else if idx := strings.Index(cleaned, "```"); idx >= 0 {
		cleaned = cleaned[idx+3:]
	}
	if idx := strings.LastIndex(cleaned, "```"); idx >= 0 {
		cleaned = cleaned[:idx]
	}
	cleaned = strings.TrimSpace(cleaned)

	if err := json.Unmarshal([]byte(cleaned), target); err != nil {
		return fmt.Errorf("parsing JSON response: %w\nRaw response:\n%s", err, raw)
	}

	return nil
}
