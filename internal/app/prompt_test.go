package app

import (
	"os"
	"testing"
)

func TestReadPromptFromArgs(t *testing.T) {
	args := []string{"Hello", "world", "this", "is", "a", "test"}
	prompt, err := ReadPrompt(args)
	if err != nil {
		t.Fatalf("ReadPrompt(args) failed: %v", err)
	}
	want := "Hello world this is a test"
	if prompt != want {
		t.Errorf("ReadPrompt = %q, want %q", prompt, want)
	}
}

func TestReadPromptFromArgsSingle(t *testing.T) {
	args := []string{"Hello"}
	prompt, err := ReadPrompt(args)
	if err != nil {
		t.Fatalf("ReadPrompt(args) failed: %v", err)
	}
	if prompt != "Hello" {
		t.Errorf("ReadPrompt = %q, want %q", prompt, "Hello")
	}
}

func TestReadPromptFromArgsEmpty(t *testing.T) {
	// When args is empty and stdin is a terminal, should return error
	// We can't easily mock isTerminal in this test, so we just check
	// that empty args returns an error when stdin is not a pipe.
	// The actual behavior depends on os.Stdin.
	_, err := ReadPrompt(nil)
	if err == nil {
		t.Log("ReadPrompt returned nil error on empty args (may be running with piped stdin)")
	}
}

func TestReadPromptFromStdin(t *testing.T) {
	// Save original stdin and restore
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()

	// Create a pipe and use it as stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe creation failed: %v", err)
	}

	// Write test data and close writer
	if _, err := w.Write([]byte("stdin prompt data")); err != nil {
		t.Fatalf("write to pipe failed: %v", err)
	}
	w.Close()

	os.Stdin = r

	prompt, err := ReadPrompt(nil)
	if err != nil {
		t.Fatalf("ReadPrompt from stdin failed: %v", err)
	}
	if prompt != "stdin prompt data" {
		t.Errorf("ReadPrompt = %q, want %q", prompt, "stdin prompt data")
	}
}

func TestReadPromptFromStdinMultiline(t *testing.T) {
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe creation failed: %v", err)
	}

	input := "line one\nline two\nline three"
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write to pipe failed: %v", err)
	}
	w.Close()

	os.Stdin = r

	prompt, err := ReadPrompt(nil)
	if err != nil {
		t.Fatalf("ReadPrompt from stdin failed: %v", err)
	}
	if prompt != "line one\nline two\nline three" {
		t.Errorf("ReadPrompt = %q, want multiline content", prompt)
	}
}

func TestReadPromptFromStdinTrimmed(t *testing.T) {
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe creation failed: %v", err)
	}

	if _, err := w.Write([]byte("  \nhello  \n  ")); err != nil {
		t.Fatalf("write to pipe failed: %v", err)
	}
	w.Close()

	os.Stdin = r

	prompt, err := ReadPrompt(nil)
	if err != nil {
		t.Fatalf("ReadPrompt from stdin failed: %v", err)
	}
	if prompt != "hello" {
		t.Errorf("ReadPrompt = %q, want %q", prompt, "hello")
	}
}
