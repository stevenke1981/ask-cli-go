package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ── Gemini token storage ──

// GeminiTokens holds OAuth tokens + GCP project ID for Gemini API access.
type GeminiTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	ProjectID    string    `json:"project_id"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// GeminiTokensPath returns the path to the Gemini tokens file.
func GeminiTokensPath() string {
	return filepath.Join(DefaultBaseDir(), "gemini-tokens.json")
}

// SaveGeminiTokens persists Gemini OAuth tokens to disk.
func SaveGeminiTokens(tokens *GeminiTokens) error {
	dir := DefaultBaseDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create tokens dir: %w", err)
	}

	tokens.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal gemini tokens: %w", err)
	}

	if err := os.WriteFile(GeminiTokensPath(), data, 0600); err != nil {
		return fmt.Errorf("write gemini tokens: %w", err)
	}
	return nil
}

// LoadGeminiTokens reads Gemini OAuth tokens from disk.
func LoadGeminiTokens() (*GeminiTokens, error) {
	data, err := os.ReadFile(GeminiTokensPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read gemini tokens: %w", err)
	}

	var tokens GeminiTokens
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("parse gemini tokens: %w", err)
	}

	if tokens.AccessToken == "" {
		return nil, nil
	}
	return &tokens, nil
}

// DeleteGeminiTokens removes the Gemini tokens file.
func DeleteGeminiTokens() error {
	if err := os.Remove(GeminiTokensPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete gemini tokens: %w", err)
	}
	return nil
}

// ── Grok session storage ──

// GrokSession holds the API key for Grok/xAI access.
type GrokSession struct {
	APIKey    string    `json:"api_key"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GrokSessionPath returns the path to the Grok session file.
func GrokSessionPath() string {
	return filepath.Join(DefaultBaseDir(), "grok-session.json")
}

// SaveGrokSession persists the Grok API key to disk.
func SaveGrokSession(apiKey string) error {
	dir := DefaultBaseDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create grok dir: %w", err)
	}

	session := GrokSession{
		APIKey:    apiKey,
		UpdatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal grok session: %w", err)
	}

	if err := os.WriteFile(GrokSessionPath(), data, 0600); err != nil {
		return fmt.Errorf("write grok session: %w", err)
	}
	return nil
}

// LoadGrokSession reads the Grok API key from disk.
func LoadGrokSession() (*GrokSession, error) {
	data, err := os.ReadFile(GrokSessionPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read grok session: %w", err)
	}

	var session GrokSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parse grok session: %w", err)
	}

	if session.APIKey == "" {
		return nil, nil
	}
	return &session, nil
}

// DeleteGrokSession removes the Grok session file.
func DeleteGrokSession() error {
	if err := os.Remove(GrokSessionPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete grok session: %w", err)
	}
	return nil
}
