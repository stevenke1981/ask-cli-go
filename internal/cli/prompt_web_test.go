package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ask-cli/ask-cli/internal/webbridge"
)

type fakePromptBridge struct {
	triggerURL string
	result     webbridge.Result
	startErr   error
	claimErr   error
	resultErr  error
	started    bool
	closed     bool
}

func (f *fakePromptBridge) Start(context.Context) error {
	f.started = true
	return f.startErr
}

func (f *fakePromptBridge) TriggerURL() string {
	return f.triggerURL
}

func (f *fakePromptBridge) WaitForClaim(context.Context) error {
	return f.claimErr
}

func (f *fakePromptBridge) WaitForResult(context.Context) (webbridge.Result, error) {
	return f.result, f.resultErr
}

func (f *fakePromptBridge) Close(context.Context) error {
	f.closed = true
	return nil
}

func TestRunPromptWebOpensNormalChromeAndWritesResponse(t *testing.T) {
	bridge := &fakePromptBridge{
		triggerURL: "https://chatgpt.com/#ask-cli-token=test&ask-cli-port=1234",
		result: webbridge.Result{
			ID:      "task",
			Content: "response from normal Chrome",
		},
	}
	restorePromptWebGlobals(t)
	newWebPromptBridge = func(string, time.Duration) (promptBridge, error) {
		return bridge, nil
	}

	var openedURL string
	openWebPromptURL = func(_ context.Context, targetURL string) error {
		openedURL = targetURL
		return nil
	}
	outputFile = filepath.Join(t.TempDir(), "response.md")
	timeoutSec = 10
	images = nil

	if err := runPromptWeb("hello"); err != nil {
		t.Fatalf("runPromptWeb() error = %v", err)
	}
	if !bridge.started || !bridge.closed {
		t.Fatalf("bridge lifecycle: started=%v closed=%v", bridge.started, bridge.closed)
	}
	if openedURL != bridge.triggerURL {
		t.Fatalf("opened URL = %q, want %q", openedURL, bridge.triggerURL)
	}
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "response from normal Chrome" {
		t.Fatalf("output = %q", data)
	}
}

func TestRunPromptWebGuidesExtensionSetupWhenUnclaimed(t *testing.T) {
	restorePromptWebGlobals(t)
	newWebPromptBridge = func(string, time.Duration) (promptBridge, error) {
		return &fakePromptBridge{
			triggerURL: "https://chatgpt.com/#ask-cli-token=test&ask-cli-port=1234",
			claimErr:   context.DeadlineExceeded,
		}, nil
	}
	openWebPromptURL = func(context.Context, string) error { return nil }
	timeoutSec = 10
	images = nil

	err := runPromptWeb("hello")
	if err == nil || !strings.Contains(err.Error(), "ask extension") {
		t.Fatalf("runPromptWeb() error = %v, want extension setup guidance", err)
	}
}

func TestRunPromptWebRejectsImages(t *testing.T) {
	restorePromptWebGlobals(t)
	images = []string{"image.png"}
	if err := runPromptWeb("hello"); err == nil || !strings.Contains(err.Error(), "does not support --image") {
		t.Fatalf("runPromptWeb() error = %v, want image error", err)
	}
}

func TestRunPromptWebReturnsPageError(t *testing.T) {
	restorePromptWebGlobals(t)
	newWebPromptBridge = func(string, time.Duration) (promptBridge, error) {
		return &fakePromptBridge{
			triggerURL: "https://chatgpt.com/#ask-cli-token=test&ask-cli-port=1234",
			result: webbridge.Result{
				ID:    "task",
				Error: "ChatGPT composer not found",
			},
		}, nil
	}
	openWebPromptURL = func(context.Context, string) error { return nil }
	timeoutSec = 10
	images = nil

	err := runPromptWeb("hello")
	if err == nil || !strings.Contains(err.Error(), "composer not found") {
		t.Fatalf("runPromptWeb() error = %v, want page error", err)
	}
}

func TestDefaultBackendIsWeb(t *testing.T) {
	if backend != "web" {
		t.Fatalf("default backend = %q, want web", backend)
	}
}

func restorePromptWebGlobals(t *testing.T) {
	t.Helper()
	originalFactory := newWebPromptBridge
	originalOpener := openWebPromptURL
	originalOutput := outputFile
	originalTimeout := timeoutSec
	originalImages := images
	t.Cleanup(func() {
		newWebPromptBridge = originalFactory
		openWebPromptURL = originalOpener
		outputFile = originalOutput
		timeoutSec = originalTimeout
		images = originalImages
	})
}
