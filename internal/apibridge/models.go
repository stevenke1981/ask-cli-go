// Package apibridge provides an OpenAI-compatible API layer that routes
// requests to different AI providers (ChatGPT, Gemini, Grok) using their
// web subscription credentials.
package apibridge

import (
	"encoding/json"
	"fmt"
)

// ── OpenAI-compatible request types ──

// ChatMessage represents one message in the chat completion request.
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

// ChatResponse is an OpenAI-compatible chat completion response.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice represents one completion choice.
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage contains token usage information.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ── Provider routing ──

// Provider identifies an AI service provider.
type Provider string

const (
	ProviderChatGPT Provider = "chatgpt"
	ProviderGemini  Provider = "gemini"
	ProviderGrok    Provider = "grok"
)

// ModelRoute maps a model name to a provider and its internal model ID.
type ModelRoute struct {
	Provider    Provider
	ServiceName string // provider display name
	InternalID  string // provider-specific model identifier
}

// ModelRegistry contains all supported models and their routing.
var ModelRegistry = map[string]ModelRoute{
	// ChatGPT models
	"gpt-4o":          {Provider: ProviderChatGPT, InternalID: "text-davinci-002-render-sha"},
	"gpt-4o-mini":     {Provider: ProviderChatGPT, InternalID: "text-davinci-002-render-sha"},
	"chatgpt-4o":      {Provider: ProviderChatGPT, InternalID: "gpt-4o"},
	"chatgpt-4o-mini": {Provider: ProviderChatGPT, InternalID: "gpt-4o-mini"},

	// Gemini models
	"gemini-2.5-pro":   {Provider: ProviderGemini, InternalID: "gemini-2.5-pro"},
	"gemini-2.5-flash": {Provider: ProviderGemini, InternalID: "gemini-2.5-flash"},
	"gemini-2.0-flash": {Provider: ProviderGemini, InternalID: "gemini-2.0-flash"},

	// Grok models
	"grok-4": {Provider: ProviderGrok, InternalID: "grok-4-0709"},
	"grok-3": {Provider: ProviderGrok, InternalID: "grok-3-auto"},
	"grok-2": {Provider: ProviderGrok, InternalID: "grok-2-1212"},
}

// ResolveModel finds the route for a given model name.
// Falls back: model prefix matching, then default for provider.
func ResolveModel(model string) (ModelRoute, bool) {
	if route, ok := ModelRegistry[model]; ok {
		return route, true
	}

	// Try prefix matching: "gpt-4" → ChatGPT, "gemini" → Gemini, "grok" → Grok
	for prefix, provider := range prefixMatchers {
		if len(model) >= len(prefix) && model[:len(prefix)] == prefix {
			for _, route := range ModelRegistry {
				if route.Provider == provider {
					r := route
					r.InternalID = model
					return r, true
				}
			}
		}
	}
	return ModelRoute{}, false
}

var prefixMatchers = map[string]Provider{
	"gpt":     ProviderChatGPT,
	"chatgpt": ProviderChatGPT,
	"gemini":  ProviderGemini,
	"grok":    ProviderGrok,
}

// ── Format conversion ──

// ConvertToSSE serializes a chunk as OpenAI SSE data line.
func ConvertToSSE(chunk map[string]any) (string, error) {
	data, err := json.Marshal(chunk)
	if err != nil {
		return "", fmt.Errorf("marshal SSE chunk: %w", err)
	}
	return fmt.Sprintf("data: %s\n\n", string(data)), nil
}

// SSEDone is the standard OpenAI SSE termination signal.
const SSEDone = "data: [DONE]\n\n"
