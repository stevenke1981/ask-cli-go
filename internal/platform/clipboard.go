package platform

import (
	"fmt"
	"os/exec"
	"runtime"
)

// CopyToClipboard copies the given text to the system clipboard.
func CopyToClipboard(text string) error {
	switch runtime.GOOS {
	case "windows":
		return copyWindows(text)
	case "darwin":
		return copyMacOS(text)
	case "linux":
		return copyLinux(text)
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
}

func copyWindows(text string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "Set-Clipboard")
	cmd.Stdin = pipeWriter(text)
	return cmd.Run()
}

func copyMacOS(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = pipeWriter(text)
	return cmd.Run()
}

func copyLinux(text string) error {
	// Try xclip first, then wl-copy (Wayland)
	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = pipeWriter(text)
	if err := cmd.Run(); err == nil {
		return nil
	}
	cmd = exec.Command("wl-copy")
	cmd.Stdin = pipeWriter(text)
	return cmd.Run()
}

// pipeWriter returns a string that can be used as stdin for a command.
func pipeWriter(text string) *stringReader {
	return &stringReader{data: text}
}

type stringReader struct {
	data string
	pos  int
}

func (r *stringReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF") // Will be ignored by os/exec
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
