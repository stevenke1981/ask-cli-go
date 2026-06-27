package app

import (
	"os"
	"path/filepath"
)

// Config holds the runtime configuration for the ask CLI.
type Config struct {
	// Backend is the browser automation backend ("rod" or "chromedp").
	Backend string

	// Headless controls whether Chrome runs in headless mode.
	Headless bool

	// NewSession creates a new ChatGPT conversation before sending.
	NewSession bool

	// Verbose enables debug logging.
	Verbose bool

	// OutputFile is the path for writing the Markdown response.
	OutputFile string

	// ImageOutput is the path for downloading generated images.
	ImageOutput string

	// Images is a list of local image paths to attach to the prompt.
	Images []string

	// ProfileDir is the Chrome user data directory.
	ProfileDir string

	// TimeoutSec is the maximum wait time in seconds for a response.
	TimeoutSec int

	// BrowserPath is an optional custom Chrome/Chromium executable path.
	BrowserPath string
}

// DefaultConfig returns a Config with sensible defaults and environment overrides.
func DefaultConfig() *Config {
	cfg := &Config{
		Backend:    "rod",
		Headless:   true,
		NewSession: false,
		Verbose:    false,
		ProfileDir: DefaultProfileDir(),
		TimeoutSec: 180,
	}

	// Environment overrides
	if v := os.Getenv("ASK_CLI_BACKEND"); v != "" {
		cfg.Backend = v
	}
	if v := os.Getenv("ASK_CLI_PROFILE_DIR"); v != "" {
		cfg.ProfileDir = v
	}
	if v := os.Getenv("ASK_CLI_BROWSER_PATH"); v != "" {
		cfg.BrowserPath = v
	}

	return cfg
}

// ResolveProfileDir returns the effective profile directory.
func (c *Config) ResolveProfileDir() string {
	if c.ProfileDir != "" {
		return c.ProfileDir
	}
	return DefaultProfileDir()
}

// ScreenshotPath generates a default screenshot path with timestamp.
func ScreenshotPath() string {
	return filepath.Join(DefaultScreenshotsDir(), "chatgpt-screenshot.png")
}

// DumpPath generates a default dump path with timestamp.
func DumpPath() string {
	return filepath.Join(DefaultDumpsDir(), "chatgpt-dump.html")
}
