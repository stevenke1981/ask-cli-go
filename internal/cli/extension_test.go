package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtensionCommandContract(t *testing.T) {
	if extensionCmd == nil {
		t.Fatal("extensionCmd = nil")
	}
	if extensionCmd.Name() != "extension" {
		t.Fatalf("extension command name = %q", extensionCmd.Name())
	}
}

func TestRunExtensionSetupOpensChromeAndFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "chatgpt-bridge")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASK_CLI_EXTENSION_DIR", dir)

	originalChrome := openChromeExtensions
	originalFolder := openExtensionFolder
	t.Cleanup(func() {
		openChromeExtensions = originalChrome
		openExtensionFolder = originalFolder
	})
	chromeOpened := false
	folderOpened := ""
	openChromeExtensions = func(context.Context) error {
		chromeOpened = true
		return nil
	}
	openExtensionFolder = func(path string) error {
		folderOpened = path
		return nil
	}

	var output bytes.Buffer
	if err := runExtensionSetup(context.Background(), &output); err != nil {
		t.Fatalf("runExtensionSetup() error = %v", err)
	}
	if !chromeOpened || folderOpened != dir {
		t.Fatalf("chromeOpened=%v folderOpened=%q", chromeOpened, folderOpened)
	}
	if !strings.Contains(output.String(), "Load unpacked") || !strings.Contains(output.String(), dir) {
		t.Fatalf("output = %q", output.String())
	}
}
