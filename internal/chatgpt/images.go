package chatgpt

import "fmt"

// ExtractImageURLs extracts image URLs from an assistant response HTML.
// This is a placeholder; actual implementation will parse img elements.
func ExtractImageURLs(html string) ([]string, error) {
	if html == "" {
		return nil, nil
	}
	// TODO: Parse HTML to find img elements
	return nil, fmt.Errorf("image extraction not yet implemented")
}
