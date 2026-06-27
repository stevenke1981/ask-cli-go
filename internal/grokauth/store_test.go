package grokauth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadTokens(t *testing.T) {
	dir := t.TempDir()

	tokens := Tokens{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		Email:        "test@example.com",
		AuthMode:     "oauth",
	}

	if err := SaveOAuthTokens(dir, tokens); err != nil {
		t.Fatalf("SaveOAuthTokens() error = %v", err)
	}

	loaded, err := LoadOAuthTokens(dir)
	if err != nil {
		t.Fatalf("LoadOAuthTokens() error = %v", err)
	}

	if loaded.AccessToken != "test-access-token" {
		t.Fatalf("AccessToken = %q, want %q", loaded.AccessToken, "test-access-token")
	}
	if loaded.RefreshToken != "test-refresh-token" {
		t.Fatalf("RefreshToken = %q", loaded.RefreshToken)
	}
	if loaded.AuthMode != "oauth" {
		t.Fatalf("AuthMode = %q", loaded.AuthMode)
	}
}

func TestLoadTokensFileNotExist(t *testing.T) {
	dir := t.TempDir()
	tokens, err := LoadOAuthTokens(dir)
	if err != nil {
		t.Fatalf("LoadOAuthTokens() for missing file error = %v", err)
	}
	if tokens.AccessToken != "" {
		t.Fatal("expected empty tokens for missing file")
	}
}

func TestTokenExpired(t *testing.T) {
	tests := []struct {
		name   string
		tokens Tokens
		want   bool
	}{
		{"zero time", Tokens{}, false},
		{"not expired", Tokens{ExpiresAt: time.Now().Add(1 * time.Hour)}, false},
		{"expired", Tokens{ExpiresAt: time.Now().Add(-1 * time.Hour)}, true},
		{"just expired", Tokens{ExpiresAt: time.Now().Add(-1 * time.Minute)}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tokens.Expired(0); got != tt.want {
				t.Errorf("Expired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidAccessToken(t *testing.T) {
	tokens := Tokens{
		AccessToken: "valid-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	tok, valid := tokens.ValidAccessToken(0)
	if !valid {
		t.Fatal("expected valid token")
	}
	if tok != "valid-token" {
		t.Fatalf("token = %q", tok)
	}

	// Expired token
	tokens.ExpiresAt = time.Now().Add(-1 * time.Hour)
	_, valid = tokens.ValidAccessToken(0)
	if valid {
		t.Fatal("expected expired token to be invalid")
	}

	// Empty token
	tokens = Tokens{}
	_, valid = tokens.ValidAccessToken(0)
	if valid {
		t.Fatal("expected empty token to be invalid")
	}
}

func TestOAuthPath(t *testing.T) {
	dir := filepath.Join("test", "data")
	path := OAuthPath(dir)
	if path != filepath.Join("test", "data", "grok_oauth.json") {
		t.Fatalf("OAuthPath = %q", path)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	tokens := Tokens{AccessToken: "test"}

	if err := SaveOAuthTokens(dir, tokens); err != nil {
		t.Fatalf("SaveOAuthTokens() to nested dir error = %v", err)
	}

	if _, err := os.Stat(OAuthPath(dir)); os.IsNotExist(err) {
		t.Fatal("file was not created in nested directory")
	}
}

func TestPKCEPair(t *testing.T) {
	verifier, challenge, err := pkcePair()
	if err != nil {
		t.Fatalf("pkcePair() error = %v", err)
	}
	if len(verifier) == 0 {
		t.Fatal("verifier is empty")
	}
	if len(challenge) == 0 {
		t.Fatal("challenge is empty")
	}
	if verifier == challenge {
		t.Fatal("verifier should not equal challenge")
	}
}

func TestCLIProxyHeaders(t *testing.T) {
	h := CLIProxyHeaders("grok-build-0.1")
	if h == nil {
		t.Fatal("CLIProxyHeaders() returned nil")
	}
	if h["X-XAI-Token-Auth"] != CLITokenAuthHeader {
		t.Fatalf("X-XAI-Token-Auth = %q", h["X-XAI-Token-Auth"])
	}
	if h["x-grok-client-identifier"] != CLIClientIdentifier {
		t.Fatalf("x-grok-client-identifier = %q", h["x-grok-client-identifier"])
	}
	if h["x-grok-model-override"] != "grok-build-0.1" {
		t.Fatalf("x-grok-model-override = %q", h["x-grok-model-override"])
	}
}
