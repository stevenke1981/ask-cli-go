package profile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestProfileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Non-existent directory
	if ProfileExists(filepath.Join(tmpDir, "nope")) {
		t.Error("ProfileExists should be false for non-existent dir")
	}

	// Existing directory
	subDir := filepath.Join(tmpDir, "Default")
	if err := os.MkdirAll(subDir, 0700); err != nil {
		t.Fatal(err)
	}
	if !ProfileExists(subDir) {
		t.Error("ProfileExists should be true for existing dir")
	}

	// Existing file (not a directory)
	filePath := filepath.Join(tmpDir, "somefile")
	if err := os.WriteFile(filePath, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	if ProfileExists(filePath) {
		t.Error("ProfileExists should be false for a file")
	}
}

func TestCookiesDBPath(t *testing.T) {
	base := filepath.FromSlash("/home/user/.config/google-chrome/Default")
	path := CookiesDBPath(base)
	expected := filepath.Join(base, "Network", "Cookies")
	if path != expected {
		t.Errorf("CookiesDBPath = %q, want %q", path, expected)
	}
}

func TestLocalStatePath(t *testing.T) {
	base := filepath.FromSlash("/home/user/.config/google-chrome")
	path := LocalStatePath(base)
	expected := filepath.Join(base, "Local State")
	if path != expected {
		t.Errorf("LocalStatePath = %q, want %q", path, expected)
	}
}

func TestReadLocalState_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	lsPath := filepath.Join(tmpDir, "Local State")
	content := `{
		"os_crypt": {
			"encrypted_key": "RFBBUEkBAAAA0Iyd3wEV0RGMegDAT8JX6wEAAADQw6JzGLVqQ4GBptFDPJE2AAAAAAIAAAAAABBmAAAAAQAAIAAAAJmCz6uR4C5y+Rr3pKwZ5M7SBLy1sVfOuYOBx2QyV2EjAAAAAA6AAAAAAgAAIAAAABk7GC65P2eJmCvYjG+zrPgAAACgYnPKSHMJ9FO1pceU0RAAAAA1X1YBeigS2XKcnVJk+ROQUAAAAAlB52x7ZqVHbXBNgqEoDkQ0hCmRZXUQSCeRS4Qh3Svh1GplFvkHxYW04hFIzIQI9Wl2EsJbODIfqek/UAPItUF"
		}
	}`
	if err := os.WriteFile(lsPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	ls, err := ReadLocalState(lsPath)
	if err != nil {
		t.Fatalf("ReadLocalState failed: %v", err)
	}
	if ls.OSCrypt.EncryptedKey == "" {
		t.Fatal("EncryptedKey should not be empty")
	}
	if len(ls.OSCrypt.EncryptedKey) < 50 {
		t.Errorf("EncryptedKey too short: %d chars", len(ls.OSCrypt.EncryptedKey))
	}
}

func TestReadLocalState_NoFile(t *testing.T) {
	_, err := ReadLocalState("/nonexistent/Local State")
	if err == nil {
		t.Fatal("ReadLocalState should error on missing file")
	}
}

func TestReadLocalState_EmptyKey(t *testing.T) {
	tmpDir := t.TempDir()
	lsPath := filepath.Join(tmpDir, "Local State")
	content := `{"os_crypt": {"encrypted_key": ""}}`
	if err := os.WriteFile(lsPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := ReadLocalState(lsPath)
	if err == nil {
		t.Fatal("ReadLocalState should error on empty encrypted_key")
	}
	if !errors.Is(err, ErrNoMasterKey) {
		t.Errorf("error = %v, should wrap %v", err, ErrNoMasterKey)
	}
}

func TestReadLocalState_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	lsPath := filepath.Join(tmpDir, "Local State")
	content := `{invalid json}`
	if err := os.WriteFile(lsPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := ReadLocalState(lsPath)
	if err == nil {
		t.Fatal("ReadLocalState should error on invalid JSON")
	}
}

func TestReadLocalState_MissingField(t *testing.T) {
	tmpDir := t.TempDir()
	lsPath := filepath.Join(tmpDir, "Local State")
	content := `{"other_field": 123}`
	if err := os.WriteFile(lsPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := ReadLocalState(lsPath)
	if err == nil {
		t.Fatal("ReadLocalState should error on missing os_crypt.encrypted_key")
	}
}

func TestCookieDecryptFallback_PlainText(t *testing.T) {
	// For unencrypted cookies, decryptCookieValueOrFallback should
	// return the value as plaintext if it looks like printable text.
	// Since masterKey is empty, AES-GCM will fail and fallback kicks in.
	result := decryptCookieValueOrFallback(nil, []byte("hello"))
	if result != "hello" {
		t.Errorf("fallback = %q, want %q", result, "hello")
	}
}

func TestCookieDecryptFallback_Empty(t *testing.T) {
	result := decryptCookieValueOrFallback(nil, nil)
	if result != "" {
		t.Errorf("fallback = %q, want empty", result)
	}
	result = decryptCookieValueOrFallback(nil, []byte{})
	if result != "" {
		t.Errorf("fallback = %q, want empty", result)
	}
}

func TestCookieDecryptFallback_Binary(t *testing.T) {
	// Binary data should NOT be treated as plaintext
	result := decryptCookieValueOrFallback(nil, []byte{0x00, 0x01, 0x02, 0xFF})
	if result != "" {
		t.Errorf("fallback should be empty for binary data, got %q", result)
	}
}

func TestIsLikelyPlainText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty", "", false},
		{"ascii", "hello world", true},
		{"with_tab", "hello\tworld", true},
		{"with_newline", "hello\nworld", true},
		{"numbers", "abc123!@#", true},
		{"null_byte", "hello\x00world", false},
		{"high_bytes", "hello\x80world", false},
		{"unicode_non_ascii", "héllo", false}, // é = U+00E9 is non-ASCII
		{"mixed_unicode", "hello café", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLikelyPlainText(tt.input)
			if got != tt.want {
				t.Errorf("isLikelyPlainText(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
