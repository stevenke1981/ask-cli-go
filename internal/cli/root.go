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

var (
	// Global flags
	headless    bool
	newSession  bool
	verbose     bool
	outputFile  string
	imageOutput string
	images      []string
	backend     string
	provider    string // ai provider: chatgpt, gemini, grok
	profileDir  string
	timeoutSec  int
	modelFlag   string // model name override

	// Build info (set via ldflags)
	Version   = "0.1.0"
	CommitSHA = "dev"

	// knownCommands is the set of registered subcommand names for dispatch.
	knownCommands = make(map[string]*cobra.Command)
)

var rootCmd = &cobra.Command{
	Use:   "ask",
	Short: "AI web CLI with persistent browser authorization",
	Long: `A command-line tool that lets you ask questions to ChatGPT, Gemini,
and Grok directly from your terminal using your web subscription.

By default, ask-cli uses a local extension bridge to interact with the
authenticated ChatGPT page in your regular Google Chrome profile.

  ask auth               Open ChatGPT, Gemini, and Grok in regular Chrome
  ask auth gemini        Open one provider in regular Chrome
  ask extension          Set up the normal-Chrome ChatGPT bridge once
  ask login              Extract ChatGPT session from Chrome and verify auth
  ask login gemini       Authenticate with Gemini via OAuth
  ask "prompt"           Send a prompt through the regular Chrome web page
  ask --provider gemini "Ask Gemini"
  ask --provider grok "Ask Grok"
  ask --model gemini-2.5-pro "Specific model"
  ask open               Open Chrome browser (browser mode only)
  ask get                Retrieve the latest response
  ask dump               Dump current page HTML for debugging (browser mode)
  ask screenshot         Take a screenshot of the current page (browser mode)`,
	Args:    cobra.ArbitraryArgs,
	Version: Version,
	RunE:    rootRun,
}

func rootRun(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		// Check if first arg matches a subcommand name
		if sub, ok := knownCommands[args[0]]; ok {
			return sub.RunE(sub, args[1:])
		}
		// Otherwise treat as prompt
		return runPrompt(args)
	}

	// No args — try reading from stdin, fall back to help
	err := runPrompt(args)
	if err == nil {
		return nil
	}
	return cmd.Help()
}

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().StringVar(
		&backend,
		"backend",
		"web",
		"Backend: web (regular Chrome extension), api, rod, chromedp",
	)
	rootCmd.PersistentFlags().StringVar(
		&provider,
		"provider",
		"chatgpt",
		"AI provider: chatgpt, gemini, grok",
	)
	rootCmd.PersistentFlags().StringVar(
		&profileDir,
		"profile",
		"",
		"Chrome user data directory (default: ~/.ask-cli/chrome-profile)",
	)
	rootCmd.PersistentFlags().IntVar(&timeoutSec, "timeout", 180, "Maximum wait time in seconds for a response")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print verbose debugging status messages")
	rootCmd.PersistentFlags().StringVar(
		&modelFlag,
		"model",
		"",
		"Model override (e.g. gemini-2.5-pro, gemini-2.5-flash, gemini-3.5-flash, grok-3, grok-3-r1, grok-build, grok-build-0.1, grok-composer-2.5-fast)",
	)

	// Local flags for root (ask command)
	rootCmd.Flags().BoolVar(&headless, "headless", true, "Run Chrome in headless mode")
	rootCmd.Flags().BoolVar(&newSession, "new", false, "Create a brand new ChatGPT session")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write the final response in Markdown format to the specified file")
	rootCmd.Flags().StringVarP(&imageOutput, "image-output", "i", "", "Write downloaded images to the specified folder or file path")
	rootCmd.Flags().StringSliceVar(&images, "image", nil, "Attach one or more local image files to the prompt")
}

// registerSubcommand adds a command and stores it in the lookup map.
func registerSubcommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
	knownCommands[cmd.Name()] = cmd
}

// newBrowserFromFlags creates a Browser from global CLI flags.
func newBrowserFromFlags() browser.Browser {
	return browser.NewBrowser(backend)
}

// resolveProfileDir returns the effective profile directory.
func resolveProfileDir() string {
	if profileDir != "" {
		return profileDir
	}
	return app.DefaultProfileDir()
}

// browserOptionsFromFlags creates BrowserOptions from global CLI flags.
func browserOptionsFromFlags() browser.BrowserOptions {
	return browser.BrowserOptions{
		Headless:   headless,
		NewSession: newSession,
		Verbose:    verbose,
		ProfileDir: resolveProfileDir(),
		Timeout:    time.Duration(timeoutSec) * time.Second,
	}
}

// withBrowser is a helper that starts a browser, runs fn, and stops the browser.
func withBrowser(ctx context.Context, fn func(b browser.Browser) error) error {
	b := newBrowserFromFlags()
	opts := browserOptionsFromFlags()

	if verbose {
		fmt.Printf("[ask] Starting browser (backend=%s, headless=%v, profile=%s)\n",
			backend, opts.Headless, opts.ProfileDir)
	}

	if err := b.Start(ctx, opts); err != nil {
		return fmt.Errorf("browser start failed: %w", err)
	}

	err := fn(b)

	if stopErr := b.Stop(ctx); stopErr != nil && err == nil {
		err = fmt.Errorf("browser stop failed: %w", stopErr)
	}

	return err
}

// Execute adds all child commands to the root command and sets flags.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
