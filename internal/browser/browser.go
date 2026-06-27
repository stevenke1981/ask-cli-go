// Package browser provides an abstraction over Chrome DevTools Protocol backends.
//
// The Browser interface defines all operations needed to interact with ChatGPT
// through a browser. The default implementation uses rod, with chromedp as an
// optional alternative.
package browser

import (
	"context"
	"errors"
	"time"
)

// Common errors returned by browser operations.
var (
	ErrNotLoggedIn      = errors.New("not logged in to ChatGPT: run 'ask login' first")
	ErrComposerNotFound = errors.New("cannot find the prompt input area")
	ErrTimeout          = errors.New("timed out waiting for response")
	ErrNotSupported     = errors.New("operation not supported by this backend")
	ErrImageUpload      = errors.New("failed to upload image")
)

// BrowserOptions contains configuration for creating a browser instance.
type BrowserOptions struct {
	// Headless controls whether Chrome runs without a visible window.
	Headless bool

	// NewSession creates a new ChatGPT conversation.
	NewSession bool

	// Verbose enables detailed debug logging.
	Verbose bool

	// ProfileDir is the Chrome user data directory for session persistence.
	ProfileDir string

	// Timeout is the maximum time to wait for operations.
	Timeout time.Duration

	// BrowserPath is an optional custom Chrome/Chromium executable path.
	BrowserPath string
}

// Browser defines the interface for browser automation backends.
type Browser interface {
	// Start launches a browser instance and connects to it.
	Start(ctx context.Context, opts BrowserOptions) error

	// Stop closes the browser and cleans up resources.
	Stop(ctx context.Context) error

	// Navigate directs the browser to the specified URL.
	Navigate(ctx context.Context, url string) error

	// NewChat starts a new ChatGPT conversation.
	NewChat(ctx context.Context) error

	// AttachImages uploads one or more image files to the composer.
	AttachImages(ctx context.Context, paths []string) error

	// SendPrompt enters the prompt text into the composer and submits it.
	SendPrompt(ctx context.Context, prompt string) error

	// WaitForResponseDone waits for the assistant to finish generating a response.
	WaitForResponseDone(ctx context.Context) error

	// LatestResponseMarkdown retrieves the latest assistant response as Markdown.
	LatestResponseMarkdown(ctx context.Context) (string, error)

	// DumpHTML returns the full HTML content of the current page.
	DumpHTML(ctx context.Context) (string, error)

	// Screenshot captures a screenshot of the current page.
	Screenshot(ctx context.Context) ([]byte, error)

	// DownloadResponseImages downloads images from the latest assistant response.
	DownloadResponseImages(ctx context.Context, outPath string) ([]string, error)
}

// NewBrowser creates a browser backend based on the backend name.
// Supported: "rod", "chromedp". Defaults to "rod".
func NewBrowser(backend string) Browser {
	switch backend {
	case "chromedp":
		return &ChromedpBrowser{}
	default:
		return &RodBrowser{}
	}
}
