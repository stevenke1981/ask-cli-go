package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SessionData holds cached authentication tokens.
type SessionData struct {
	// SessionToken is the __Secure-next-auth.session-token cookie value.
	SessionToken string `json:"session_token"`

	// AccessToken is the Bearer token from /api/auth/session.
	AccessToken string `json:"access_token,omitempty"`

	// AccessTokenExpires is when the access token expires (RFC3339).
	AccessTokenExpires string `json:"access_token_expires,omitempty"`

	// ProfileSource is the Chrome profile directory the token came from.
	ProfileSource string `json:"profile_source,omitempty"`

	// UpdatedAt is when this cache was last updated.
	UpdatedAt string `json:"updated_at"`
}

// SessionTokenPath returns the path to the cached session file.
func SessionTokenPath() string {
	return filepath.Join(DefaultBaseDir(), "session.json")
}

// SaveSession persists session data to disk.
func SaveSession(s *SessionData) error {
	dir := DefaultBaseDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}

	s.UpdatedAt = time.Now().Format(time.RFC3339)

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	if err := os.WriteFile(SessionTokenPath(), data, 0600); err != nil {
		return fmt.Errorf("write session file: %w", err)
	}
	return nil
}

// LoadSession reads session data from disk.
func LoadSession() (*SessionData, error) {
	data, err := os.ReadFile(SessionTokenPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no cached session
		}
		return nil, fmt.Errorf("read session file: %w", err)
	}

	var s SessionData
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse session file: %w", err)
	}

	if s.SessionToken == "" {
		return nil, nil
	}

	return &s, nil
}

// DeleteSession removes the cached session file.
func DeleteSession() error {
	if err := os.Remove(SessionTokenPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete session file: %w", err)
	}
	return nil
}

// SaveSessionToken is a convenience function that saves just the session token.
func SaveSessionToken(token, profileSource string) error {
	s := &SessionData{
		SessionToken:  token,
		ProfileSource: profileSource,
		UpdatedAt:     time.Now().Format(time.RFC3339),
	}
	return SaveSession(s)
}

// IsAccessTokenExpired returns true if the cached access token is missing or expired.
func (s *SessionData) IsAccessTokenExpired() bool {
	if s.AccessToken == "" {
		return true
	}
	if s.AccessTokenExpires == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, s.AccessTokenExpires)
	if err != nil {
		return true
	}
	return time.Now().After(t.Add(-5 * time.Minute)) // 5 min grace period
}
