package profile

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

// DefaultChromeDataDir returns the default Chrome user data directory
// for the current platform.
func DefaultChromeDataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("LOCALAPPDATA")
		if appData == "" {
			return "", fmt.Errorf("LOCALAPPDATA environment variable not set")
		}
		return filepath.Join(appData, "Google", "Chrome", "User Data"), nil
	case "darwin":
		return defaultHomeDir("Library", "Application Support", "Google", "Chrome")
	case "linux":
		return defaultHomeDir(".config", "google-chrome")
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// DefaultProfileDir returns the default Chrome profile directory
// (usually "Default" under the user data directory).
func DefaultProfileDir() (string, error) {
	dataDir, err := DefaultChromeDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "Default"), nil
}

// ProfileExists reports whether the given profile directory exists.
func ProfileExists(profileDir string) bool {
	info, err := os.Stat(profileDir)
	return err == nil && info.IsDir()
}

// CookiesDBPath returns the expected path to the Cookies SQLite database
// for a given profile directory.
func CookiesDBPath(profileDir string) string {
	return filepath.Join(profileDir, "Network", "Cookies")
}

// LocalStatePath returns the expected path to the Local State file
// given a Chrome user data directory.
func LocalStatePath(chromeDataDir string) string {
	return filepath.Join(chromeDataDir, "Local State")
}

func defaultHomeDir(elem ...string) (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		u, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		home = u.HomeDir
	}
	parts := append([]string{home}, elem...)
	return filepath.Join(parts...), nil
}
