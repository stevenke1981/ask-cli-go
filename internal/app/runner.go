package app

import (
	"context"
	"fmt"
	"time"
)

// Runner orchestrates the full lifecycle of sending a prompt to ChatGPT.
type Runner struct {
	Config *Config
}

// NewRunner creates a new Runner with the given configuration.
func NewRunner(cfg *Config) *Runner {
	return &Runner{Config: cfg}
}

// Result holds the output of a ChatGPT interaction.
type Result struct {
	ResponseMarkdown string
	DumpHTML         string
	ScreenshotData   []byte
	DownloadedImages []string
	Warning          string
}

// RunPrompt sends a prompt to ChatGPT and returns the result.
func (r *Runner) RunPrompt(ctx context.Context, prompt string) (*Result, error) {
	// TODO: Implement browser-based prompt flow
	if r.Config.Verbose {
		fmt.Printf("[verbose] Using backend: %s\n", r.Config.Backend)
		fmt.Printf("[verbose] Profile: %s\n", r.Config.ResolveProfileDir())
	}
	return nil, fmt.Errorf("not yet implemented")
}

// Login opens a browser for manual login.
func (r *Runner) Login(ctx context.Context) error {
	if r.Config.Verbose {
		fmt.Println("[verbose] Opening Chrome for manual login...")
	}
	// TODO: Implement browser login flow
	return fmt.Errorf("not yet implemented")
}

// GetLatestResponse retrieves the latest assistant response.
func (r *Runner) GetLatestResponse(ctx context.Context) (*Result, error) {
	// TODO: Implement
	return nil, fmt.Errorf("not yet implemented")
}

// DumpHTML retrieves the current page HTML.
func (r *Runner) DumpHTML(ctx context.Context) (string, error) {
	// TODO: Implement
	return "", fmt.Errorf("not yet implemented")
}

// Screenshot captures a screenshot of the current page.
func (r *Runner) Screenshot(ctx context.Context) ([]byte, error) {
	// TODO: Implement
	return nil, fmt.Errorf("not yet implemented")
}

// NewChat creates a new ChatGPT conversation.
func (r *Runner) NewChat(ctx context.Context) error {
	if r.Config.Verbose {
		fmt.Println("[verbose] Creating new ChatGPT conversation...")
	}
	// TODO: Implement
	return fmt.Errorf("not yet implemented")
}

// ResolveTimeout returns the configured timeout duration.
func (r *Runner) ResolveTimeout() time.Duration {
	if r.Config.TimeoutSec > 0 {
		return time.Duration(r.Config.TimeoutSec) * time.Second
	}
	return 180 * time.Second
}
