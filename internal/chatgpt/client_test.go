package chatgpt

import (
	"strings"
	"testing"
)

func TestParseSSEStream_SimpleResponse(t *testing.T) {
	input := `data: {"message":{"id":"msg1","author":{"role":"assistant"},"content":{"content_type":"text","parts":["Hello! How can I help you?"]},"end_turn":true}}

data: {"conversation_id":"conv1","error":null}
`
	resp, err := parseSSEStream(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseSSEStream failed: %v", err)
	}
	if resp == nil {
		t.Fatal("parseSSEStream returned nil")
	}
	if resp.Text != "Hello! How can I help you?" {
		t.Errorf("Text = %q, want %q", resp.Text, "Hello! How can I help you?")
	}
	if resp.Markdown != "Hello! How can I help you?" {
		t.Errorf("Markdown = %q, want %q", resp.Markdown, "Hello! How can I help you?")
	}
}

func TestParseSSEStream_MultipleMessages(t *testing.T) {
	input := `data: {"message":{"id":"msg1","author":{"role":"assistant"},"content":{"content_type":"text","parts":["Building"," the"]},"end_turn":false}}

data: {"message":{"id":"msg2","author":{"role":"assistant"},"content":{"content_type":"text","parts":[" response..."]},"end_turn":true}}

data: {"conversation_id":"conv2","error":null}
`
	resp, err := parseSSEStream(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseSSEStream failed: %v", err)
	}
	if resp.Text != "Building the response..." {
		t.Errorf("Text = %q, want %q", resp.Text, "Building the response...")
	}
}

func TestParseSSEStream_WithPartsArray(t *testing.T) {
	input := `data: {"message":{"id":"m1","author":{"role":"assistant"},"content":{"content_type":"text","parts":["First ","second ","third"]},"end_turn":true}}

data: {"conversation_id":"c1","error":null}
`
	resp, err := parseSSEStream(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseSSEStream failed: %v", err)
	}
	if resp.Text != "First second third" {
		t.Errorf("Text = %q, want %q", resp.Text, "First second third")
	}
}

func TestParseSSEStream_ContentText(t *testing.T) {
	// Some versions use Content.Text instead of Parts
	input := `data: {"message":{"id":"m1","author":{"role":"assistant"},"content":{"content_type":"text","text":"Direct text response"},"end_turn":true}}

data: {"conversation_id":"c1","error":null}
`
	resp, err := parseSSEStream(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseSSEStream failed: %v", err)
	}
	if resp.Text != "Direct text response" {
		t.Errorf("Text = %q, want %q", resp.Text, "Direct text response")
	}
}

func TestParseSSEStream_ApiError(t *testing.T) {
	input := `data: {"conversation_id":"c1","error":"rate_limit_exceeded"}

data: {"conversation_id":"c1","error":null}
`
	_, err := parseSSEStream(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for rate_limit_exceeded, got nil")
	}
	if !strings.Contains(err.Error(), "rate_limit_exceeded") {
		t.Errorf("error = %q, should contain rate_limit_exceeded", err.Error())
	}
}

func TestParseSSEStream_EmptyResponse(t *testing.T) {
	input := `data: {"conversation_id":"c1","error":null}
`
	_, err := parseSSEStream(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for empty response, got nil")
	}
}

func TestParseSSEStream_EmptyInput(t *testing.T) {
	_, err := parseSSEStream(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseSSEStream_DoneMarker(t *testing.T) {
	input := `data: {"message":{"id":"m1","author":{"role":"assistant"},"content":{"content_type":"text","parts":["Done"]},"end_turn":true}}

data: [DONE]

data: {"conversation_id":"c1","error":null}
`
	resp, err := parseSSEStream(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseSSEStream failed: %v", err)
	}
	if resp.Text != "Done" {
		t.Errorf("Text = %q, want %q", resp.Text, "Done")
	}
}

func TestParseSSEStream_NonDataLines(t *testing.T) {
	input := `: keepalive comment

data: {"message":{"id":"m1","author":{"role":"assistant"},"content":{"content_type":"text","parts":["Response"]},"end_turn":true}}

event: done
data: {"conversation_id":"c1","error":null}
`
	resp, err := parseSSEStream(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseSSEStream failed: %v", err)
	}
	if resp.Text != "Response" {
		t.Errorf("Text = %q, want %q", resp.Text, "Response")
	}
}

func TestParseSSEStream_MalformedJSON(t *testing.T) {
	input := `data: {"message": {"id":"m1","author":{"role":"assis

data: {"conversation_id":"c1","error":null}
`
	_, err := parseSSEStream(strings.NewReader(input))
	// Malformed JSON for the message should be skipped; as long as
	// there's no error reported and a non-empty response is found,
	// we consider it acceptable. If no valid message event, expect error.
	if err == nil {
		t.Fatal("expected error for malformed input (no valid message events)")
	}
}

func TestNewUUID_Format(t *testing.T) {
	u := newUUID()
	if len(u) != 36 {
		t.Errorf("UUID length = %d, want 36", len(u))
	}
	if u[8] != '-' || u[13] != '-' || u[18] != '-' || u[23] != '-' {
		t.Errorf("UUID format incorrect: %q", u)
	}
	if u[14] != '4' {
		t.Errorf("UUID version should be 4 at position 14, got %c", u[14])
	}
}

func TestNewUUID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		u := newUUID()
		if seen[u] {
			t.Fatalf("duplicate UUID generated at iteration %d: %s", i, u)
		}
		seen[u] = true
	}
}
