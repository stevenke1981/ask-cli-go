package webbridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestExtensionManifestUsesMinimalPermissions(t *testing.T) {
	manifestPath := filepath.Join(
		"..", "..", "extension", "chatgpt-bridge", "manifest.json",
	)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read extension manifest: %v", err)
	}

	var manifest struct {
		ManifestVersion int      `json:"manifest_version"`
		Permissions     []string `json:"permissions"`
		HostPermissions []string `json:"host_permissions"`
		Background      struct {
			ServiceWorker string `json:"service_worker"`
		} `json:"background"`
		ContentScripts []struct {
			Matches []string `json:"matches"`
			JS      []string `json:"js"`
		} `json:"content_scripts"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}

	if manifest.ManifestVersion != 3 {
		t.Fatalf("manifest version = %d, want 3", manifest.ManifestVersion)
	}
	for _, forbidden := range []string{"cookies", "debugger", "scripting"} {
		if slices.Contains(manifest.Permissions, forbidden) {
			t.Fatalf("manifest requests forbidden permission %q", forbidden)
		}
	}
	for _, required := range []string{
		"https://chatgpt.com/*",
		"http://127.0.0.1/*",
	} {
		if !slices.Contains(manifest.HostPermissions, required) {
			t.Fatalf("manifest lacks host permission %q", required)
		}
	}
	if manifest.Background.ServiceWorker != "background.js" {
		t.Fatalf("service worker = %q", manifest.Background.ServiceWorker)
	}
	if len(manifest.ContentScripts) != 1 ||
		!slices.Contains(manifest.ContentScripts[0].Matches, "https://chatgpt.com/*") ||
		!slices.Contains(manifest.ContentScripts[0].JS, "content.js") {
		t.Fatalf("unexpected content scripts: %#v", manifest.ContentScripts)
	}
}
