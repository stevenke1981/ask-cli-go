// Package apibridge provides an OpenAI-compatible API server that routes
// chat completion requests to multiple AI providers (ChatGPT, Gemini, Grok)
// using web subscription credentials.
//
// This implements the architecture described in §5 and §8 of
// web-ai-api-bridge-research.md: a local API bridge that converts the
// OpenAI Chat Completions format to each provider's private API format.
package apibridge

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// ── Provider interface ──

// ProviderClient is the interface each AI provider implements.
type ProviderClient interface {
	// SendPrompt sends a prompt and returns the response text.
	SendPrompt(ctx context.Context, prompt string) (string, error)
	// IsAuthenticated returns whether the provider has valid credentials.
	IsAuthenticated() bool
	// Name returns the provider name for display.
	Name() string
}

// ── Bridge server ──

// Server is a lightweight OpenAI-compatible API server that routes
// requests to the appropriate provider.
type Server struct {
	mu           sync.RWMutex
	providers    map[string]ProviderClient
	modelMapping map[string]string // model → provider ID

	listener   net.Listener
	httpServer *http.Server
	port       int
}

// NewServer creates an API bridge server.
func NewServer() *Server {
	return &Server{
		providers:    make(map[string]ProviderClient),
		modelMapping: make(map[string]string),
	}
}

// RegisterProvider adds a provider to the bridge.
func (s *Server) RegisterProvider(id string, client ProviderClient, models ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[id] = client
	for _, model := range models {
		s.modelMapping[model] = id
	}
}

// Start binds the bridge to an ephemeral IPv4 loopback port.
func (s *Server) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen on loopback: %w", err)
	}
	s.listener = listener
	s.port = listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	mux.HandleFunc("/v1/models", s.handleListModels)
	mux.HandleFunc("/health", s.handleHealth)
	s.httpServer = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[apibridge] serve error: %v", err)
		}
	}()
	if done := ctx.Done(); done != nil {
		go func() {
			<-done
			_ = s.Close()
		}()
	}
	return nil
}

// Port returns the listening port.
func (s *Server) Port() int { return s.port }

// BaseURL returns the base URL of the running server.
func (s *Server) BaseURL() string {
	if s.listener == nil {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d", s.port)
}

// Close stops the server.
func (s *Server) Close() error {
	if s.httpServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := s.httpServer.Shutdown(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// ── HTTP handlers ──

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	setJSONHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Messages) == 0 {
		writeJSONError(w, http.StatusBadRequest, "messages is required")
		return
	}

	// Resolve model → provider
	route, ok := ResolveModel(req.Model)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("unknown model: %s", req.Model))
		return
	}

	s.mu.RLock()
	provider, ok := s.providers[string(route.Provider)]
	s.mu.RUnlock()
	if !ok {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("provider %q is not registered", route.Provider))
		return
	}

	// Build prompt from messages
	prompt := buildPrompt(req.Messages)
	if prompt == "" {
		writeJSONError(w, http.StatusBadRequest, "no user message found")
		return
	}

	// Send to provider
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	response, err := provider.SendPrompt(ctx, prompt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("%s error: %v", provider.Name(), err))
		return
	}

	// Build OpenAI-compatible response
	resp := ChatResponse{
		ID:     "chatcmpl-" + randomID(12),
		Object: "chat.completion",
		Model:  req.Model,
		Choices: []Choice{
			{
				Index:        0,
				Message:      ChatMessage{Role: "assistant", Content: response},
				FinishReason: "stop",
			},
		},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	setJSONHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	type modelEntry struct {
		ID     string `json:"id"`
		Object string `json:"object"`
	}

	models := make([]modelEntry, 0, len(ModelRegistry))
	for id := range ModelRegistry {
		models = append(models, modelEntry{ID: id, Object: "model"})
	}

	resp := map[string]any{
		"object": "list",
		"data":   models,
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	setJSONHeaders(w)
	s.mu.RLock()
	providers := make([]string, 0, len(s.providers))
	for id, p := range s.providers {
		status := "authenticated"
		if !p.IsAuthenticated() {
			status = "unauthenticated"
		}
		providers = append(providers, fmt.Sprintf("%s:%s", id, status))
	}
	s.mu.RUnlock()

	json.NewEncoder(w).Encode(map[string]any{
		"status":    "ok",
		"providers": providers,
	})
}

// ── Helpers ──

func setJSONHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func buildPrompt(messages []ChatMessage) string {
	// Extract the last user message as the prompt.
	// For more complex conversations, this would need proper merging.
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	return ""
}

func randomID(length int) string {
	data := make([]byte, length)
	_, _ = rand.Read(data)
	return hex.EncodeToString(data)
}
