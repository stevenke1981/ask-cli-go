package browser

import "context"

// ChromedpBrowser implements the Browser interface using the chromedp library.
type ChromedpBrowser struct {
	// TODO: Add chromedp context and allocator fields when chromedp dependency is added.
}

// Start launches Chrome using chromedp.
func (cb *ChromedpBrowser) Start(ctx context.Context, opts BrowserOptions) error {
	// TODO: Implement chromedp exec allocator
	_ = opts
	return errNotImplemented("chromedp")
}

// Stop closes the browser.
func (cb *ChromedpBrowser) Stop(ctx context.Context) error {
	return nil
}

// Navigate directs the browser.
func (cb *ChromedpBrowser) Navigate(ctx context.Context, url string) error {
	return errNotImplemented("chromedp")
}

// NewChat starts a new conversation.
func (cb *ChromedpBrowser) NewChat(ctx context.Context) error {
	return ErrNotSupported
}

// AttachImages is not supported in chromedp backend yet.
func (cb *ChromedpBrowser) AttachImages(ctx context.Context, paths []string) error {
	return ErrNotSupported
}

// SendPrompt sends a prompt.
func (cb *ChromedpBrowser) SendPrompt(ctx context.Context, prompt string) error {
	return errNotImplemented("chromedp")
}

// WaitForResponseDone waits for response.
func (cb *ChromedpBrowser) WaitForResponseDone(ctx context.Context) error {
	return errNotImplemented("chromedp")
}

// LatestResponseMarkdown gets the response.
func (cb *ChromedpBrowser) LatestResponseMarkdown(ctx context.Context) (string, error) {
	return "", errNotImplemented("chromedp")
}

// DumpHTML returns page HTML.
func (cb *ChromedpBrowser) DumpHTML(ctx context.Context) (string, error) {
	return "", errNotImplemented("chromedp")
}

// Screenshot captures a screenshot.
func (cb *ChromedpBrowser) Screenshot(ctx context.Context) ([]byte, error) {
	return nil, errNotImplemented("chromedp")
}

// DownloadResponseImages is not supported in chromedp backend yet.
func (cb *ChromedpBrowser) DownloadResponseImages(ctx context.Context, outPath string) ([]string, error) {
	return nil, ErrNotSupported
}

// ensure ChromedpBrowser implements Browser.
var _ Browser = (*ChromedpBrowser)(nil)

func errNotImplemented(backend string) error {
	return &ErrNotImplemented{Backend: backend}
}

// ErrNotImplemented is returned when a feature is not yet implemented for a backend.
type ErrNotImplemented struct {
	Backend string
}

func (e *ErrNotImplemented) Error() string {
	return e.Backend + " backend not yet implemented"
}
