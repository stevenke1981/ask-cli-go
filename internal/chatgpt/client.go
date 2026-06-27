// Package chatgpt provides direct HTTP API access to ChatGPT
// using session cookies extracted from a Chrome browser profile.
package chatgpt

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// defaultAPI endpoints
const (
	defaultBaseURL  = "https://chatgpt.com"
	authSessionPath = "/api/auth/session"
	conversationAPI = "/backend-api/conversation"
	defaultModel    = "text-davinci-002-render-sha"
)

// MessagePart represents a single part of a message content.
type MessagePart struct {
	ContentType string `json:"content_type"`
	Parts       []any  `json:"parts"`
}

// MessageAuthor represents the author of a message.
type MessageAuthor struct {
	Role string `json:"role"`
}

// Message represents a single message in a conversation.
type Message struct {
	ID      string        `json:"id"`
	Author  MessageAuthor `json:"author"`
	Content MessagePart   `json:"content"`
}

// ConversationRequest is the payload sent to /backend-api/conversation.
type ConversationRequest struct {
	Action                     string    `json:"action"`
	Messages                   []Message `json:"messages"`
	ConversationID             *string   `json:"conversation_id"`
	ParentMessageID            *string   `json:"parent_message_id"`
	Model                      string    `json:"model"`
	TimezoneOffsetMin          int       `json:"timezone_offset_min"`
	Suggestions                []string  `json:"suggestions"`
	HistoryAndTrainingDisabled bool      `json:"history_and_training_disabled"`
}

// SSEEvent represents a single server-sent event from the API.
type SSEEvent struct {
	ConversationID string          `json:"conversation_id,omitempty"`
	Message        *SSEMessage     `json:"message,omitempty"`
	Error          *string         `json:"error,omitempty"`
	Raw            json.RawMessage `json:"-"`
}

// SSEMessage contains partial message content from an SSE event.
type SSEMessage struct {
	ID      string        `json:"id"`
	Author  MessageAuthor `json:"author"`
	Content struct {
		ContentType string `json:"content_type"`
		Parts       []any  `json:"parts"`
		Text        string `json:"text,omitempty"`
	} `json:"content"`
	Status    string  `json:"status,omitempty"`
	EndTurn   bool    `json:"end_turn,omitempty"`
	Weight    float64 `json:"weight,omitempty"`
	Recipient string  `json:"recipient,omitempty"`
}

// AuthSession is the response from /api/auth/session.
type AuthSession struct {
	AccessToken string `json:"accessToken"`
	User        struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Image string `json:"image"`
	} `json:"user"`
	Expires string `json:"expires"`
}

// Client is an HTTP client for the ChatGPT backend API.
type Client struct {
	httpClient   *http.Client
	baseURL      string
	sessionToken string
	accessToken  string
	userAgent    string
}

// NewClient creates a new ChatGPT API client authenticated with the given
// session token (from the __Secure-next-auth.session-token cookie).
func NewClient(sessionToken string) *Client {
	jar, _ := cookiejar.New(nil)
	baseURL := defaultBaseURL
	cj := &cookieJar{jar: jar, baseURL: baseURL, sessionToken: sessionToken}

	return &Client{
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 90 * time.Second,
			},
			Jar:     cj,
			Timeout: 0, // no timeout at client level; use context
		},
		baseURL:      baseURL,
		sessionToken: sessionToken,
		userAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
	}
}

// Authenticate exchanges the session token for an access token.
// This is optional but recommended — some endpoints require a Bearer token.
func (c *Client) Authenticate(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+authSessionPath, nil)
	if err != nil {
		return fmt.Errorf("create auth request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var session AuthSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return fmt.Errorf("parse auth response: %w", err)
	}

	if session.AccessToken == "" {
		return fmt.Errorf("no access token in auth response")
	}

	c.accessToken = session.AccessToken
	return nil
}

// IsAuthenticated reports whether an access token has been obtained.
func (c *Client) IsAuthenticated() bool {
	return c.accessToken != ""
}

// SendPrompt sends a single user message and returns the full assistant response.
// This creates a new conversation if no conversation is active.
func (c *Client) SendPrompt(ctx context.Context, prompt string) (*Response, error) {
	// Generate a unique message ID
	msgID := newUUID()

	parentMsgID := newUUID()
	convReq := ConversationRequest{
		Action: "next",
		Messages: []Message{
			{
				ID:     msgID,
				Author: MessageAuthor{Role: "user"},
				Content: MessagePart{
					ContentType: "text",
					Parts:       []any{prompt},
				},
			},
		},
		ConversationID:             nil,
		ParentMessageID:            &parentMsgID,
		Model:                      defaultModel,
		TimezoneOffsetMin:          -480,
		Suggestions:                []string{},
		HistoryAndTrainingDisabled: false,
	}

	return c.sendConversationRequest(ctx, convReq)
}

// sendConversationRequest sends a conversation request and processes the SSE stream.
func (c *Client) sendConversationRequest(ctx context.Context, req ConversationRequest) (*Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+conversationAPI, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("User-Agent", c.userAgent)
	if c.accessToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("conversation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("conversation API error (HTTP %d): %s", resp.StatusCode, string(rBody))
	}

	return parseSSEStream(resp.Body)
}

// parseSSEStream reads the SSE response from the ChatGPT API and extracts
// the final assistant message.
func parseSSEStream(r io.Reader) (*Response, error) {
	scanner := bufio.NewScanner(r)
	// SSE lines can be longer than the default buffer
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var fullText strings.Builder
	var lastError string

	for scanner.Scan() {
		line := scanner.Text()

		// SSE data lines start with "data: "
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimSpace(line[6:]) // strip "data: "

		// Ignore empty or done markers
		if data == "" || data == "[DONE]" {
			continue
		}

		var evt SSEEvent
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			continue // skip malformed events
		}

		if evt.Error != nil && *evt.Error != "" {
			lastError = *evt.Error
			continue
		}

		if evt.Message != nil {
			// Extract text parts
			for _, part := range evt.Message.Content.Parts {
				if s, ok := part.(string); ok {
					fullText.WriteString(s)
				}
			}
			// Some versions use Content.Text
			if evt.Message.Content.Text != "" {
				fullText.WriteString(evt.Message.Content.Text)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("SSE read error: %w", err)
	}

	if lastError != "" {
		return nil, fmt.Errorf("chatgpt API error: %s", lastError)
	}

	text := strings.TrimSpace(fullText.String())
	if text == "" {
		return nil, fmt.Errorf("empty response from ChatGPT")
	}

	return &Response{
		Text:     text,
		Markdown: text,
	}, nil
}

// cookieJar is a custom http.CookieJar that adds the session cookie to every
// request to the base URL.
type cookieJar struct {
	jar          *cookiejar.Jar
	baseURL      string
	sessionToken string
}

func (c *cookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.jar.SetCookies(u, cookies)
}

func (c *cookieJar) Cookies(u *url.URL) []*http.Cookie {
	cookies := c.jar.Cookies(u)
	// Add the secure session token for chatgpt.com
	if strings.HasSuffix(u.Host, "chatgpt.com") || strings.HasSuffix(u.Host, "chat.openai.com") {
		cookies = append(cookies, &http.Cookie{
			Name:     "__Secure-next-auth.session-token",
			Value:    c.sessionToken,
			Domain:   ".chatgpt.com",
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}
	return cookies
}

// newUUID generates a proper RFC 4122 v4 UUID using crypto/rand.
func newUUID() string {
	u := make([]byte, 16)
	_, _ = rand.Read(u)
	// Set version 4
	u[6] = (u[6] & 0x0f) | 0x40
	// Set variant bits (10xx)
	u[8] = (u[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}
