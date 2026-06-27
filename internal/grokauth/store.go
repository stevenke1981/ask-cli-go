package grokauth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Tokens holds Grok OAuth token data.
type Tokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Email        string    `json:"email,omitempty"`
	AuthMode     string    `json:"auth_mode,omitempty"`
}

// Expired reports whether the access token is expired, with an optional skew.
func (t Tokens) Expired(skew time.Duration) bool {
	if t.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().Add(skew).After(t.ExpiresAt)
}

// ValidAccessToken returns the token string if it exists and is not expired.
func (t Tokens) ValidAccessToken(skew time.Duration) (string, bool) {
	tok := strings.TrimSpace(t.AccessToken)
	if tok == "" {
		return "", false
	}
	if t.Expired(skew) {
		return "", false
	}
	return tok, true
}

// OAuthPath returns the path to the Grok OAuth tokens file.
func OAuthPath(dataDir string) string {
	return filepath.Join(dataDir, "grok_oauth.json")
}

// LoadOAuthTokens reads persisted Grok OAuth tokens.
// Returns empty Tokens (no error) if the file does not exist.
func LoadOAuthTokens(dataDir string) (Tokens, error) {
	path := OAuthPath(dataDir)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Tokens{}, nil
		}
		return Tokens{}, fmt.Errorf("read grok oauth: %w", err)
	}
	var tok Tokens
	if err := json.Unmarshal(b, &tok); err != nil {
		return Tokens{}, fmt.Errorf("parse grok oauth: %w", err)
	}
	return tok, nil
}

// SaveOAuthTokens persists Grok OAuth tokens to disk.
func SaveOAuthTokens(dataDir string, tok Tokens) error {
	b, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("marshal grok oauth: %w", err)
	}
	path := OAuthPath(dataDir)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create grok auth dir: %w", err)
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		return fmt.Errorf("write grok oauth: %w", err)
	}
	return nil
}
