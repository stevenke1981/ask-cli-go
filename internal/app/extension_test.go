package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtensionDirUsesEnvironmentOverride(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "chatgpt-bridge")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASK_CLI_EXTENSION_DIR", dir)

	got, err := ExtensionDir()
	if err != nil {
		t.Fatalf("ExtensionDir() error = %v", err)
	}
	if got != dir {
		t.Fatalf("ExtensionDir() = %q, want %q", got, dir)
	}
}

func TestFindExtensionDirRequiresManifest(t *testing.T) {
	empty := t.TempDir()
	valid := filepath.Join(t.TempDir(), "chatgpt-bridge")
	if err := os.MkdirAll(valid, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(valid, "manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := findExtensionDir([]string{empty, valid})
	if err != nil {
		t.Fatalf("findExtensionDir() error = %v", err)
	}
	if got != valid {
		t.Fatalf("findExtensionDir() = %q, want %q", got, valid)
	}
}
