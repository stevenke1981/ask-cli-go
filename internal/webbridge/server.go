// Package webbridge provides a one-shot loopback bridge between ask-cli and
// the ChatGPT normal-Chrome extension.
package webbridge

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const maxResultBodyBytes = 2 << 20

// Task is one prompt handed to the Chrome extension.
type Task struct {
	ID        string `json:"id"`
	Prompt    string `json:"prompt"`
	TimeoutMS int64  `json:"timeout_ms"`
}

// Result is the response returned by the Chrome extension.
type Result struct {
	ID      string `json:"id"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Server hosts one authenticated prompt task on an ephemeral loopback port.
type Server struct {
	task  Task
	token string

	listener   net.Listener
	httpServer *http.Server

	claimOnce sync.Once
	claimed   chan struct{}
	results   chan Result
}

// NewServer creates a bridge for one prompt.
func NewServer(prompt string, timeout time.Duration) (*Server, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("prompt must not be empty")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be greater than zero")
	}

	token, err := randomHex(32)
	if err != nil {
		return nil, fmt.Errorf("generate bridge token: %w", err)
	}
	taskID, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generate task ID: %w", err)
	}

	return &Server{
		task: Task{
			ID:        taskID,
			Prompt:    prompt,
			TimeoutMS: timeout.Milliseconds(),
		},
		token:   token,
		claimed: make(chan struct{}),
		results: make(chan Result, 1),
	}, nil
}

// Start binds the bridge to an ephemeral IPv4 loopback port.
func (s *Server) Start(ctx context.Context) error {
	if s.listener != nil {
		return fmt.Errorf("bridge already started")
	}

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen on loopback: %w", err)
	}
	s.listener = listener

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/task", s.handleTask)
	mux.HandleFunc("/v1/result", s.handleResult)
	s.httpServer = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		_ = s.httpServer.Serve(listener)
	}()
	if done := ctx.Done(); done != nil {
		go func() {
			<-done
			_ = s.Close(context.Background())
		}()
	}
	return nil
}

// TriggerURL opens ChatGPT while handing bridge coordinates to the extension
// through a URL fragment, which is never sent to the remote server.
func (s *Server) TriggerURL() string {
	if s.listener == nil {
		return ""
	}
	_, port, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil {
		return ""
	}

	values := make(url.Values)
	values.Set("ask-cli-token", s.token)
	values.Set("ask-cli-port", port)
	return "https://chatgpt.com/#" + values.Encode()
}

// WaitForClaim waits until the extension authenticates and retrieves the task.
func (s *Server) WaitForClaim(ctx context.Context) error {
	select {
	case <-s.claimed:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for extension: %w", ctx.Err())
	}
}

// WaitForResult waits for the extension's matching result.
func (s *Server) WaitForResult(ctx context.Context) (Result, error) {
	select {
	case result := <-s.results:
		return result, nil
	case <-ctx.Done():
		return Result{}, fmt.Errorf("wait for ChatGPT response: %w", ctx.Err())
	}
}

// Close stops the loopback server.
func (s *Server) Close(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	err := s.httpServer.Shutdown(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) baseURL() string {
	if s.listener == nil {
		return ""
	}
	return "http://" + s.listener.Addr().String()
}

func (s *Server) handleTask(w http.ResponseWriter, r *http.Request) {
	s.setBridgeHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authenticate(r.URL.Query().Get("token")) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	s.claimOnce.Do(func() {
		close(s.claimed)
	})
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.task); err != nil {
		http.Error(w, "encode task", http.StatusInternalServerError)
	}
}

func (s *Server) handleResult(w http.ResponseWriter, r *http.Request) {
	s.setBridgeHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authenticate(r.URL.Query().Get("token")) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxResultBodyBytes)
	defer r.Body.Close()
	var result Result
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, "invalid result", http.StatusBadRequest)
		return
	}
	if result.ID != s.task.ID {
		http.Error(w, "result ID does not match task", http.StatusConflict)
		return
	}

	select {
	case s.results <- result:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "result already submitted", http.StatusConflict)
	}
}

func (s *Server) authenticate(candidate string) bool {
	if len(candidate) != len(s.token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(s.token)) == 1
}

func (s *Server) setBridgeHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Cache-Control", "no-store")
}

func randomHex(byteCount int) (string, error) {
	data := make([]byte, byteCount)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}
