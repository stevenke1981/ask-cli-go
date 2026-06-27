package app

import (
	"os"
	"path/filepath"
	"runtime"
)

// DefaultBaseDir returns the base directory for ask-cli data (~/.ask-cli).
// Defined as a variable so tests can override it.
var DefaultBaseDir = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".ask-cli")
	}
	return filepath.Join(home, ".ask-cli")
}

// DefaultProfileDir returns the default Chrome profile directory.
func DefaultProfileDir() string {
	return filepath.Join(DefaultBaseDir(), "chrome-profile")
}

// DefaultDownloadsDir returns the default downloads directory.
func DefaultDownloadsDir() string {
	return filepath.Join(DefaultBaseDir(), "downloads")
}

// DefaultScreenshotsDir returns the default screenshots directory.
func DefaultScreenshotsDir() string {
	return filepath.Join(DefaultBaseDir(), "screenshots")
}

// DefaultDumpsDir returns the default HTML dumps directory.
func DefaultDumpsDir() string {
	return filepath.Join(DefaultBaseDir(), "dumps")
}

// EnsureDirs creates all required data directories.
func EnsureDirs() error {
	dirs := []string{
		DefaultBaseDir(),
		DefaultProfileDir(),
		DefaultDownloadsDir(),
		DefaultScreenshotsDir(),
		DefaultDumpsDir(),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

// NormalizePath converts the path to use the platform's native separators.
func NormalizePath(p string) string {
	return filepath.Clean(p)
}

// IsWindows returns true if running on Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMacOS returns true if running on macOS.
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// IsLinux returns true if running on Linux.
func IsLinux() bool {
	return runtime.GOOS == "linux"
}
