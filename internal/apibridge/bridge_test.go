package apibridge

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// mockProvider implements ProviderClient for testing.
type mockProvider struct {
	name           string
	authenticated  bool
	promptResponse string
	promptError    error
}

func (m *mockProvider) SendPrompt(ctx context.Context, prompt string) (string, error) {
	if m.promptError != nil {
		return "", m.promptError
	}
	return m.promptResponse, nil
}

func (m *mockProvider) IsAuthenticated() bool { return m.authenticated }
func (m *mockProvider) Name() string          { return m.name }

func TestModelRegistry(t *testing.T) {
	cases := []struct {
		model    string
		wantProv Provider
		wantOK   bool
	}{
		{"gpt-4o", ProviderChatGPT, true},
		{"gemini-2.5-pro", ProviderGemini, true},
		{"grok-3", ProviderGrok, true},
		{"unknown-model", "", false},
	}

	for _, tc := range cases {
		route, ok := ResolveModel(tc.model)
		if ok != tc.wantOK {
			t.Errorf("ResolveModel(%q) ok=%v, want %v", tc.model, ok, tc.wantOK)
			continue
		}
		if ok && route.Provider != tc.wantProv {
			t.Errorf("ResolveModel(%q) provider=%q, want %q", tc.model, route.Provider, tc.wantProv)
		}
	}
}

func TestPrefixMatching(t *testing.T) {
	cases := []struct {
		model    string
		wantProv Provider
		wantOK   bool
	}{
		{"gpt-anything", ProviderChatGPT, true},
		{"gemini-experimental", ProviderGemini, true},
		{"grok-latest", ProviderGrok, true},
	}

	for _, tc := range cases {
		route, ok := ResolveModel(tc.model)
		if ok != tc.wantOK {
			t.Errorf("ResolveModel(%q) ok=%v, want %v", tc.model, ok, tc.wantOK)
			continue
		}
		if ok && route.Provider != tc.wantProv {
			t.Errorf("ResolveModel(%q) provider=%q, want %q", tc.model, route.Provider, tc.wantProv)
		}
	}
}

func TestServerRoutesToCorrectProvider(t *testing.T) {
	srv := NewServer()
	srv.RegisterProvider("chatgpt", &mockProvider{
		name: "ChatGPT", authenticated: true, promptResponse: "chatgpt response",
	}, "gpt-4o")
	srv.RegisterProvider("gemini", &mockProvider{
		name: "Gemini", authenticated: true, promptResponse: "gemini response",
	}, "gemini-2.5-pro")

	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// Test ChatGPT routing
	resp := sendChatRequest(t, srv.BaseURL(), "gpt-4o", "hello")
	if resp != "chatgpt response" {
		t.Fatalf("chatgpt response = %q", resp)
	}

	// Test Gemini routing
	resp = sendChatRequest(t, srv.BaseURL(), "gemini-2.5-pro", "hello")
	if resp != "gemini response" {
		t.Fatalf("gemini response = %q", resp)
	}
}

func TestServerListsModels(t *testing.T) {
	srv := NewServer()
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	resp, err := http.Get(srv.BaseURL() + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var body struct {
		Object string `json:"object"`
		Data   []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) == 0 {
		t.Fatal("no models returned")
	}
}

func TestServerHealth(t *testing.T) {
	srv := NewServer()
	srv.RegisterProvider("chatgpt", &mockProvider{
		name: "ChatGPT", authenticated: true,
	})
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	resp, err := http.Get(srv.BaseURL() + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestServerUnknownModel(t *testing.T) {
	srv := NewServer()
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	resp, err := http.Post(srv.BaseURL()+"/v1/chat/completions", "application/json",
		bytes.NewReader([]byte(`{"model":"unknown","messages":[{"role":"user","content":"hi"}]}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestServerNoProvider(t *testing.T) {
	srv := NewServer()
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	resp, err := http.Post(srv.BaseURL()+"/v1/chat/completions", "application/json",
		bytes.NewReader([]byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func sendChatRequest(t *testing.T, baseURL, model, prompt string) string {
	t.Helper()
	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	data, _ := json.Marshal(body)

	resp, err := http.Post(baseURL+"/v1/chat/completions", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(chatResp.Choices) == 0 {
		t.Fatal("no choices")
	}
	return chatResp.Choices[0].Message.Content
}
