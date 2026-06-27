package cli

import (
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open [URL|PROMPT]",
	Short: "Open Chrome browser, optionally navigate to a URL or send a prompt",
	Long: `Opens Chrome browser. If provided with a prompt, sends it to ChatGPT
and retrieves the response. If provided with a URL-like argument,
navigates to that URL. If no argument is given, opens ChatGPT homepage.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPrompt(args)
	},
}

func init() {
	registerSubcommand(openCmd)
	openCmd.Flags().BoolVar(&headless, "headless", true, "Run Chrome in headless mode")
	openCmd.Flags().BoolVar(&newSession, "new", false, "Create a brand new ChatGPT session")
	openCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write the final response to the specified file")
}
