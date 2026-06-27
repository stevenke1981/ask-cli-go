package browser

import "fmt"

// SelectorSet holds CSS selector candidates for ChatGPT DOM elements.
// Multiple candidates allow fallback when the ChatGPT UI changes.
type SelectorSet struct {
	// ComposerCandidates are selectors for the prompt input area.
	ComposerCandidates []string

	// SendButtonCandidates are selectors for the send/submit button.
	SendButtonCandidates []string

	// AssistantMessageCandidates are selectors for assistant response elements.
	AssistantMessageCandidates []string

	// FileInputCandidates are selectors for the file upload input.
	FileInputCandidates []string

	// StopGeneratingCandidates are selectors for the stop generation button.
	StopGeneratingCandidates []string

	// NewChatCandidates are selectors for the new chat button/option.
	NewChatCandidates []string
}

// DefaultSelectors returns the default selector set with known ChatGPT selectors.
func DefaultSelectors() *SelectorSet {
	return &SelectorSet{
		ComposerCandidates: []string{
			"#prompt-textarea",
			"textarea",
			"[contenteditable=\"true\"]",
			"[data-testid=\"prompt-textarea\"]",
		},
		SendButtonCandidates: []string{
			"button[data-testid=\"send-button\"]",
			"button[aria-label=\"Send prompt\"]",
			"button:has(svg)",
		},
		AssistantMessageCandidates: []string{
			"[data-message-author-role=\"assistant\"]",
			"article",
			".markdown",
		},
		FileInputCandidates: []string{
			"input[type=\"file\"]",
			"input[accept*=\"image\"]",
		},
		StopGeneratingCandidates: []string{
			"button[data-testid=\"stop-generating-button\"]",
			"button:has(svg.icon-stop)",
		},
		NewChatCandidates: []string{
			"a[href=\"/\"])",
			"nav a:first-child",
			"button:has-text(\"New chat\")",
		},
	}
}

// ErrSelectorNotFound is returned when no candidate selector matches.
type ErrSelectorNotFound struct {
	Category string
}

func (e *ErrSelectorNotFound) Error() string {
	return fmt.Sprintf("selector not found for category: %s", e.Category)
}
