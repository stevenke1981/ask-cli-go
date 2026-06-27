package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ask-cli/ask-cli/internal/app"
	"github.com/ask-cli/ask-cli/internal/browser"
	"github.com/spf13/cobra"
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Take a screenshot of the current browser tab for debugging",
	Long:  `Opens Chrome, navigates to ChatGPT, and takes a screenshot of the page for debugging.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScreenshot()
	},
}

func init() {
	registerSubcommand(screenshotCmd)
	screenshotCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path for the PNG screenshot (default: ~/.ask-cli/screenshots/chatgpt-YYYYMMDD-HHMMSS.png)")
}

func runScreenshot() error {
	if backend == "api" {
		return fmt.Errorf("screenshot requires a browser backend (run with --backend rod)")
	}

	if err := app.EnsureDirs(); err != nil {
		return fmt.Errorf("cannot create data directories: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	return withBrowser(ctx, func(b browser.Browser) error {
		if verbose {
			fmt.Fprintf(os.Stderr, "[ask] Navigating to ChatGPT...\n")
		}

		if err := b.Navigate(ctx, "https://chatgpt.com/"); err != nil {
			return fmt.Errorf("cannot navigate to ChatGPT: %w", err)
		}

		data, err := b.Screenshot(ctx)
		if err != nil {
			return fmt.Errorf("cannot take screenshot: %w", err)
		}

		// Determine output path
		path := outputFile
		if path == "" {
			path = app.ScreenshotPath()
		}

		if err := app.WriteOutputBytes(data, path); err != nil {
			return fmt.Errorf("cannot write screenshot file: %w", err)
		}

		fmt.Printf("Screenshot saved to %s\n", path)
		return nil
	})
}
