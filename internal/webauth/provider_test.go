package webauth

import (
	"reflect"
	"testing"
)

func TestResolveProvidersDefaultsToAll(t *testing.T) {
	got, err := ResolveProviders(nil)
	if err != nil {
		t.Fatalf("ResolveProviders(nil) error = %v", err)
	}

	want := []string{"chatgpt", "gemini", "grok"}
	if ids := providerIDs(got); !reflect.DeepEqual(ids, want) {
		t.Fatalf("provider IDs = %v, want %v", ids, want)
	}
}

func TestResolveProvidersAcceptsAliasesAndDeduplicates(t *testing.T) {
	got, err := ResolveProviders([]string{"openai", "google", "xai", "CHATGPT"})
	if err != nil {
		t.Fatalf("ResolveProviders() error = %v", err)
	}

	want := []string{"chatgpt", "gemini", "grok"}
	if ids := providerIDs(got); !reflect.DeepEqual(ids, want) {
		t.Fatalf("provider IDs = %v, want %v", ids, want)
	}
}

func TestResolveProvidersAllCannotBeCombined(t *testing.T) {
	if _, err := ResolveProviders([]string{"all", "grok"}); err == nil {
		t.Fatal("ResolveProviders() error = nil, want an error")
	}
}

func TestResolveProvidersRejectsUnknownProvider(t *testing.T) {
	if _, err := ResolveProviders([]string{"claude"}); err == nil {
		t.Fatal("ResolveProviders() error = nil, want an error")
	}
}

func providerIDs(providers []Provider) []string {
	ids := make([]string, len(providers))
	for i, provider := range providers {
		ids[i] = provider.ID
	}
	return ids
}
