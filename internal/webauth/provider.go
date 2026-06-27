// Package webauth manages persistent browser authorization for supported AI
// web applications without copying session secrets out of Chrome.
package webauth

import (
	"fmt"
	"slices"
	"strings"
)

// Provider describes one supported AI web application's login flow.
type Provider struct {
	ID          string
	DisplayName string
	LoginURL    string
}

var providers = []Provider{
	{
		ID:          "chatgpt",
		DisplayName: "ChatGPT",
		LoginURL:    "https://chatgpt.com/",
	},
	{
		ID:          "gemini",
		DisplayName: "Gemini",
		LoginURL:    "https://gemini.google.com/app",
	},
	{
		ID:          "grok",
		DisplayName: "Grok",
		LoginURL:    "https://grok.com/",
	},
}

var providerAliases = map[string]string{
	"chatgpt": "chatgpt",
	"openai":  "chatgpt",
	"gemini":  "gemini",
	"google":  "gemini",
	"grok":    "grok",
	"xai":     "grok",
	"x.ai":    "grok",
}

// ProviderByName resolves a canonical provider name or documented alias.
func ProviderByName(name string) (Provider, error) {
	canonical, ok := providerAliases[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return Provider{}, fmt.Errorf(
			"unknown provider %q (valid: chatgpt, gemini, grok, all)", name,
		)
	}
	for _, provider := range providers {
		if provider.ID == canonical {
			return provider, nil
		}
	}
	return Provider{}, fmt.Errorf("provider %q is not configured", canonical)
}

// ResolveProviders resolves CLI provider arguments. No arguments and "all"
// both select all providers in the stable registry order.
func ResolveProviders(names []string) ([]Provider, error) {
	if len(names) == 0 {
		return slices.Clone(providers), nil
	}

	if len(names) > 1 {
		for _, name := range names {
			if strings.EqualFold(strings.TrimSpace(name), "all") {
				return nil, fmt.Errorf("provider %q cannot be combined with other providers", "all")
			}
		}
	}
	if len(names) == 1 && strings.EqualFold(strings.TrimSpace(names[0]), "all") {
		return slices.Clone(providers), nil
	}

	resolved := make([]Provider, 0, len(names))
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		provider, err := ProviderByName(name)
		if err != nil {
			return nil, err
		}
		if !seen[provider.ID] {
			resolved = append(resolved, provider)
			seen[provider.ID] = true
		}
	}
	return resolved, nil
}
