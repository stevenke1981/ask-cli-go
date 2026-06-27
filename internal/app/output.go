package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultOutputName returns a timestamped output filename (e.g. ask-20260627-124500.md).
func DefaultOutputName(subdir string) string {
	name := fmt.Sprintf("ask-%s.md", time.Now().Format("20060102-150405"))
	if subdir == "" {
		return name
	}
	return filepath.Join("sessions", subdir, name)
}

// WriteOutput writes the content to stdout and saves to a file.
//
//   - outputPath: explicit path; if non-empty, it is used as-is (subdir ignored).
//   - subdir:     when outputPath is empty, auto-saves to sessions/<subdir>/ask-<timestamp>.md.
//   - content:    the response text (Markdown).
func WriteOutput(content string, outputPath string, subdir string) error {
	// Always print to stdout
	fmt.Print(content)

	// Ensure trailing newline for terminal readability
	if len(content) > 0 && content[len(content)-1] != '\n' {
		fmt.Println()
	}

	// Determine output path
	savePath := outputPath
	if savePath == "" {
		savePath = DefaultOutputName(subdir)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(savePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("cannot create output directory: %w", err)
		}
	}

	// Write file
	if err := os.WriteFile(savePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("cannot write output file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\n📄 Saved to %s\n", savePath)
	return nil
}

// WriteOutputBytes writes binary content (e.g., screenshot) to a file.
func WriteOutputBytes(data []byte, outputPath string) error {
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("cannot create output directory: %w", err)
		}
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("cannot write output file: %w", err)
	}

	return nil
}
