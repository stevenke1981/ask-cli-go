package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/ask-cli/ask-cli/internal/app"
	"github.com/ask-cli/ask-cli/internal/platform"
	"github.com/ask-cli/ask-cli/internal/webauth"
	"github.com/spf13/cobra"
)

var openAuthURLs = platform.OpenChromeURLs

var authCmd = &cobra.Command{
	Use:   "auth [chatgpt|gemini|grok|all]...",
	Short: "Open AI login pages in your regular Google Chrome",
	Long: `Opens each provider's official website in the installed Google Chrome
without Playwright, Rod, CDP, headless mode, remote debugging, or a separate
test profile. Chrome uses your normal browser process and existing user profile.

With no provider argument, ChatGPT, Gemini, and Grok are opened. Complete login
manually in Chrome; the providers and Chrome retain their own normal sessions.
The global --backend and --profile flags intentionally do not affect this
command.

Examples:
  ask auth
  ask auth chatgpt
  ask auth gemini grok
  ask auth openai google xai`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWebAuth(cmd.Context(), args, cmd.OutOrStdout())
	},
}

func init() {
	registerSubcommand(authCmd)
}

func resolveAuthProviders(args []string) ([]webauth.Provider, error) {
	return webauth.ResolveProviders(args)
}

func runWebAuth(ctx context.Context, args []string, output io.Writer) error {
	providers, err := resolveAuthProviders(args)
	if err != nil {
		return err
	}

	urls := make([]string, len(providers))
	for i, provider := range providers {
		urls[i] = provider.LoginURL
	}
	if err := openAuthURLs(ctx, urls); err != nil {
		return fmt.Errorf("open web authorization pages: %w", err)
	}

	fmt.Fprintln(output, "Opened in your regular Google Chrome profile:")
	for _, provider := range providers {
		fmt.Fprintf(output, "  - %s: %s\n", provider.DisplayName, provider.LoginURL)
	}
	fmt.Fprintln(output, "Complete sign-in manually in Chrome. No automated browser is attached.")
	if extensionDir, extensionErr := app.ExtensionDir(); extensionErr == nil {
		fmt.Fprintf(
			output,
			"To use 'ask \"prompt\"' with the ChatGPT page, load this extension once:\n  %s\nRun 'ask extension' to open the setup screens.\n",
			extensionDir,
		)
	}
	return nil
}
