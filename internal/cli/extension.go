package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/ask-cli/ask-cli/internal/app"
	"github.com/ask-cli/ask-cli/internal/platform"
	"github.com/spf13/cobra"
)

var openChromeExtensions = platform.OpenChromeExtensions
var openExtensionFolder = platform.OpenFolder

var extensionCmd = &cobra.Command{
	Use:   "extension",
	Short: "Set up the normal-Chrome ChatGPT bridge extension",
	Long: `Opens chrome://extensions and the local unpacked extension folder.
Enable Developer mode, click "Load unpacked", and select the opened folder.
This one-time step lets ask-cli interact with ChatGPT in regular Chrome.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runExtensionSetup(cmd.Context(), cmd.OutOrStdout())
	},
}

func init() {
	registerSubcommand(extensionCmd)
}

func runExtensionSetup(ctx context.Context, output io.Writer) error {
	extensionDir, err := app.ExtensionDir()
	if err != nil {
		return err
	}
	if err := openChromeExtensions(ctx); err != nil {
		return err
	}
	if err := openExtensionFolder(extensionDir); err != nil {
		return err
	}

	fmt.Fprintln(output, "One-time Chrome extension setup:")
	fmt.Fprintln(output, "  1. Enable Developer mode in chrome://extensions")
	fmt.Fprintln(output, "  2. Click \"Load unpacked\"")
	fmt.Fprintf(output, "  3. Select: %s\n", extensionDir)
	fmt.Fprintln(output, "  4. Keep the extension enabled, then run: ask \"your prompt\"")
	return nil
}
