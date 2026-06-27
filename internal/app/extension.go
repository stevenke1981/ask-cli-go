package app

import (
	"fmt"
	"os"
	"path/filepath"
)

// ExtensionDir locates the unpacked ChatGPT bridge extension.
func ExtensionDir() (string, error) {
	if configured := os.Getenv("ASK_CLI_EXTENSION_DIR"); configured != "" {
		return findExtensionDir([]string{configured})
	}

	candidates := make([]string, 0, 4)
	if executable, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executable)
		candidates = append(
			candidates,
			filepath.Join(executableDir, "extension", "chatgpt-bridge"),
			filepath.Join(executableDir, "..", "extension", "chatgpt-bridge"),
		)
	}
	if workingDir, err := os.Getwd(); err == nil {
		candidates = append(
			candidates,
			filepath.Join(workingDir, "extension", "chatgpt-bridge"),
		)
	}
	return findExtensionDir(candidates)
}

func findExtensionDir(candidates []string) (string, error) {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		info, err := os.Stat(filepath.Join(absolute, "manifest.json"))
		if err == nil && !info.IsDir() {
			return filepath.Clean(absolute), nil
		}
	}
	return "", fmt.Errorf(
		"ChatGPT bridge extension not found; set ASK_CLI_EXTENSION_DIR to the folder containing manifest.json",
	)
}
