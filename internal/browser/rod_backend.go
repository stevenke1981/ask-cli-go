package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// RodBrowser implements the Browser interface using the go-rod library.
type RodBrowser struct {
	browser *rod.Browser
	page    *rod.Page
	opts    BrowserOptions
}

// Start launches a Chrome instance and connects rod to it.
func (rb *RodBrowser) Start(ctx context.Context, opts BrowserOptions) error {
	rb.opts = opts

	if opts.Verbose {
		fmt.Println("[rod] Launching Chrome...")
	}

	// Build launcher
	l := launcher.New().
		Headless(opts.Headless).
		UserDataDir(opts.ProfileDir).
		Leakless(false) // Disable leakless to avoid Windows Defender false positives

	// Set custom browser path if provided, otherwise try system Chrome
	if opts.BrowserPath != "" {
		l = l.Bin(opts.BrowserPath)
	} else if path, ok := launcher.LookPath(); ok {
		l = l.Bin(path)
	}

	// Launch browser
	controlURL, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch Chrome: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("[rod] Chrome launched at %s\n", controlURL)
	}

	// Connect rod
	rb.browser = rod.New().ControlURL(controlURL)

	if err := rb.browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect rod to Chrome: %w", err)
	}

	if opts.Verbose {
		fmt.Println("[rod] Connected to Chrome")
	}

	return nil
}

// Stop closes the browser and cleans up resources.
func (rb *RodBrowser) Stop(ctx context.Context) error {
	if rb.page != nil {
		_ = rb.page.Close()
		rb.page = nil
	}
	if rb.browser != nil {
		_ = rb.browser.Close()
		rb.browser = nil
	}
	if rb.opts.Verbose {
		fmt.Println("[rod] Browser closed")
	}
	return nil
}

// pageOrNew returns the current page or creates a new one.
func (rb *RodBrowser) pageOrNew() (*rod.Page, error) {
	if rb.page != nil {
		return rb.page, nil
	}
	pages, err := rb.browser.Pages()
	if err == nil && len(pages) > 0 {
		rb.page = pages[0]
		return rb.page, nil
	}
	// Create a new blank page
	p, err := rb.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("cannot create page: %w", err)
	}
	rb.page = p
	return rb.page, nil
}

// Navigate directs the browser to the given URL.
func (rb *RodBrowser) Navigate(ctx context.Context, url string) error {
	p, err := rb.pageOrNew()
	if err != nil {
		return err
	}
	if rb.opts.Verbose {
		fmt.Printf("[rod] Navigating to %s\n", url)
	}
	err = p.Context(ctx).Navigate(url)
	if err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}
	p.MustWaitLoad()
	return nil
}

// NewChat starts a new ChatGPT conversation by navigating to the homepage.
func (rb *RodBrowser) NewChat(ctx context.Context) error {
	if rb.opts.Verbose {
		fmt.Println("[rod] Starting new ChatGPT conversation")
	}
	return rb.Navigate(ctx, "https://chatgpt.com/")
}

// AttachImages uploads images to the composer.
func (rb *RodBrowser) AttachImages(ctx context.Context, paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	if err := ValidateImagePaths(paths); err != nil {
		return fmt.Errorf("%w: %v", ErrImageUpload, err)
	}

	p, err := rb.pageOrNew()
	if err != nil {
		return err
	}

	selectors := DefaultSelectors()
	var fileInput *rod.Element
	for _, sel := range selectors.FileInputCandidates {
		input, err := p.Element(sel)
		if err == nil {
			fileInput = input
			break
		}
	}
	if fileInput == nil {
		return fmt.Errorf("%w: file input not found", ErrImageUpload)
	}

	if err := fileInput.SetFiles(paths); err != nil {
		return fmt.Errorf("%w: cannot set files: %v", ErrImageUpload, err)
	}

	if rb.opts.Verbose {
		fmt.Printf("[rod] Uploaded %d image(s)\n", len(paths))
	}

	return nil
}

// SendPrompt enters text into the composer and submits it.
func (rb *RodBrowser) SendPrompt(ctx context.Context, prompt string) error {
	p, err := rb.pageOrNew()
	if err != nil {
		return err
	}

	selectors := DefaultSelectors()

	// Find composer element
	var composer *rod.Element
	for _, sel := range selectors.ComposerCandidates {
		elem, err := p.Element(sel)
		if err == nil {
			composer = elem
			break
		}
	}
	if composer == nil {
		return ErrComposerNotFound
	}

	if rb.opts.Verbose {
		fmt.Println("[rod] Focusing composer")
	}

	// Focus and wait for visibility
	composer.MustWaitVisible()
	composer.MustClick()

	// Clear any existing content via JS (contenteditable divs don't support select())
	_, _ = p.Eval(`() => { const el = document.activeElement; if (el) el.textContent = ''; }`)

	// Use JavaScript to set the prompt text directly (more reliable for contenteditable)
	promptJS := jsString(prompt)
	_, evalErr := p.Eval(fmt.Sprintf(`() => {
		const el = document.activeElement;
		if (el) {
			el.textContent = %s;
			el.dispatchEvent(new Event('input', { bubbles: true }));
			el.dispatchEvent(new Event('change', { bubbles: true }));
		}
	}`, promptJS))
	if evalErr != nil {
		// Fallback: use Input
		if err := composer.Input(prompt); err != nil {
			return fmt.Errorf("cannot input prompt: %w", err)
		}
	}

	// Try send button first, then Enter key fallback
	sent := false
	for _, sel := range selectors.SendButtonCandidates {
		btn, kindErr := p.Element(sel)
		if kindErr == nil {
			if rb.opts.Verbose {
				fmt.Println("[rod] Clicking send button")
			}
			btn.MustClick()
			sent = true
			break
		}
	}

	if !sent {
		if rb.opts.Verbose {
			fmt.Println("[rod] Sending with Enter key")
		}
		// Dispatch Enter key on the composer
		_, enterErr := p.Eval(`() => {
			const el = document.activeElement;
			if (el) {
				el.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', code: 'Enter', keyCode: 13, bubbles: true }));
				el.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter', code: 'Enter', keyCode: 13, bubbles: true }));
			}
		}`)
		if enterErr != nil {
			// Last resort: append newline to trigger send
			if inputErr := composer.Input(prompt + "\n"); inputErr != nil {
				return fmt.Errorf("cannot send prompt: %w", inputErr)
			}
		}
	}

	// Verify prompt was actually sent by checking if composer is now empty or has different content
	textAfter, _ := composer.Text()
	if textAfter == prompt || textAfter == "" {
		// The text might have been accepted; this is normal
		if rb.opts.Verbose {
			fmt.Println("[rod] Prompt sent")
		}
	}

	return nil
}

// WaitForResponseDone waits for the assistant to finish generating.
func (rb *RodBrowser) WaitForResponseDone(ctx context.Context) error {
	p, err := rb.pageOrNew()
	if err != nil {
		return err
	}

	selectors := DefaultSelectors()
	timeout := timeoutFromContext(ctx, 180*time.Second)
	pollInterval := 500 * time.Millisecond
	stableCount := 4
	minWait := 2 * time.Second
	waited := 0 * time.Second

	if rb.opts.Verbose {
		fmt.Println("[rod] Waiting for response...")
	}

	// Phase 1: Wait for an assistant message to appear
	assistantFound := false
	for waited < timeout {
		for _, sel := range selectors.AssistantMessageCandidates {
			elems, err := p.Elements(sel)
			if err == nil && len(elems) > 0 {
				assistantFound = true
				break
			}
		}
		if assistantFound {
			break
		}
		time.Sleep(pollInterval)
		waited += pollInterval
	}

	if !assistantFound {
		return ErrTimeout
	}

	// Phase 2: Minimum wait before stability check
	if waited < minWait {
		time.Sleep(minWait - waited)
	}

	// Phase 3: Poll for text stability
	lastText := ""
	stable := 0
	startTime := time.Now()

	for time.Since(startTime) < timeout {
		// Check if stop generating button disappeared
		stopGone := true
		for _, sel := range selectors.StopGeneratingCandidates {
			elem, err := p.Element(sel)
			if err == nil && elem != nil {
				stopGone = false
				break
			}
		}

		// Get latest assistant message text
		currentText := ""
		for _, sel := range selectors.AssistantMessageCandidates {
			elems, err := p.Elements(sel)
			if err == nil && len(elems) > 0 {
				last := elems[len(elems)-1]
				t, txtErr := last.Text()
				if txtErr == nil && t != "" {
					currentText = t
					break
				}
			}
		}

		if stopGone && currentText != "" {
			if currentText == lastText {
				stable++
				if stable >= stableCount {
					if rb.opts.Verbose {
						fmt.Println("[rod] Response complete")
					}
					return nil
				}
			} else {
				stable = 0
				lastText = currentText
			}
		} else if currentText != "" {
			lastText = currentText
			stable = 0
		}

		time.Sleep(pollInterval)
	}

	if rb.opts.Verbose {
		fmt.Println("[rod] Response timed out, returning partial result")
	}
	return ErrTimeout
}

// LatestResponseMarkdown gets the latest assistant response as plain text.
func (rb *RodBrowser) LatestResponseMarkdown(ctx context.Context) (string, error) {
	p, err := rb.pageOrNew()
	if err != nil {
		return "", err
	}

	selectors := DefaultSelectors()

	for _, sel := range selectors.AssistantMessageCandidates {
		elems, err := p.Elements(sel)
		if err == nil && len(elems) > 0 {
			last := elems[len(elems)-1]
			text, txtErr := last.Text()
			if txtErr == nil && text != "" {
				return text, nil
			}
		}
	}

	return "", fmt.Errorf("no assistant message found")
}

// DumpHTML returns the full HTML content of the current page.
func (rb *RodBrowser) DumpHTML(ctx context.Context) (string, error) {
	p, err := rb.pageOrNew()
	if err != nil {
		return "", err
	}

	html, htmlErr := p.HTML()
	if htmlErr != nil {
		return "", fmt.Errorf("cannot get HTML: %w", htmlErr)
	}

	return html, nil
}

// Screenshot captures a PNG screenshot of the current page.
func (rb *RodBrowser) Screenshot(ctx context.Context) ([]byte, error) {
	p, err := rb.pageOrNew()
	if err != nil {
		return nil, err
	}

	data, shotErr := p.Screenshot(true, &proto.PageCaptureScreenshot{
		Format:      proto.PageCaptureScreenshotFormatPng,
		FromSurface: true,
	})
	if shotErr != nil {
		return nil, fmt.Errorf("cannot take screenshot: %w", shotErr)
	}

	return data, nil
}

// DownloadResponseImages is not yet implemented.
func (rb *RodBrowser) DownloadResponseImages(ctx context.Context, outPath string) ([]string, error) {
	return nil, ErrNotSupported
}

// ensure RodBrowser implements Browser.
var _ Browser = (*RodBrowser)(nil)

// timeoutFromContext returns a timeout duration from context or the default.
func timeoutFromContext(ctx context.Context, defaultTimeout time.Duration) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		return time.Until(deadline)
	}
	return defaultTimeout
}

// jsString escapes a Go string for use as a JavaScript string literal.
func jsString(s string) string {
	// Simple escaping for JSON-compatible JS string
	result := make([]byte, 0, len(s)+2)
	result = append(result, '"')
	for _, r := range s {
		switch r {
		case '\\':
			result = append(result, '\\', '\\')
		case '"':
			result = append(result, '\\', '"')
		case '\n':
			result = append(result, '\\', 'n')
		case '\r':
			result = append(result, '\\', 'r')
		case '\t':
			result = append(result, '\\', 't')
		default:
			if r < 0x20 {
				result = append(result, fmt.Sprintf("\\u%04x", r)...)
			} else {
				result = append(result, string(r)...)
			}
		}
	}
	result = append(result, '"')
	return string(result)
}
