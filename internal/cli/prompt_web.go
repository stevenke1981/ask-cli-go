package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ask-cli/ask-cli/internal/app"
	"github.com/ask-cli/ask-cli/internal/platform"
	"github.com/ask-cli/ask-cli/internal/webbridge"
)

const extensionConnectTimeout = 12 * time.Second

type promptBridge interface {
	Start(context.Context) error
	TriggerURL() string
	WaitForClaim(context.Context) error
	WaitForResult(context.Context) (webbridge.Result, error)
	Close(context.Context) error
}

var newWebPromptBridge = func(prompt string, timeout time.Duration) (promptBridge, error) {
	return webbridge.NewServer(prompt, timeout)
}

var openWebPromptURL = func(ctx context.Context, targetURL string) error {
	return platform.OpenChromeURLs(ctx, []string{targetURL})
}

func runPromptWeb(prompt string) error {
	if len(images) > 0 {
		return fmt.Errorf("the normal-Chrome web bridge does not support --image yet")
	}

	timeout := time.Duration(timeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	bridge, err := newWebPromptBridge(prompt, timeout)
	if err != nil {
		return fmt.Errorf("create ChatGPT web bridge: %w", err)
	}
	if err := bridge.Start(ctx); err != nil {
		return fmt.Errorf("start ChatGPT web bridge: %w", err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer closeCancel()
		_ = bridge.Close(closeCtx)
	}()

	if verbose {
		fmt.Fprintln(os.Stderr, "[ask] Opening ChatGPT in regular Google Chrome...")
	}
	if err := openWebPromptURL(ctx, bridge.TriggerURL()); err != nil {
		return fmt.Errorf("open ChatGPT in regular Chrome: %w", err)
	}

	claimTimeout := extensionConnectTimeout
	if timeout < claimTimeout {
		claimTimeout = timeout
	}
	claimCtx, claimCancel := context.WithTimeout(ctx, claimTimeout)
	err = bridge.WaitForClaim(claimCtx)
	claimCancel()
	if err != nil {
		return fmt.Errorf(
			"ChatGPT bridge extension did not connect; run 'ask extension' for one-time setup: %w",
			err,
		)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, "[ask] Prompt claimed by the normal-Chrome extension; waiting for ChatGPT...")
	}
	result, err := bridge.WaitForResult(ctx)
	if err != nil {
		return err
	}
	if result.Error != "" {
		return fmt.Errorf("ChatGPT page interaction failed: %s", result.Error)
	}
	if result.Content == "" {
		return fmt.Errorf("ChatGPT returned an empty response")
	}

	if err := app.WriteOutput(result.Content, outputFile, "chatgpt"); err != nil {
		return fmt.Errorf("cannot write output: %w", err)
	}
	return nil
}
