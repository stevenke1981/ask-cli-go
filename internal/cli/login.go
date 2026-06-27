package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ask-cli/ask-cli/internal/app"
	"github.com/ask-cli/ask-cli/internal/chatgpt"
	"github.com/ask-cli/ask-cli/internal/gemini"
	"github.com/ask-cli/ask-cli/internal/grok"
	"github.com/ask-cli/ask-cli/internal/platform"
	"github.com/ask-cli/ask-cli/internal/profile"
	"github.com/spf13/cobra"
)

var (
	loginWeb   bool
	loginToken string
)

var loginCmd = &cobra.Command{
	Use:   "login [chatgpt|gemini|grok]",
	Short: "Set up authentication for an AI provider",
	Long: `Sets up authentication to use AI services.

ChatGPT (default):
  1. Chrome auto-extract: Reads your ChatGPT session from Chrome's profile
  2. --web: Opens a local web server to paste session token
  3. --token: Directly provide the __Secure-next-auth.session-token value

Gemini:
  Opens a browser-based OAuth 2.0 + PKCE flow to authorize ask-cli
  to use your Gemini Advanced subscription.

Grok:
  Prompts for an xAI API key (set XAI_API_KEY environment variable).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		loginTarget := "chatgpt"
		if len(args) > 0 {
			loginTarget = strings.ToLower(args[0])
		}

		switch loginTarget {
		case "chatgpt", "openai":
			return runLogin()
		case "gemini", "google":
			return runLoginGemini()
		case "grok", "xai", "x.ai":
			return runLoginGrok()
		default:
			return fmt.Errorf("unknown login target %q (valid: chatgpt, gemini, grok)", loginTarget)
		}
	},
}

func init() {
	registerSubcommand(loginCmd)
	loginCmd.Flags().BoolVar(&loginWeb, "web", false, "Start a local web server for browser-based token input")
	loginCmd.Flags().StringVar(&loginToken, "token", "", "Provide the session token directly as a string")
}

// runLogin handles ChatGPT login (existing logic).
func runLogin() error {
	// Method 3: --token flag
	if loginToken != "" {
		return loginWithToken(loginToken)
	}

	// Method 2: --web flag (local web server)
	if loginWeb {
		return loginWithWebServer()
	}

	// Method 1: Chrome profile extraction (default)
	return loginWithChromeProfile()
}

// ── Gemini Login (OAuth 2.0 + PKCE) ──

func runLoginGemini() error {
	client := gemini.NewClient()

	// Check for persisted tokens
	tokens, err := app.LoadGeminiTokens()
	if err == nil && tokens != nil {
		client.SetTokens(tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresAt)
		if !client.IsAccessTokenExpired() {
			fmt.Println("✓ Gemini tokens are valid.")
			fmt.Println("  To force re-authentication, delete:")
			fmt.Println("  " + app.GeminiTokensPath())
			return nil
		}
		if client.HasRefreshToken() {
			if verbose {
				fmt.Fprintf(os.Stderr, "[ask] Refreshing Gemini token...\n")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := client.RefreshToken(ctx); err == nil {
				// Save refreshed tokens
				app.SaveGeminiTokens(&app.GeminiTokens{
					AccessToken:  client.AccessToken(),
					RefreshToken: tokens.RefreshToken,
					ExpiresAt:    time.Now().Add(3600 * time.Second),
				})
				fmt.Println("✓ Gemini token refreshed successfully!")
				fmt.Println("  You can now use: ask --provider gemini \"your question\"")
				return nil
			}
			fmt.Fprintf(os.Stderr, "[ask] Token refresh failed, starting fresh OAuth flow.\n")
		}
	}

	// Start OAuth server
	server, loginURL, err := gemini.StartOAuthServer(client)
	if err != nil {
		return fmt.Errorf("cannot start OAuth server: %w", err)
	}
	defer server.Stop()

	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║   Gemini OAuth Authorization                       ║")
	fmt.Println("║                                                    ║")
	fmt.Printf("║   Opening browser for Google OAuth...                ║\n")
	fmt.Println("║                                                    ║")
	fmt.Println("║   1. Sign in to your Google account                 ║")
	fmt.Println("║   2. Grant permissions to ask-cli                   ║")
	fmt.Println("║   3. The page will close automatically              ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")

	// Open auth page in Chrome
	if err := platform.OpenChromeURLs(context.Background(), []string{loginURL}); err != nil {
		fmt.Fprintf(os.Stderr, "[ask] Could not open Chrome: %v\n", err)
		fmt.Fprintf(os.Stderr, "[ask] Open this URL manually:\n%s\n", loginURL)
	}

	// Wait for OAuth completion
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := server.WaitForAuth(ctx); err != nil {
		return fmt.Errorf("OAuth failed: %w", err)
	}

	// Discover GCP project (calls loadCodeAssist / onboardUser)
	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Discovering GCP project...\n")
	}
	discoverCtx, discoverCancel := context.WithTimeout(context.Background(), 60*time.Second)
	if err := client.DiscoverProject(discoverCtx); err != nil {
		discoverCancel()
		fmt.Fprintf(os.Stderr, "[ask] Warning: could not discover GCP project: %v\n", err)
		fmt.Fprintf(os.Stderr, "[ask] Set GOOGLE_CLOUD_PROJECT env var to use Gemini.\n")
		return nil
	}
	discoverCancel()

	// Save tokens + project ID
	saveTokens := app.GeminiTokens{
		AccessToken:  client.AccessToken(),
		RefreshToken: "", // Will be set if we got one
		ExpiresAt:    time.Now().Add(3600 * time.Second),
		ProjectID:    client.ProjectID(),
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Saving Gemini tokens (project: %s)...\n", saveTokens.ProjectID)
	}

	if err := app.SaveGeminiTokens(&saveTokens); err != nil {
		return fmt.Errorf("save gemini tokens: %w", err)
	}

	fmt.Println("\n✓ Gemini OAuth completed successfully!")
	fmt.Printf("  Project: %s\n", saveTokens.ProjectID)
	fmt.Println("  You can now use: ask --provider gemini \"your question\"")
	return nil
}

// ── Grok Login ──

func runLoginGrok() error {
	// Check if XAI_API_KEY is already set
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey != "" {
		// Test the key
		client := grok.NewClient(apiKey)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if verbose {
			fmt.Fprintf(os.Stderr, "[ask] Verifying XAI_API_KEY...\n")
		}

		models, err := client.ListModels(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ask] Warning: API key validation failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "[ask] Saving anyway — you can retry later.\n")
		} else {
			fmt.Printf("✓ XAI_API_KEY is valid (%d models available)\n", len(models))
		}

		if err := app.SaveGrokSession(apiKey); err != nil {
			return fmt.Errorf("cannot save Grok session: %w", err)
		}

		fmt.Println("  You can now use: ask --provider grok \"your question\"")
		return nil
	}

	// Check for saved session
	session, err := app.LoadGrokSession()
	if err == nil && session != nil && session.APIKey != "" {
		// Test saved key
		client := grok.NewClient(session.APIKey)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		models, err := client.ListModels(ctx)
		if err == nil {
			fmt.Printf("✓ Saved Grok API key is valid (%d models available)\n", len(models))
			fmt.Println("  You can use: ask --provider grok \"your question\"")
			return nil
		}
		fmt.Fprintf(os.Stderr, "[ask] Saved Grok API key is invalid: %v\n", err)
		fmt.Fprintf(os.Stderr, "[ask] Please enter a new API key.\n")
	}

	// Prompt for API key
	fmt.Print("Enter your xAI API key (or press Enter to skip): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read API key: %w", err)
	}

	apiKey = strings.TrimSpace(input)
	if apiKey == "" {
		fmt.Println("Skipped. You can set XAI_API_KEY later.")
		return nil
	}

	// Validate and save
	client := grok.NewClient(apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	models, err := client.ListModels(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ask] Warning: API key validation failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "[ask] Saving anyway — you can retry later.\n")
	} else {
		fmt.Printf("✓ Grok API key is valid (%d models available)\n", len(models))
	}

	if err := app.SaveGrokSession(apiKey); err != nil {
		return fmt.Errorf("cannot save Grok session: %w", err)
	}

	fmt.Println("  You can now use: ask --provider grok \"your question\"")
	return nil
}

// ── Existing ChatGPT login helpers ──

// loginWithChromeProfile extracts the session token from Chrome's cookies DB.
func loginWithChromeProfile() error {
	chromeDataDir := resolveChromeDataDir()

	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Chrome user data dir: %s\n", chromeDataDir)
	}

	if !profile.ProfileExists(chromeDataDir) {
		fmt.Fprintf(os.Stderr, "[ask] Chrome profile not found at %s\n", chromeDataDir)
		fmt.Fprintf(os.Stderr, "[ask] Try 'ask login --web' for browser-based login.\n")
		return nil
	}

	// Read Local State
	localStatePath := profile.LocalStatePath(chromeDataDir)
	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Reading Local State: %s\n", localStatePath)
	}

	ls, err := profile.ReadLocalState(localStatePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ask] Cannot read Chrome Local State: %v\n", err)
		fmt.Fprintf(os.Stderr, "[ask] Try 'ask login --web' for browser-based login.\n")
		return nil
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Decrypting master key (DPAPI)...\n")
	}

	masterKey, err := profile.DecryptMasterKey(ls)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ask] Cannot decrypt master key: %v\n", err)
		fmt.Fprintf(os.Stderr, "[ask] Try 'ask login --web' for browser-based login.\n")
		return nil
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Master key decrypted (%d bytes)\n", len(masterKey))
	}

	// Try all Chrome profiles
	type namedDir struct {
		Name string
		Path string
	}
	candidates := []namedDir{{Name: "Default", Path: filepath.Join(chromeDataDir, "Default")}}
	for i := 1; i <= 10; i++ {
		name := fmt.Sprintf("Profile %d", i)
		path := filepath.Join(chromeDataDir, name)
		if profile.ProfileExists(path) {
			candidates = append(candidates, namedDir{Name: name, Path: path})
		}
	}

	var sessionToken string
	var foundInProfile string
	var lastErr error

	for _, cd := range candidates {
		cookiesPath := profile.CookiesDBPath(cd.Path)

		if _, err := os.Stat(cookiesPath); os.IsNotExist(err) {
			if verbose {
				fmt.Fprintf(os.Stderr, "[ask]   No cookies DB at %s\n", cookiesPath)
			}
			continue
		}

		token, err := profile.ExtractSessionToken(cd.Path, masterKey)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "[ask]   Profile %s: %v\n", cd.Name, err)
			}
			lastErr = err
			continue
		}

		sessionToken = token
		foundInProfile = cd.Name
		break
	}

	if sessionToken == "" {
		fmt.Fprintf(os.Stderr, "[ask] Could not extract ChatGPT session from Chrome profiles.\n")
		if lastErr != nil {
			fmt.Fprintf(os.Stderr, "[ask] Last error: %v\n", lastErr)
		}
		fmt.Fprintf(os.Stderr, "\nPossible solutions:\n")
		fmt.Fprintf(os.Stderr, "  1. Close Chrome completely and run 'ask login' again\n")
		fmt.Fprintf(os.Stderr, "  2. Use 'ask login --web' to paste the token via browser\n")
		fmt.Fprintf(os.Stderr, "  3. Use 'ask login --token <token>' directly\n")
		return nil
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Session token found in profile %s\n", foundInProfile)
	}

	return verifyAndSaveToken(sessionToken, filepath.Join(chromeDataDir, foundInProfile))
}

// loginWithWebServer starts a local HTTP server for token input via browser.
func loginWithWebServer() error {
	server, url, err := chatgpt.StartLoginServer()
	if err != nil {
		return fmt.Errorf("cannot start login server: %w", err)
	}
	defer server.Stop()

	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║   ChatGPT Login Helper                              ║")
	fmt.Println("║                                                    ║")
	fmt.Printf("║   Open this URL in your browser:                     ║\n")
	fmt.Printf("║   %s  ║\n", url)
	fmt.Println("║                                                    ║")
	fmt.Println("║   Steps:                                           ║")
	fmt.Println("║   1. Open Chrome → go to chatgpt.com               ║")
	fmt.Println("║   2. F12 → Application → Cookies → chatgpt.com     ║")
	fmt.Println("║   3. Copy __Secure-next-auth.session-token value    ║")
	fmt.Println("║   4. Paste into the web page and click save         ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	token, verified, err := server.WaitForToken(ctx)
	if err != nil {
		return fmt.Errorf("login timed out or cancelled")
	}

	if !verified {
		fmt.Fprintf(os.Stderr, "[ask] Warning: token could not be verified with ChatGPT API.\n")
		fmt.Fprintf(os.Stderr, "[ask] It may be expired or invalid. Saving anyway.\n")
	}

	return verifyAndSaveToken(token, "manual (web)")
}

// loginWithToken directly saves a provided session token.
func loginWithToken(token string) error {
	return verifyAndSaveToken(token, "manual (--token flag)")
}

// verifyAndSaveToken attempts to verify the token and saves it.
func verifyAndSaveToken(token, profileSource string) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Verifying session with ChatGPT API...\n")
	}

	client := chatgpt.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Authenticate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "[ask] Warning: session token found but API auth failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "[ask] The token may be expired. Still saving — you can retry later.\n")
		// Save anyway — the token might work for API calls even if auth endpoint fails
	}

	if err := app.SaveSessionToken(token, profileSource); err != nil {
		return fmt.Errorf("cannot save session: %w", err)
	}

	fmt.Println("✓ ChatGPT session saved successfully!")
	fmt.Printf("  Source: %s\n", profileSource)
	fmt.Println("  You can now use: ask \"your question\"")

	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Session cached in %s\n", app.SessionTokenPath())
	}

	return nil
}

// resolveChromeDataDir returns the Chrome data directory or empty string.
func resolveChromeDataDir() string {
	if profileDir != "" {
		return profileDir
	}
	dir, err := profile.DefaultChromeDataDir()
	if err != nil {
		return ""
	}
	return dir
}
