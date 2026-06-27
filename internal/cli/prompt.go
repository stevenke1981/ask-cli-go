package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ask-cli/ask-cli/internal/app"
	"github.com/ask-cli/ask-cli/internal/browser"
	"github.com/ask-cli/ask-cli/internal/chatgpt"
	"github.com/ask-cli/ask-cli/internal/gemini"
	"github.com/ask-cli/ask-cli/internal/grok"
	"github.com/ask-cli/ask-cli/internal/grokauth"
)

// runPrompt handles sending a prompt and getting the response.
func runPrompt(args []string) error {
	prompt, err := app.ReadPrompt(args)
	if err != nil {
		return err
	}

	// Ensure data directories exist
	if err := app.EnsureDirs(); err != nil {
		return fmt.Errorf("cannot create data directories: %w", err)
	}

	// Route by provider + backend
	switch strings.ToLower(provider) {
	case "chatgpt":
		return runPromptChatGPT(prompt)
	case "gemini":
		return runPromptGemini(prompt)
	case "grok":
		return runPromptGrok(prompt)
	default:
		return fmt.Errorf("unknown provider %q (valid: chatgpt, gemini, grok)", provider)
	}
}

// runPromptChatGPT routes ChatGPT prompts by backend.
func runPromptChatGPT(prompt string) error {
	switch backend {
	case "web":
		return runPromptWeb(prompt)
	case "api":
		return runPromptAPI(prompt)
	case "rod", "chromedp":
		return runPromptBrowser(prompt)
	default:
		return fmt.Errorf("unknown backend %q for chatgpt (valid: web, api, rod, chromedp)", backend)
	}
}

// runPromptGemini sends a prompt using the Gemini OAuth + API client.
func runPromptGemini(prompt string) error {
	tokens, err := app.LoadGeminiTokens()
	if err != nil {
		return fmt.Errorf("load gemini tokens: %w", err)
	}

	client := gemini.NewClient()

	if tokens != nil {
		client.SetTokens(tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresAt, tokens.ProjectID)
	}

	// Apply --model override
	if modelFlag != "" {
		client.SetModel(modelFlag)
	}

	if !client.IsAuthenticated() {
		return fmt.Errorf("Gemini not authenticated; run 'ask login gemini' first")
	}

	// If no project ID, try to discover it
	if !client.HasProjectID() {
		if verbose {
			fmt.Fprintf(os.Stderr, "[ask] Discovering GCP project...\n")
		}
		discoverCtx, discoverCancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := client.DiscoverProject(discoverCtx); err != nil {
			discoverCancel()
			return fmt.Errorf("Gemini project discovery failed: %w", err)
		}
		discoverCancel()
		// Save the discovered project ID
		if tokens != nil {
			tokens.ProjectID = client.ProjectID()
			app.SaveGeminiTokens(tokens)
		} else {
			app.SaveGeminiTokens(&app.GeminiTokens{
				AccessToken:  client.AccessToken(),
				RefreshToken: "",
				ProjectID:    client.ProjectID(),
			})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Sending prompt to Gemini...\n")
	}

	response, err := client.SendPrompt(ctx, prompt)
	if err != nil {
		return fmt.Errorf("Gemini error: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Response received\n")
	}

	if err := app.WriteOutput(response, outputFile, "gemini"); err != nil {
		return fmt.Errorf("cannot write output: %w", err)
	}
	return nil
}

// runPromptGrok sends a prompt using the Grok/xAI API client.
// Priority: XAI_API_KEY env var → OAuth CLI proxy → saved API key session.
func runPromptGrok(prompt string) error {
	dataDir := app.DefaultBaseDir()

	// 1. Try XAI_API_KEY env var (API key mode)
	if apiKey := os.Getenv("XAI_API_KEY"); apiKey != "" {
		return sendGrokPrompt(prompt, grok.NewClient(apiKey))
	}

	// 2. Try OAuth CLI proxy
	token, err := grokauth.ResolveAccessToken(context.Background(), dataDir)
	if err == nil && token != "" {
		client := grok.NewCLIProxyClient(dataDir)
		client.SetAPIKey(token)
		if modelFlag != "" {
			client.SetModel(modelFlag)
		}
		return sendGrokPrompt(prompt, client)
	}

	// 3. Fall back to saved API key session
	session, err := app.LoadGrokSession()
	if err == nil && session != nil && session.APIKey != "" {
		return sendGrokPrompt(prompt, grok.NewClient(session.APIKey))
	}

	return fmt.Errorf("Grok not configured; run 'ask login grok' to set up OAuth, or set XAI_API_KEY")
}

// sendGrokPrompt is a shared helper for sending prompts to any Grok client.
func sendGrokPrompt(prompt string, client *grok.Client) error {
	// Apply --model override
	if modelFlag != "" {
		client.SetModel(modelFlag)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Sending prompt to Grok...\n")
	}

	response, err := client.SendPrompt(ctx, prompt)
	if err != nil {
		return fmt.Errorf("Grok error: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Response received\n")
	}

	if err := app.WriteOutput(response, outputFile, "grok"); err != nil {
		return fmt.Errorf("cannot write output: %w", err)
	}
	return nil
}

// runPromptAPI sends a prompt using the ChatGPT direct HTTP API.
func runPromptAPI(prompt string) error {
	// Load cached session
	session, err := app.LoadSession()
	if err != nil {
		return fmt.Errorf("cannot load session: %w", err)
	}
	if session == nil || session.SessionToken == "" {
		return fmt.Errorf("no saved ChatGPT session found\n\nRun 'ask login' first to extract your ChatGPT session from Chrome.")
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Using cached session token\n")
	}

	client := chatgpt.NewClient(session.SessionToken)

	// Authenticate (get access token) if needed
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	if session.IsAccessTokenExpired() {
		if verbose {
			fmt.Fprintf(os.Stderr, "[ask] Getting access token...\n")
		}
		if err := client.Authenticate(ctx); err != nil {
			// Session might have expired — try to re-extract from Chrome
			fmt.Fprintf(os.Stderr, "[ask] Session may have expired: %v\n", err)
			fmt.Fprintf(os.Stderr, "[ask] Run 'ask login' to refresh the session.\n")
			return err
		}
		// Update cached session with new access token
		session.AccessToken = ""
		if err := app.SaveSession(session); err != nil && verbose {
			fmt.Fprintf(os.Stderr, "[ask] Warning: could not update session cache: %v\n", err)
		}
	} else if client.IsAuthenticated() {
		if verbose {
			fmt.Fprintf(os.Stderr, "[ask] Using cached access token\n")
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Sending prompt to ChatGPT API...\n")
	}

	// Send the prompt
	response, err := client.SendPrompt(ctx, prompt)
	if err != nil {
		return fmt.Errorf("ChatGPT API error: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[ask] Response received\n")
	}

	// Output the response
	if err := app.WriteOutput(response.Markdown, outputFile, "chatgpt"); err != nil {
		return fmt.Errorf("cannot write output: %w", err)
	}

	return nil
}

// runPromptBrowser sends a prompt using the browser automation backend.
func runPromptBrowser(prompt string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	return withBrowser(ctx, func(b browser.Browser) error {
		if verbose {
			fmt.Fprintf(os.Stderr, "[ask] Navigating to ChatGPT...\n")
		}

		if err := b.Navigate(ctx, "https://chatgpt.com/"); err != nil {
			return fmt.Errorf("cannot navigate to ChatGPT: %w", err)
		}

		if newSession {
			if verbose {
				fmt.Fprintf(os.Stderr, "[ask] Creating new conversation...\n")
			}
			if err := b.NewChat(ctx); err != nil {
				return fmt.Errorf("cannot create new chat: %w", err)
			}
		}

		if len(images) > 0 {
			if verbose {
				fmt.Fprintf(os.Stderr, "[ask] Attaching %d image(s)...\n", len(images))
			}
			if err := b.AttachImages(ctx, images); err != nil {
				return fmt.Errorf("cannot attach images: %w", err)
			}
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "[ask] Sending prompt...\n")
		}
		if err := b.SendPrompt(ctx, prompt); err != nil {
			return fmt.Errorf("cannot send prompt: %w", err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "[ask] Waiting for response...\n")
		}
		if err := b.WaitForResponseDone(ctx); err != nil {
			if err == browser.ErrTimeout {
				fmt.Fprintf(os.Stderr, "[ask] Warning: response timed out, using partial result\n")
			} else {
				return fmt.Errorf("error waiting for response: %w", err)
			}
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "[ask] Retrieving response...\n")
		}
		response, err := b.LatestResponseMarkdown(ctx)
		if err != nil {
			return fmt.Errorf("cannot get response: %w", err)
		}

		if err := app.WriteOutput(response, outputFile, "chatgpt"); err != nil {
			return fmt.Errorf("cannot write output: %w", err)
		}

		return nil
	})
}
