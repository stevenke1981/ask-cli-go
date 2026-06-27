package app

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ReadPrompt reads the prompt from either args or stdin.
// If args are non-empty, they are joined with spaces.
// If args are empty and stdin has data available, stdin is read.
// Returns an error if no prompt is available.
func ReadPrompt(args []string) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}

	// Check if stdin has data (pipe/redirect)
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("cannot check stdin: %w", err)
	}

	// Only read from stdin if it's a pipe or redirect (not a terminal)
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", fmt.Errorf("no prompt provided: provide a prompt as an argument or pipe input via stdin")
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("cannot read stdin: %w", err)
	}

	prompt := strings.TrimSpace(string(data))
	if prompt == "" {
		return "", fmt.Errorf("empty prompt from stdin")
	}

	return prompt, nil
}
