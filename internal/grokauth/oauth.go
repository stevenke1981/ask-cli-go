package grokauth

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// ── Public API ──

// LoginOptions configures the Grok OAuth login flow.
type LoginOptions struct {
	DataDir      string
	ClientID     string
	Scope        string
	RedirectURI  string
	Timeout      time.Duration
	NoBrowser    bool
	RedirectPort int
	Out          io.Writer
	In           io.Reader
}

// LoginResult holds the outcome of a successful login.
type LoginResult struct {
	Tokens      Tokens
	RedirectURI string
}

// Login runs the Grok OAuth 2.0 + PKCE login flow.
// It opens a browser to auth.x.ai, starts a local callback server,
// exchanges the authorization code for tokens, and persists them.
func Login(ctx context.Context, opts LoginOptions) (LoginResult, error) {
	clientID := strings.TrimSpace(opts.ClientID)
	if clientID == "" {
		clientID = DefaultClientID
	}
	scope := strings.TrimSpace(opts.Scope)
	if scope == "" {
		scope = DefaultScope
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	verifier, challenge, err := pkcePair()
	if err != nil {
		return LoginResult{}, fmt.Errorf("pkce: %w", err)
	}
	state, err := randomURLSafe(24)
	if err != nil {
		return LoginResult{}, fmt.Errorf("state: %w", err)
	}

	redirectURI := strings.TrimSpace(opts.RedirectURI)
	if redirectURI == "" {
		redirectURI = DefaultRedirectURI
	}

	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	in := opts.In
	if in == nil {
		in = os.Stdin
	}

	// Start callback listener on ephemeral port
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ln, resultCh, err := startEphemeralCallbackListener(waitCtx, state, opts.RedirectPort)
	if err != nil {
		return LoginResult{}, fmt.Errorf("callback listener: %w", err)
	}
	usedRedirectURI := ln.redirectURI

	oauthCfg := oauth2Config(clientID, usedRedirectURI, scope)
	authURL := oauthCfg.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	// Open browser
	if !opts.NoBrowser {
		if err := openBrowser(authURL); err != nil && opts.Out != nil {
			fmt.Fprintln(out, "Could not open browser automatically.")
		}
	}

	// Show instructions — support both automatic redirect and manual paste
	fmt.Fprintln(out, "╔════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(out, "║   Grok / xAI OAuth Authorization                         ║")
	fmt.Fprintln(out, "║                                                          ║")
	fmt.Fprintln(out, "║   If a browser didn't open, paste this URL manually:     ║")
	fmt.Fprintf(out, "║   %-56s ║\n", authURL)
	fmt.Fprintln(out, "║                                                          ║")
	fmt.Fprintln(out, "║   Steps:                                                 ║")
	fmt.Fprintln(out, "║   1. Sign in with your X/Twitter account                 ║")
	fmt.Fprintln(out, "║   2. Grant access to ask-cli                              ║")
	fmt.Fprintln(out, "║                                                          ║")
	fmt.Fprintln(out, "║   The page will redirect back automatically, OR          ║")
	fmt.Fprintln(out, "║   if you see a code — paste it below and press Enter.    ║")
	fmt.Fprintln(out, "╚════════════════════════════════════════════════════════════╝")

	// Wait for callback OR manual paste
	type codeOrErr struct {
		code string
		err  error
	}

	manualCh := make(chan codeOrErr, 1)
	go func() {
		manualCode, err := readLine(in)
		if err != nil {
			manualCh <- codeOrErr{err: fmt.Errorf("read input: %w", err)}
			return
		}
		manualCh <- codeOrErr{code: strings.TrimSpace(manualCode)}
	}()

	var authCode string
	select {
	case <-waitCtx.Done():
		return LoginResult{}, fmt.Errorf("oauth login timed out: %w", waitCtx.Err())
	case res := <-resultCh:
		if res.err != nil {
			return LoginResult{}, res.err
		}
		if res.state != state {
			return LoginResult{}, errors.New("oauth state mismatch")
		}
		authCode = res.code
	case m := <-manualCh:
		if m.err != nil {
			return LoginResult{}, m.err
		}
		if m.code == "" {
			return LoginResult{}, errors.New("no authorization code provided")
		}
		authCode = m.code
	}
	if authCode == "" {
		return LoginResult{}, errors.New("oauth returned empty authorization code")
	}

	// Exchange code for tokens
	oauthCfg = oauth2Config(clientID, usedRedirectURI, scope)
	token, err := oauthCfg.Exchange(ctx, authCode, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		return LoginResult{}, fmt.Errorf("token exchange: %w", err)
	}

	tokens := tokensFromOAuth2(token)
	if err := SaveOAuthTokens(opts.DataDir, tokens); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Tokens: tokens, RedirectURI: usedRedirectURI}, nil
}

// RefreshTokens refreshes the Grok OAuth access token using the stored refresh token.
func RefreshTokens(ctx context.Context, dataDir, clientID string) (Tokens, error) {
	if strings.TrimSpace(clientID) == "" {
		clientID = DefaultClientID
	}
	tok, err := LoadOAuthTokens(dataDir)
	if err != nil {
		return Tokens{}, err
	}
	rt := strings.TrimSpace(tok.RefreshToken)
	if rt == "" {
		return Tokens{}, errors.New("no grok refresh token available; run 'ask login grok' first")
	}

	redirectURI := DefaultRedirectURI
	oauthCfg := oauth2Config(clientID, redirectURI, DefaultScope)
	src := oauthCfg.TokenSource(ctx, &oauth2.Token{RefreshToken: rt})
	newTok, err := src.Token()
	if err != nil {
		return Tokens{}, fmt.Errorf("grok token refresh failed: %w", err)
	}
	out := mergeTokenResponse(tok, tokensFromOAuth2(newTok))
	if err := SaveOAuthTokens(dataDir, out); err != nil {
		return Tokens{}, err
	}
	return out, nil
}

// ── Internal ──

type callbackResult struct {
	code  string
	state string
	err   error
}

type ephemeralListener struct {
	redirectURI string
	port        int
}

func startEphemeralCallbackListener(ctx context.Context, wantState string, preferredPort int) (*ephemeralListener, <-chan callbackResult, error) {
	port := preferredPort
	if port <= 0 {
		port = 8080
	}

	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		// Try random port
		listener, err = net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			return nil, nil, fmt.Errorf("cannot listen on loopback: %w", err)
		}
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port
	callbackPath := "/callback"
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d%s", actualPort, callbackPath)

	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Missing authorization code.")
			resultCh <- callbackResult{err: errors.New("callback missing code")}
			return
		}
		if state != wantState {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "State mismatch.")
			resultCh <- callbackResult{err: errors.New("oauth state mismatch")}
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Authentication successful! You can close this window.")
		resultCh <- callbackResult{code: code, state: state}
	})

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		_ = server.Serve(listener)
	}()
	go func() {
		<-ctx.Done()
		server.Close()
	}()

	return &ephemeralListener{redirectURI: redirectURI, port: actualPort}, resultCh, nil
}

func oauth2Config(clientID, redirectURI, scope string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:    clientID,
		Endpoint:    oauth2.Endpoint{AuthURL: AuthorizeURL, TokenURL: TokenURL},
		RedirectURL: redirectURI,
		Scopes:      strings.Fields(scope),
	}
}

func tokensFromOAuth2(token *oauth2.Token) Tokens {
	out := Tokens{AuthMode: "oauth"}
	if v := strings.TrimSpace(token.AccessToken); v != "" {
		out.AccessToken = v
	}
	if v := strings.TrimSpace(token.RefreshToken); v != "" {
		out.RefreshToken = v
	}
	if !token.Expiry.IsZero() {
		out.ExpiresAt = token.Expiry
	}
	return out
}

func mergeTokenResponse(prev, next Tokens) Tokens {
	out := prev
	if v := strings.TrimSpace(next.AccessToken); v != "" {
		out.AccessToken = v
	}
	if v := strings.TrimSpace(next.RefreshToken); v != "" {
		out.RefreshToken = v
	}
	if !next.ExpiresAt.IsZero() {
		out.ExpiresAt = next.ExpiresAt
	}
	if out.AuthMode == "" {
		out.AuthMode = next.AuthMode
	}
	if out.AuthMode == "" {
		out.AuthMode = "oauth"
	}
	return out
}

func pkcePair() (verifier, challenge string, err error) {
	verifier, err = randomURLSafe(48)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomURLSafe(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// ResolveAccessToken resolves a Grok API access token from OAuth tokens.
// It tries the cache first, then refreshes if needed.
func ResolveAccessToken(ctx context.Context, dataDir string) (string, error) {
	const skew = 2 * time.Minute

	tok, err := LoadOAuthTokens(dataDir)
	if err != nil {
		return "", err
	}
	if access, valid := tok.ValidAccessToken(skew); valid {
		return access, nil
	}
	if strings.TrimSpace(tok.RefreshToken) != "" {
		refreshed, rerr := RefreshTokens(ctx, dataDir, DefaultClientID)
		if rerr == nil {
			if access, valid := refreshed.ValidAccessToken(skew); valid {
				return access, nil
			}
		}
	}
	return "", errors.New("no valid grok token; run 'ask login grok'")
}

// readLine reads a line of text from the given io.Reader (e.g. stdin).
// It strips trailing newline characters and returns the trimmed string.
func readLine(r io.Reader) (string, error) {
	// Use bufio.Scanner with a bufio.Reader fallback
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return scanner.Text(), nil
}

// CLIProxyHeaders returns headers required by cli-chat-proxy.grok.com.
func CLIProxyHeaders(model string) map[string]string {
	if model == "" {
		model = "grok-build"
	}
	return map[string]string{
		"X-XAI-Token-Auth":         CLITokenAuthHeader,
		"x-grok-model-override":    model,
		"x-grok-client-version":    DefaultCLIProxyClientVersion,
		"x-grok-client-identifier": CLIClientIdentifier,
	}
}
