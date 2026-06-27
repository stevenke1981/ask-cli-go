package platform

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"slices"
)

// FindChromePath attempts to locate a regular Google Chrome installation.
func FindChromePath() (string, error) {
	candidates := chromeCandidates()
	for _, p := range candidates {
		if pathExists(p) {
			return p, nil
		}
	}
	return "", fmt.Errorf("Google Chrome not found")
}

func chromeCandidates() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			os.Getenv("LOCALAPPDATA") + `\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		}
	default: // linux
		paths, _ := exec.LookPath("google-chrome")
		if paths != "" {
			return []string{paths}
		}
		return []string{"/usr/bin/google-chrome", "/usr/bin/google-chrome-stable"}
	}
}

func pathExists(p string) bool {
	if p == "" {
		return false
	}
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// OpenChromeURLs opens URLs in the user's regular Google Chrome process. It
// deliberately adds no automation, remote-debugging, headless, or profile
// flags, so an existing normal Chrome profile and process are reused.
func OpenChromeURLs(ctx context.Context, urls []string) error {
	args := chromeCommandArgs(urls)
	if len(args) == 0 {
		return fmt.Errorf("no valid HTTPS URLs to open")
	}

	chromePath, err := FindChromePath()
	if err != nil {
		return err
	}
	if err := exec.CommandContext(ctx, chromePath, args...).Start(); err != nil {
		return fmt.Errorf("open Google Chrome: %w", err)
	}
	return nil
}

// OpenChromeExtensions opens Chrome's extension management page in regular
// Google Chrome. No automation or profile flags are added.
func OpenChromeExtensions(ctx context.Context) error {
	args := chromeInternalPageArgs("chrome://extensions")
	if len(args) == 0 {
		return fmt.Errorf("invalid Chrome internal page")
	}
	chromePath, err := FindChromePath()
	if err != nil {
		return err
	}
	if err := exec.CommandContext(ctx, chromePath, args...).Start(); err != nil {
		return fmt.Errorf("open Chrome extensions page: %w", err)
	}
	return nil
}

// OpenFolder opens a local directory in the operating system's file manager.
func OpenFolder(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("open extension folder: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("extension path is not a directory: %s", path)
	}

	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("explorer.exe", path)
	case "darwin":
		command = exec.Command("open", path)
	default:
		command = exec.Command("xdg-open", path)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("open extension folder: %w", err)
	}
	return nil
}

func chromeCommandArgs(urls []string) []string {
	if len(urls) == 0 {
		return nil
	}
	for _, rawURL := range urls {
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
			return nil
		}
	}
	return slices.Clone(urls)
}

func chromeInternalPageArgs(page string) []string {
	if page != "chrome://extensions" {
		return nil
	}
	return []string{page}
}
