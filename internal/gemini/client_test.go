package gemini

import (
	"testing"
	"time"
)

func TestPKCEChallenge(t *testing.T) {
	verifier, challenge, err := PKCEChallenge()
	if err != nil {
		t.Fatalf("PKCEChallenge() error = %v", err)
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

func TestNewClientDefaults(t *testing.T) {
	// Set env so ClientIDFromEnv has something to return
	t.Setenv("GEMINI_CLIENT_ID", "test-client-id")
	t.Setenv("GEMINI_CLIENT_SECRET", "test-client-secret")

	c := NewClient()
	if c == nil {
		t.Fatal("NewClient() returned nil")
	}
	if c.IsAuthenticated() {
		t.Fatal("new client should not be authenticated")
	}
	if c.clientID != "test-client-id" {
		t.Fatalf("clientID = %q, want %q", c.clientID, "test-client-id")
	}
}

func TestAuthURL(t *testing.T) {
	t.Setenv("GEMINI_CLIENT_ID", "test-client-id.apps.googleusercontent.com")
	c := NewClient()
	authURL, _, err := c.AuthURL()
	if err != nil {
		t.Fatalf("AuthURL() error = %v", err)
	}
	if authURL == "" {
		t.Fatal("auth URL is empty")
	}
	if !contains(authURL, "accounts.google.com") {
		t.Fatalf("auth URL = %q, should contain accounts.google.com", authURL)
	}
	if !contains(authURL, "code_challenge=") {
		t.Fatalf("auth URL = %q, should contain code_challenge", authURL)
	}
	if !contains(authURL, "S256") {
		t.Fatalf("auth URL = %q, should contain S256", authURL)
	}
}

func TestSetAndGetTokens(t *testing.T) {
	c := NewClient()
	if c.IsAuthenticated() {
		t.Fatal("should not be authenticated initially")
	}

	expires := time.Now().Add(1 * time.Hour)
	c.SetTokens("test-access", "test-refresh", expires)

	if !c.IsAuthenticated() {
		t.Fatal("should be authenticated after SetTokens")
	}
	if !c.HasRefreshToken() {
		t.Fatal("should have refresh token")
	}
	if c.AccessToken() != "test-access" {
		t.Fatalf("AccessToken() = %q", c.AccessToken())
	}
}

func TestIsAccessTokenExpired(t *testing.T) {
	c := NewClient()

	// Not expired
	expires := time.Now().Add(1 * time.Hour)
	c.SetTokens("access", "refresh", expires)
	if c.IsAccessTokenExpired() {
		t.Fatal("token should not be expired (1 hour from now)")
	}

	// Expired
	expires = time.Now().Add(-1 * time.Hour)
	c.SetTokens("access", "refresh", expires)
	if !c.IsAccessTokenExpired() {
		t.Fatal("token should be expired (1 hour ago)")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
