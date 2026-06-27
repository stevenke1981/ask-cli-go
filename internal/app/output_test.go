package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteOutputStdout(t *testing.T) {
	// WriteOutput with empty outputPath prints to stdout.
	// We can't easily capture stdout in tests without a helper,
	// but we can verify it doesn't error.
	orig := os.Stdout
	defer func() { os.Stdout = orig }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	os.Stdout = w

	err = WriteOutput("hello stdout", "", "test")
	w.Close()
	if err != nil {
		t.Fatalf("WriteOutput to stdout failed: %v", err)
	}

	buf := make([]byte, 64)
	n, _ := r.Read(buf)
	got := strings.TrimSpace(string(buf[:n]))
	if got != "hello stdout" {
		t.Errorf("stdout = %q, want %q", got, "hello stdout")
	}
}

func TestWriteOutputFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.md")

	if err := WriteOutput("# Hello\n\nWorld", outputPath, ""); err != nil {
		t.Fatalf("WriteOutput to file failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading output file failed: %v", err)
	}
	if string(data) != "# Hello\n\nWorld" {
		t.Errorf("file content = %q, want %q", string(data), "# Hello\n\nWorld")
	}
}

func TestWriteOutputCreatesParentDir(t *testing.T) {
	tmpDir := t.TempDir()
	// Write to a nested directory that doesn't exist yet
	outputPath := filepath.Join(tmpDir, "subdir", "nested", "out.txt")

	if err := WriteOutput("content", outputPath, ""); err != nil {
		t.Fatalf("WriteOutput with nested dir failed: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("output file was not created")
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading output file failed: %v", err)
	}
	if string(data) != "content" {
		t.Errorf("file content = %q, want %q", string(data), "content")
	}
}

func TestWriteOutputBytes(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "image.png")

	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	if err := WriteOutputBytes(data, outputPath); err != nil {
		t.Fatalf("WriteOutputBytes failed: %v", err)
	}

	written, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading binary file failed: %v", err)
	}
	if len(written) != len(data) {
		t.Errorf("file length = %d, want %d", len(written), len(data))
	}
	for i := range data {
		if written[i] != data[i] {
			t.Errorf("byte %d = %02x, want %02x", i, written[i], data[i])
		}
	}
}

func TestWriteOutputBytesCreatesParentDir(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "deeply", "nested", "bin.dat")

	if err := WriteOutputBytes([]byte{0x01, 0x02, 0x03}, outputPath); err != nil {
		t.Fatalf("WriteOutputBytes with nested dir failed: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("binary output file was not created")
	}
}
