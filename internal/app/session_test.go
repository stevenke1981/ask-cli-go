package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadSession(t *testing.T) {
	// Use a temp directory to avoid polluting real config
	tmpDir := t.TempDir()
	origBase := DefaultBaseDir
	DefaultBaseDir = func() string { return tmpDir }
	defer func() { DefaultBaseDir = origBase }()

	s := &SessionData{
		SessionToken:  "test-session-token-xyz",
		AccessToken:   "test-access-token-abc",
		ProfileSource: "C:\\Chrome\\Default",
	}

	if err := SaveSession(s); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, "session.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("session.json was not created")
	}

	loaded, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadSession returned nil")
	}
	if loaded.SessionToken != "test-session-token-xyz" {
		t.Errorf("SessionToken = %q, want %q", loaded.SessionToken, "test-session-token-xyz")
	}
	if loaded.AccessToken != "test-access-token-abc" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "test-access-token-abc")
	}
	if loaded.ProfileSource != "C:\\Chrome\\Default" {
		t.Errorf("ProfileSource = %q, want %q", loaded.ProfileSource, "C:\\Chrome\\Default")
	}
}

func TestLoadSessionNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	origBase := DefaultBaseDir
	DefaultBaseDir = func() string { return tmpDir }
	defer func() { DefaultBaseDir = origBase }()

	loaded, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession on missing file should not error: %v", err)
	}
	if loaded != nil {
		t.Fatal("LoadSession should return nil when no file exists")
	}
}

func TestDeleteSession(t *testing.T) {
	tmpDir := t.TempDir()
	origBase := DefaultBaseDir
	DefaultBaseDir = func() string { return tmpDir }
	defer func() { DefaultBaseDir = origBase }()

	// Create a session first
	if err := SaveSessionToken("tok", "profile"); err != nil {
		t.Fatalf("SaveSessionToken failed: %v", err)
	}

	// Verify exists
	path := filepath.Join(tmpDir, "session.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("session.json was not created")
	}

	// Delete
	if err := DeleteSession(); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("session.json was not deleted")
	}

	// Deleting again should not error
	if err := DeleteSession(); err != nil {
		t.Fatalf("DeleteSession on already-deleted file failed: %v", err)
	}
}

func TestSaveSessionToken(t *testing.T) {
	tmpDir := t.TempDir()
	origBase := DefaultBaseDir
	DefaultBaseDir = func() string { return tmpDir }
	defer func() { DefaultBaseDir = origBase }()

	if err := SaveSessionToken("session-123", "C:\\Chrome\\Profile"); err != nil {
		t.Fatalf("SaveSessionToken failed: %v", err)
	}

	loaded, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession after SaveSessionToken failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadSession returned nil")
	}
	if loaded.SessionToken != "session-123" {
		t.Errorf("SessionToken = %q, want %q", loaded.SessionToken, "session-123")
	}
	if loaded.AccessToken != "" {
		t.Errorf("AccessToken should be empty, got %q", loaded.AccessToken)
	}
}

func TestIsAccessTokenExpired(t *testing.T) {
	tests := []struct {
		name    string
		session *SessionData
		expired bool
	}{
		{
			name:    "empty token",
			session: &SessionData{AccessToken: ""},
			expired: true,
		},
		{
			name:    "no expiry",
			session: &SessionData{AccessToken: "tok"},
			expired: true,
		},
		{
			name: "valid future",
			session: &SessionData{
				AccessToken:        "tok",
				AccessTokenExpires: time.Now().Add(30 * time.Minute).Format(time.RFC3339),
			},
			expired: false,
		},
		{
			name: "expired past",
			session: &SessionData{
				AccessToken:        "tok",
				AccessTokenExpires: time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
			},
			expired: true,
		},
		{
			name: "within grace period (4 min before expiry)",
			session: &SessionData{
				AccessToken:        "tok",
				AccessTokenExpires: time.Now().Add(4 * time.Minute).Format(time.RFC3339),
			},
			expired: true, // 5-min grace means this is considered expired
		},
		{
			name: "just beyond grace period (6 min before expiry)",
			session: &SessionData{
				AccessToken:        "tok",
				AccessTokenExpires: time.Now().Add(6 * time.Minute).Format(time.RFC3339),
			},
			expired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.session.IsAccessTokenExpired()
			if got != tt.expired {
				t.Errorf("IsAccessTokenExpired() = %v, want %v", got, tt.expired)
			}
		})
	}
}
