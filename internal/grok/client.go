// Package grok provides access to Grok/xAI through the official
// OpenAI-compatible API.
//
// Two authentication modes:
//
//  1. API Key (api.x.ai)   — uses XAI_API_KEY, calls api.x.ai/v1 directly.
//  2. OAuth + CLI Proxy    — OAuth login via auth.x.ai, then calls
//     cli-chat-proxy.grok.com/v1 with special headers (SuperGrok / X Premium+).
package grok

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ask-cli/ask-cli/internal/grokauth"
)

// Default endpoints.
const (
	DefaultBaseURL   = "https://api.x.ai"
	CLIProxyBaseURL  = "https://cli-chat-proxy.grok.com/v1"
	ChatEndpoint     = "/v1/chat/completions"
	ModelsEndpoint   = "/v1/models"
	DefaultModel     = "grok-4.3"
	DefaultOAuthMode = "grok-build"
)

// ── Types ──

// ChatMessage represents one message in the chat.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is an OpenAI-compatible chat completion request.
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

// ChatResponse is the OpenAI-compatible response from xAI.
type ChatResponse struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Model   string          `json:"model"`
	Choices []ChatChoice    `json:"choices"`
	Usage   *Usage          `json:"usage,omitempty"`
	Raw     json.RawMessage `json:"-"`
}

// ChatChoice represents one completion choice.
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage contains token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Model describes a model available through the xAI API.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse is the list of available models.
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// StreamChunk is a single chunk in a streaming response.
type StreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

// StreamChoice is a choice in a stream chunk.
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

// StreamDelta is the incremental content in a stream chunk.
type StreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// ErrorResponse from xAI.
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
	} `json:"error"`
}

// ── Client ──

// Client handles requests to the xAI (Grok) API.
type Client struct {
	mu         sync.RWMutex
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string

	// OAuth / CLI proxy fields
	dataDir     string
	isCLIProxy  bool
	extraHeader map[string]string // e.g. x-grok-client-version
}

// NewClient creates a Grok API client with the given API key (api.x.ai).
func NewClient(apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    5,
				IdleConnTimeout: 90 * time.Second,
			},
		},
		baseURL: DefaultBaseURL,
		apiKey:  apiKey,
		model:   DefaultModel,
	}
}

// NewCLIProxyClient creates a Grok client backed by OAuth + cli-chat-proxy.
// It resolves the access token from the OAuth store and adds the required
// CLI proxy headers for SuperGrok / X Premium+ access.
func NewCLIProxyClient(dataDir string) *Client {
	token, _ := grokauth.ResolveAccessToken(context.Background(), dataDir)
	headers := grokauth.CLIProxyHeaders(DefaultOAuthMode)

	c := &Client{
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    5,
				IdleConnTimeout: 90 * time.Second,
			},
		},
		baseURL:     CLIProxyBaseURL,
		apiKey:      token,
		model:       DefaultOAuthMode,
		dataDir:     dataDir,
		isCLIProxy:  true,
		extraHeader: headers,
	}

	return c
}

// SetAPIKey sets the API key.
func (c *Client) SetAPIKey(key string) {
	c.mu.Lock()
	c.apiKey = key
	c.mu.Unlock()
}

// SetModel overrides the default model.
func (c *Client) SetModel(model string) {
	c.mu.Lock()
	c.model = model
	c.mu.Unlock()
}

// IsConfigured returns true if an API key is set.
func (c *Client) IsConfigured() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.apiKey != ""
}

// ListModels returns available models from xAI.
func (c *Client) ListModels(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+ModelsEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create models request: %w", err)
	}
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("models request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read models response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var models ModelsResponse
	if err := json.Unmarshal(body, &models); err != nil {
		return nil, fmt.Errorf("parse models: %w", err)
	}
	return models.Data, nil
}

// SendPrompt sends a prompt and returns the full response text.
func (c *Client) SendPrompt(ctx context.Context, prompt string) (string, error) {
	if !c.IsConfigured() {
		return "", fmt.Errorf("grok not configured; set XAI_API_KEY or run 'ask login grok'")
	}

	c.mu.RLock()
	model := c.model
	c.mu.RUnlock()

	req := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+ChatEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	c.setAuthHeader(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try parsing error response
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("xAI error: %s (type=%s)", errResp.Error.Message, errResp.Error.Type)
		}
		return "", fmt.Errorf("xAI API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("xAI returned no choices")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

// SendPromptStream sends a prompt and reads the streaming response,
// returning the full accumulated text.
func (c *Client) SendPromptStream(ctx context.Context, prompt string) (string, error) {
	if !c.IsConfigured() {
		return "", fmt.Errorf("grok not configured; set XAI_API_KEY or run 'ask login grok'")
	}

	c.mu.RLock()
	model := c.model
	c.mu.RUnlock()

	req := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
		Stream: true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+ChatEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	c.setAuthHeader(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("stream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("xAI stream error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var fullText strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimSpace(line[6:])
		if data == "" || data == "[DONE]" {
			continue
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		for _, choice := range chunk.Choices {
			fullText.WriteString(choice.Delta.Content)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("SSE read error: %w", err)
	}

	result := strings.TrimSpace(fullText.String())
	if result == "" {
		return "", fmt.Errorf("xAI returned empty response")
	}
	return result, nil
}

func (c *Client) setAuthHeader(req *http.Request) {
	c.mu.RLock()
	key := c.apiKey
	extra := c.extraHeader
	c.mu.RUnlock()

	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
}
