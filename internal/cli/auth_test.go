package cli

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/ask-cli/ask-cli/internal/webauth"
)

func TestResolveAuthProviders(t *testing.T) {
	got, err := resolveAuthProviders([]string{"google", "openai"})
	if err != nil {
		t.Fatalf("resolveAuthProviders() error = %v", err)
	}

	want := []string{"gemini", "chatgpt"}
	ids := make([]string, len(got))
	for i, provider := range got {
		ids[i] = provider.ID
	}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("provider IDs = %v, want %v", ids, want)
	}
}

func TestRunWebAuthOpensOfficialURLsInNormalChrome(t *testing.T) {
	original := openAuthURLs
	t.Cleanup(func() {
		openAuthURLs = original
	})

	var gotURLs []string
	openAuthURLs = func(_ context.Context, urls []string) error {
		gotURLs = append([]string(nil), urls...)
		return nil
	}

	var output bytes.Buffer
	if err := runWebAuth(context.Background(), []string{"chatgpt", "gemini", "grok"}, &output); err != nil {
		t.Fatalf("runWebAuth() error = %v", err)
	}

	wantURLs := []string{
		"https://chatgpt.com/",
		"https://gemini.google.com/app",
		"https://grok.com/",
	}
	if !reflect.DeepEqual(gotURLs, wantURLs) {
		t.Fatalf("opened URLs = %v, want %v", gotURLs, wantURLs)
	}
	if !bytes.Contains(output.Bytes(), []byte("regular Google Chrome profile")) {
		t.Fatalf("output = %q, want normal Chrome profile explanation", output.String())
	}
}

func TestAuthCommandContract(t *testing.T) {
	if authCmd == nil {
		t.Fatal("authCmd = nil")
	}
	if authCmd.Name() != "auth" {
		t.Fatalf("authCmd.Name() = %q, want auth", authCmd.Name())
	}
	if authCmd.Flags().Lookup("auth-timeout") != nil {
		t.Fatal("auth command must not expose automation polling flags")
	}

	providers, err := webauth.ResolveProviders(nil)
	if err != nil || len(providers) != 3 {
		t.Fatalf("default providers = %v, %v; want all three", providers, err)
	}
}
