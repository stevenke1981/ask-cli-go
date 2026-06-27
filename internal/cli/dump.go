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

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump the current browser tab HTML for debugging",
	Long:  `Opens Chrome, navigates to ChatGPT, and saves the current page HTML to a file for debugging.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDump()
	},
}

func init() {
	registerSubcommand(dumpCmd)
	dumpCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: ~/.ask-cli/dumps/chatgpt-YYYYMMDD-HHMMSS.html)")
}

func runDump() error {
	if backend == "api" {
		return fmt.Errorf("dump requires a browser backend (run with --backend rod)")
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

		html, err := b.DumpHTML(ctx)
		if err != nil {
			return fmt.Errorf("cannot dump HTML: %w", err)
		}

		// Determine output path
		path := outputFile
		if path == "" {
			path = app.DumpPath()
		}

		if err := app.WriteOutput(html, path, "dump"); err != nil {
			return fmt.Errorf("cannot write dump file: %w", err)
		}

		fmt.Printf("HTML dumped to %s\n", path)
		return nil
	})
}
