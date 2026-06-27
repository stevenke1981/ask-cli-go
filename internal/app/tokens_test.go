package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadGeminiTokens(t *testing.T) {
	dir := t.TempDir()
	original := DefaultBaseDir
	DefaultBaseDir = func() string { return dir }
	t.Cleanup(func() { DefaultBaseDir = original })

	tokens := &GeminiTokens{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	if err := SaveGeminiTokens(tokens); err != nil {
		t.Fatalf("SaveGeminiTokens() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(GeminiTokensPath()); err != nil {
		t.Fatalf("tokens file not created: %v", err)
	}

	loaded, err := LoadGeminiTokens()
	if err != nil {
		t.Fatalf("LoadGeminiTokens() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadGeminiTokens() returned nil")
	}
	if loaded.AccessToken != "test-access-token" {
		t.Fatalf("access token = %q", loaded.AccessToken)
	}
	if loaded.RefreshToken != "test-refresh-token" {
		t.Fatalf("refresh token = %q", loaded.RefreshToken)
	}
}

func TestLoadGeminiTokensReturnsNilForMissingFile(t *testing.T) {
	dir := t.TempDir()
	DefaultBaseDir = func() string { return dir }

	tokens, err := LoadGeminiTokens()
	if err != nil {
		t.Fatalf("LoadGeminiTokens() error = %v", err)
	}
	if tokens != nil {
		t.Fatal("expected nil for missing file")
	}
}

func TestDeleteGeminiTokens(t *testing.T) {
	dir := t.TempDir()
	DefaultBaseDir = func() string { return dir }

	// Save then delete
	tokens := &GeminiTokens{AccessToken: "test"}
	if err := SaveGeminiTokens(tokens); err != nil {
		t.Fatal(err)
	}

	if err := DeleteGeminiTokens(); err != nil {
		t.Fatalf("DeleteGeminiTokens() error = %v", err)
	}

	// Verify deleted
	if _, err := os.Stat(GeminiTokensPath()); !os.IsNotExist(err) {
		t.Fatal("tokens file should be deleted")
	}

	// Calling again should not error
	if err := DeleteGeminiTokens(); err != nil {
		t.Fatalf("DeleteGeminiTokens() on missing file error = %v", err)
	}
}

func TestSaveAndLoadGrokSession(t *testing.T) {
	dir := t.TempDir()
	original := DefaultBaseDir
	DefaultBaseDir = func() string { return dir }
	t.Cleanup(func() { DefaultBaseDir = original })

	if err := SaveGrokSession("test-api-key"); err != nil {
		t.Fatalf("SaveGrokSession() error = %v", err)
	}

	session, err := LoadGrokSession()
	if err != nil {
		t.Fatalf("LoadGrokSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("LoadGrokSession() returned nil")
	}
	if session.APIKey != "test-api-key" {
		t.Fatalf("API key = %q", session.APIKey)
	}
}

func TestLoadGrokSessionReturnsNilForMissingFile(t *testing.T) {
	dir := t.TempDir()
	DefaultBaseDir = func() string { return dir }

	session, err := LoadGrokSession()
	if err != nil {
		t.Fatalf("LoadGrokSession() error = %v", err)
	}
	if session != nil {
		t.Fatal("expected nil for missing file")
	}
}

func TestGeminiTokensPaths(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".ask-cli")
	DefaultBaseDir = func() string { return dir }

	path := GeminiTokensPath()
	if path == "" {
		t.Fatal("GeminiTokensPath() returned empty")
	}
	if filepath.Base(path) != "gemini-tokens.json" {
		t.Fatalf("filename = %q, want gemini-tokens.json", filepath.Base(path))
	}
}

func TestGrokSessionPaths(t *testing.T) {
	dir := t.TempDir()
	DefaultBaseDir = func() string { return dir }

	path := GrokSessionPath()
	if path == "" {
		t.Fatal("GrokSessionPath() returned empty")
	}
	if filepath.Base(path) != "grok-session.json" {
		t.Fatalf("filename = %q, want grok-session.json", filepath.Base(path))
	}
}
