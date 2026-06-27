package chatgpt

import "fmt"

// Response holds the parsed ChatGPT assistant response.
type Response struct {
	// Text is the plain text version of the response.
	Text string

	// Markdown is the Markdown version of the response.
	Markdown string

	// HTML is the raw HTML of the response element.
	HTML string

	// ImageURLs contains URLs of images in the response.
	ImageURLs []string
}

// ParseResponse extracts structured data from an assistant message HTML element.
// This is a placeholder; actual implementation will parse the DOM element.
func ParseResponse(html string) (*Response, error) {
	if html == "" {
		return nil, fmt.Errorf("empty response HTML")
	}

	markdown := HTMLToMarkdown(html)

	return &Response{
		HTML:     html,
		Markdown: markdown,
		Text:     markdown,
	}, nil
}
