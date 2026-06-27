package grok

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("test-key")
	if c == nil {
		t.Fatal("NewClient() returned nil")
	}
	if !c.IsConfigured() {
		t.Fatal("client should be configured with API key")
	}
}

func TestClientNotConfigured(t *testing.T) {
	c := NewClient("")
	if c.IsConfigured() {
		t.Fatal("client should not be configured without API key")
	}
}

func TestSetAPIKey(t *testing.T) {
	c := NewClient("")
	c.SetAPIKey("new-key")
	if !c.IsConfigured() {
		t.Fatal("client should be configured after SetAPIKey")
	}
}

func TestSetModel(t *testing.T) {
	c := NewClient("test-key")
	c.SetModel("grok-4")
	// No getter for model, but it shouldn't panic
}

func TestSendPromptWithoutKey(t *testing.T) {
	c := NewClient("")
	_, err := c.SendPrompt(nil, "hello")
	if err == nil {
		t.Fatal("expected error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("error = %q, want 'not configured'", err.Error())
	}
}

func TestListModelsWithoutKey(t *testing.T) {
	c := NewClient("")
	_, err := c.ListModels(nil)
	if err == nil {
		t.Fatal("expected error for unconfigured client")
	}
}

func TestChatRequestJSON(t *testing.T) {
	// Test that ChatRequest marshals correctly
	req := ChatRequest{
		Model: "grok-4.3",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream:    true,
		MaxTokens: 100,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	s := string(data)
	if !strings.Contains(s, DefaultModel) {
		t.Fatalf("JSON = %s, should contain model", s)
	}
	if !strings.Contains(s, "Hello") {
		t.Fatalf("JSON = %s, should contain message", s)
	}
}
