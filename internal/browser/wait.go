package browser

import "context"

// WaitCondition defines how to wait for a page state.
type WaitCondition int

const (
	// WaitVisible waits for the element to be visible.
	WaitVisible WaitCondition = iota
	// WaitStable waits for the element's content to stop changing.
	WaitStable
	// WaitGone waits for the element to disappear.
	WaitGone
)

// PollConfig defines polling behavior for waiting on elements.
type PollConfig struct {
	Interval    int // milliseconds between polls
	StableCount int // number of consecutive stable readings
	MinWaitMs   int // minimum wait time before checking stability
}

// DefaultPollConfig returns the default polling configuration.
func DefaultPollConfig() PollConfig {
	return PollConfig{
		Interval:    500,
		StableCount: 4,
		MinWaitMs:   2000,
	}
}

// WaitForElement waits for a DOM element to match the given condition.
// This is a placeholder; actual implementation depends on the backend.
func WaitForElement(ctx context.Context, selector string, condition WaitCondition) error {
	// Backend-specific implementations will override this.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
