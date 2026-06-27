package chatgpt

import "strings"

// HTMLToMarkdown converts ChatGPT assistant message HTML to Markdown.
// This is a simplified converter that handles common elements.
// A full implementation would use goquery for proper HTML parsing.
func HTMLToMarkdown(html string) string {
	// TODO: Implement proper HTML to Markdown conversion using goquery
	if html == "" {
		return ""
	}

	// Remove HTML tags as a basic fallback
	plain := stripHTMLTags(html)
	return strings.TrimSpace(plain)
}

// stripHTMLTags removes HTML tags from the input string.
func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}
