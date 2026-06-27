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

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Retrieve the latest response from ChatGPT",
	Long: `In browser mode, launches a headless browser, navigates to ChatGPT,
and retrieves the latest assistant response from the current conversation.

In API mode (default), this shows your current session status. Use 'ask "prompt"'
to send a prompt and get a response directly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGet()
	},
}

func init() {
	registerSubcommand(getCmd)
	getCmd.Flags().BoolVar(&headless, "headless", true, "Run Chrome in headless mode")
	getCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write the response to the specified file")
}

func runGet() error {
	switch backend {
	case "api":
		return runGetAPI()
	default:
		return runGetBrowser()
	}
}

func runGetAPI() error {
	session, err := app.LoadSession()
	if err != nil {
		return fmt.Errorf("cannot load session: %w", err)
	}
	if session == nil || session.SessionToken == "" {
		fmt.Println("No saved ChatGPT session found.")
		fmt.Println("Run 'ask login' to extract your ChatGPT session from Chrome.")
		return nil
	}

	fmt.Println("ChatGPT session is active.")
	if session.ProfileSource != "" {
		fmt.Printf("  Profile: %s\n", session.ProfileSource)
	}
	if session.UpdatedAt != "" {
		fmt.Printf("  Last verified: %s\n", session.UpdatedAt)
	}
	fmt.Println()
	fmt.Println("To send a prompt, use: ask \"your question\"")
	return nil
}

func runGetBrowser() error {
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

		response, err := b.LatestResponseMarkdown(ctx)
		if err != nil {
			return fmt.Errorf("cannot get latest response: %w", err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "[ask] Response retrieved\n")
		}

		if err := app.WriteOutput(response, outputFile, "chatgpt"); err != nil {
			return fmt.Errorf("cannot write output: %w", err)
		}

		return nil
	})
}
