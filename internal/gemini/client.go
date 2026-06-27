// Package gemini provides authenticated access to the Gemini web API
// using Google OAuth 2.0 + PKCE, enabling CLI usage of Gemini Advanced
// subscriptions.
//
// Architecture (from web-ai-api-bridge-research.md §6.3):
//  1. OAuth 2.0 + PKCE flow via Google Identity
//  2. Token exchange (auth code → access token + refresh token)
//  3. API calls through Gemini web endpoints
//  4. Automatic token refresh
package gemini

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	AuthEndpoint  = "https://accounts.google.com/o/oauth2/v2/auth"
	TokenEndpoint = "https://oauth2.googleapis.com/token"
	RedirectURI   = "http://localhost:8085/oauth2callback"

	// Gemini internal API base (Cloud Code Assist).
	// Uses v1internal (not v1alpha or v1beta) — this is the internal endpoint
	// that the official Gemini CLI uses. Model name goes in the request body,
	// not in the URL path.
	APIBase = "https://cloudcode-pa.googleapis.com/v1internal"

	defaultModel = "gemini-2.5-flash"
)

var defaultScopes = []string{
	"openid",
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
}

// ── Types ──

// TokenResponse is the OAuth token exchange response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token,omitempty"`
}

// TokenError is the OAuth token error response.
type TokenError struct {
	ErrorType        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// AuthCodeResponse is the redirect callback that contains the auth code.
type AuthCodeResponse struct {
	Code  string
	State string
	Error string
}

// Client handles authenticated requests to the Gemini web API.
type Client struct {
	mu           sync.RWMutex
	httpClient   *http.Client
	clientID     string
	clientSecret string
	accessToken  string
	refreshToken string
	expiresAt    time.Time
	projectID    string
	apiBase      string

	// PKCE state
	codeVerifier string

	// Model
	model string
}

// ClientIDFromEnv returns the Google OAuth client ID from environment variables.
// It checks GEMINI_CLIENT_ID, then GOOGLE_OAUTH_CLIENT_ID.
func ClientIDFromEnv() string {
	if v := os.Getenv("GEMINI_CLIENT_ID"); v != "" {
		return v
	}
	return os.Getenv("GOOGLE_OAUTH_CLIENT_ID")
}

// ClientSecretFromEnv returns the Google OAuth client secret from environment variables.
// It checks GEMINI_CLIENT_SECRET, then GOOGLE_OAUTH_CLIENT_SECRET.
func ClientSecretFromEnv() string {
	if v := os.Getenv("GEMINI_CLIENT_SECRET"); v != "" {
		return v
	}
	return os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")
}

// NewClient creates a Gemini API client.
// If the environment does not provide GEMINI_CLIENT_ID / GEMINI_CLIENT_SECRET
// (or the legacy GOOGLE_OAUTH_* names), OAuth flows will fail with a clear
// message pointing the user to the Gemini CLI source.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    5,
				IdleConnTimeout: 90 * time.Second,
			},
		},
		clientID:     ClientIDFromEnv(),
		clientSecret: ClientSecretFromEnv(),
		model:        defaultModel,
		apiBase:      APIBase,
	}
}

// SetModel overrides the default model.
func (c *Client) SetModel(model string) {
	c.model = model
}

// ── OAuth 2.0 + PKCE ──

// PKCEChallenge generates a code verifier and its S256 challenge.
func PKCEChallenge() (verifier string, challenge string, err error) {
	// Random 32-byte verifier, base64url-encoded (no padding)
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate code verifier: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(raw)

	// S256 challenge
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])
	return verifier, challenge, nil
}

// AuthURL returns the Google OAuth authorization URL with PKCE.
func (c *Client) AuthURL() (string, string, error) {
	if c.clientID == "" {
		return "", "", fmt.Errorf("Gemini OAuth client ID not configured.\n\n" +
			"Set the GEMINI_CLIENT_ID and GEMINI_CLIENT_SECRET environment variables.\n" +
			"See: https://github.com/google-gemini/gemini-cli/blob/main/packages/core/src/code_assist/oauth2.ts")
	}

	codeVerifier, codeChallenge, err := PKCEChallenge()
	if err != nil {
		return "", "", err
	}
	c.codeVerifier = codeVerifier

	v := url.Values{
		"client_id":             {c.clientID},
		"redirect_uri":          {RedirectURI},
		"response_type":         {"code"},
		"scope":                 {strings.Join(defaultScopes, " ")},
		"access_type":           {"offline"},
		"prompt":                {"consent"},
		"code_challenge_method": {"S256"},
		"code_challenge":        {codeChallenge},
	}
	authURL := AuthEndpoint + "?" + v.Encode()
	return authURL, codeVerifier, nil
}

// ExchangeCode exchanges an authorization code for tokens.
func (c *Client) ExchangeCode(ctx context.Context, code string) error {
	data := url.Values{
		"code":          {code},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"redirect_uri":  {RedirectURI},
		"grant_type":    {"authorization_code"},
		"code_verifier": {c.codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse structured error
		var terr TokenError
		if json.Unmarshal(body, &terr) == nil && terr.ErrorType != "" {
			return fmt.Errorf("token exchange failed (HTTP %d): %s: %s", resp.StatusCode, terr.ErrorType, terr.ErrorDescription)
		}
		return fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return fmt.Errorf("parse token response: %w", err)
	}

	c.mu.Lock()
	c.accessToken = token.AccessToken
	c.refreshToken = token.RefreshToken
	c.expiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	c.mu.Unlock()
	return nil
}

// RefreshToken uses the refresh token to get a new access token.
func (c *Client) RefreshToken(ctx context.Context) error {
	c.mu.RLock()
	refreshToken := c.refreshToken
	c.mu.RUnlock()

	if refreshToken == "" {
		return fmt.Errorf("no refresh token available; re-authenticate")
	}

	data := url.Values{
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse structured error
		var terr TokenError
		if json.Unmarshal(body, &terr) == nil && terr.ErrorType != "" {
			return fmt.Errorf("token refresh failed (HTTP %d): %s: %s", resp.StatusCode, terr.ErrorType, terr.ErrorDescription)
		}
		return fmt.Errorf("token refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return fmt.Errorf("parse refresh response: %w", err)
	}

	c.mu.Lock()
	c.accessToken = token.AccessToken
	if token.RefreshToken != "" {
		c.refreshToken = token.RefreshToken // rotate refresh token if provided
	}
	c.expiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	c.mu.Unlock()
	return nil
}

// IsAuthenticated returns true if we have a valid (or refreshable) token.
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken != ""
}

// HasRefreshToken returns true if a refresh token is available.
func (c *Client) HasRefreshToken() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.refreshToken != ""
}

// IsAccessTokenExpired returns true if the access token needs refresh.
func (c *Client) IsAccessTokenExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Now().After(c.expiresAt.Add(-5 * time.Minute))
}

// AccessToken returns the current access token (for persistence).
func (c *Client) AccessToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken
}

// SetTokens sets tokens and project from persisted data (restore session).
func (c *Client) SetTokens(accessToken, refreshToken string, expiresAt time.Time, projectID ...string) {
	c.mu.Lock()
	c.accessToken = accessToken
	c.refreshToken = refreshToken
	c.expiresAt = expiresAt
	if len(projectID) > 0 {
		c.projectID = projectID[0]
	}
	c.mu.Unlock()
}

// SetProjectID sets the project ID for the Cloud Code Assist API.
func (c *Client) SetProjectID(projectID string) {
	c.mu.Lock()
	c.projectID = projectID
	c.mu.Unlock()
}

// ProjectID returns the current project ID.
func (c *Client) ProjectID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.projectID
}

// HasProjectID returns true if a project ID is available.
func (c *Client) HasProjectID() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.projectID != ""
}

// ── Project Discovery ──
//
// The Cloud Code Assist API requires a Google Cloud project ID.  The Gemini
// CLI discovers it via the loadCodeAssist endpoint.  If no project exists yet,
// onboardUser creates one automatically.

// LoadCodeAssistRequest is the payload for /v1internal:loadCodeAssist.
type LoadCodeAssistRequest struct {
	CloudAicompanionProject string            `json:"cloudaicompanionProject,omitempty"`
	Metadata                ClientMetadataMsg `json:"metadata"`
}

// ClientMetadataMsg mirrors the Gemini CLI ClientMetadata.
type ClientMetadataMsg struct {
	IDEType    string `json:"ideType,omitempty"`
	Platform   string `json:"platform,omitempty"`
	PluginType string `json:"pluginType,omitempty"`
}

// LoadCodeAssistResponse mirrors the Gemini CLI response.
type LoadCodeAssistResponse struct {
	CurrentTier             *TierInfo  `json:"currentTier,omitempty"`
	AllowedTiers            []TierInfo `json:"allowedTiers,omitempty"`
	IneligibleTiers         []TierInfo `json:"ineligibleTiers,omitempty"`
	CloudAicompanionProject string     `json:"cloudaicompanionProject,omitempty"`
}

// TierInfo represents a Gemini user tier.
type TierInfo struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	IsDefault bool   `json:"isDefault,omitempty"`
}

// OnboardUserRequest is the payload for /v1internal:onboardUser.
type OnboardUserRequest struct {
	TierID                  string            `json:"tierId,omitempty"`
	CloudAicompanionProject string            `json:"cloudaicompanionProject,omitempty"`
	Metadata                ClientMetadataMsg `json:"metadata,omitempty"`
}

// LongRunningOperation represents a Google LRO.
type LongRunningOperation struct {
	Name     string               `json:"name,omitempty"`
	Done     bool                 `json:"done"`
	Response *OnboardUserResponse `json:"response,omitempty"`
}

// OnboardUserResponse contains the resulting project.
type OnboardUserResponse struct {
	CloudAicompanionProject *ProjectRef `json:"cloudaicompanionProject,omitempty"`
}

// ProjectRef references a cloud project.
type ProjectRef struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

func clientMetadata() ClientMetadataMsg {
	return ClientMetadataMsg{
		IDEType:    "GEMINI_CLI",
		Platform:   "PLATFORM_UNSPECIFIED",
		PluginType: "GEMINI",
	}
}

// doPostJSON is a helper for the internal API POST calls.
func (c *Client) doPostJSON(ctx context.Context, method string, payload any) ([]byte, error) {
	c.mu.RLock()
	accessToken := c.accessToken
	c.mu.RUnlock()

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal %s: %w", method, err)
	}

	urlStr := c.apiBase + ":" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create %s request: %w", method, err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s request failed: %w", method, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", method, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s failed (HTTP %d): %s", method, resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// DiscoverProject calls the loadCodeAssist API to discover the user's Google
// Cloud project.  If the user has not onboarded, it triggers onboardUser to
// create one automatically.  The discovered project ID is stored on the client.
func (c *Client) DiscoverProject(ctx context.Context) error {
	c.mu.RLock()
	project := c.projectID
	c.mu.RUnlock()

	// First try loadCodeAssist
	loadReq := LoadCodeAssistRequest{
		CloudAicompanionProject: project,
		Metadata:                clientMetadata(),
	}

	raw, err := c.doPostJSON(ctx, "loadCodeAssist", loadReq)
	if err != nil {
		return fmt.Errorf("load code assist: %w", err)
	}

	var loadResp LoadCodeAssistResponse
	if err := json.Unmarshal(raw, &loadResp); err != nil {
		return fmt.Errorf("parse loadCodeAssist response: %w", err)
	}

	// If we already have a project from the response, use it
	if loadResp.CloudAicompanionProject != "" {
		c.mu.Lock()
		c.projectID = loadResp.CloudAicompanionProject
		c.mu.Unlock()
		return nil
	}

	// If we had a project ID going in but the response didn't return one, proceed with what we have
	if project != "" {
		return nil
	}

	// No project yet — onboard the user to create one.
	// Use STANDARD tier (same as gemini-cli default)
	onboardReq := OnboardUserRequest{
		TierID:                  "standard-tier",
		CloudAicompanionProject: "",
		Metadata:                clientMetadata(),
	}

	raw, err = c.doPostJSON(ctx, "onboardUser", onboardReq)
	if err != nil {
		return fmt.Errorf("onboard user: %w", err)
	}

	var lro LongRunningOperation
	if err := json.Unmarshal(raw, &lro); err != nil {
		return fmt.Errorf("parse onboardUser response: %w", err)
	}

	// If it's an LRO, poll until done
	if !lro.Done && lro.Name != "" {
		for {
			time.Sleep(3 * time.Second)
			pollRaw, pollErr := c.doPostJSON(ctx, "getOperation", map[string]string{"name": lro.Name})
			if pollErr != nil {
				break // can't poll further — proceed with whatever we have
			}
			var polled LongRunningOperation
			if pollErr = json.Unmarshal(pollRaw, &polled); pollErr != nil {
				break
			}
			lro = polled
			if lro.Done {
				break
			}
		}
	}

	// Check LRO result for project ID
	if lro.Response != nil && lro.Response.CloudAicompanionProject != nil && lro.Response.CloudAicompanionProject.ID != "" {
		c.mu.Lock()
		c.projectID = lro.Response.CloudAicompanionProject.ID
		c.mu.Unlock()
		return nil
	}

	// Last resort: check GOOGLE_CLOUD_PROJECT env var
	if envProject := os.Getenv("GOOGLE_CLOUD_PROJECT"); envProject != "" {
		c.mu.Lock()
		c.projectID = envProject
		c.mu.Unlock()
		return nil
	}

	return fmt.Errorf("could not discover Google Cloud project; set GOOGLE_CLOUD_PROJECT env var")
}

// ── Gemini Cloud Code Assist Internal API ──
//
// The Gemini CLI uses the internal Cloud Code Assist API at
// cloudcode-pa.googleapis.com/v1internal.  The request body is a wrapper
// (CAGenerateContentRequest) that carries the model name, an optional project,
// and a "request" envelope matching Vertex AI format.
//
// Source: https://github.com/google-gemini/gemini-cli/blob/main/packages/core/src/code_assist/converter.ts

// CAGenerateContentRequest is the envelope the internal API expects.
type CAGenerateContentRequest struct {
	Model              string                   `json:"model"`
	Project            string                   `json:"project,omitempty"`
	UserPromptID       string                   `json:"user_prompt_id,omitempty"`
	Request            VertexGenerateContentReq `json:"request"`
	EnabledCreditTypes []string                 `json:"enabled_credit_types,omitempty"`
}

// VertexGenerateContentReq mirrors the public Gemini GenerateContentParameters
// but nested under "request".
type VertexGenerateContentReq struct {
	Contents          []VertexContent `json:"contents"`
	SystemInstruction *VertexContent  `json:"systemInstruction,omitempty"`
}

// VertexContent is a single conversation turn.
type VertexContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []VertexPart `json:"parts"`
}

// VertexPart is a single part of content.
type VertexPart struct {
	Text string `json:"text"`
}

// CAGenerateContentResponse is the internal API response envelope.
type CAGenerateContentResponse struct {
	Response         *VertexGenerateContentResp `json:"response,omitempty"`
	TraceID          string                     `json:"traceId,omitempty"`
	ConsumedCredits  []CreditEntry              `json:"consumedCredits,omitempty"`
	RemainingCredits []CreditEntry              `json:"remainingCredits,omitempty"`
}

// VertexGenerateContentResp holds the actual response candidates.
type VertexGenerateContentResp struct {
	Candidates    []VertexCandidate `json:"candidates,omitempty"`
	UsageMetadata *UsageMetadata    `json:"usageMetadata,omitempty"`
	ModelVersion  string            `json:"modelVersion,omitempty"`
}

// VertexCandidate is one response candidate.
type VertexCandidate struct {
	Content       VertexContent  `json:"content"`
	FinishReason  string         `json:"finishReason,omitempty"`
	SafetyRatings []SafetyRating `json:"safetyRatings,omitempty"`
}

// SafetyRating represents a safety rating.
type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// UsageMetadata holds token counts.
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount,omitempty"`
	CandidatesTokenCount int `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount      int `json:"totalTokenCount,omitempty"`
}

// CreditEntry tracks credit consumption.
type CreditEntry struct {
	CreditType   string `json:"creditType"`
	CreditAmount string `json:"creditAmount,omitempty"`
}

// randomID generates a hex string for user_prompt_id.
func randomID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// SendPrompt sends a prompt to Gemini and returns the response text.
func (c *Client) SendPrompt(ctx context.Context, prompt string) (string, error) {
	if !c.IsAuthenticated() {
		return "", fmt.Errorf("gemini not authenticated; run 'ask auth gemini' first")
	}

	// Ensure token is valid
	if c.IsAccessTokenExpired() {
		if c.HasRefreshToken() {
			if err := c.RefreshToken(ctx); err != nil {
				return "", fmt.Errorf("refresh gemini token: %w", err)
			}
		} else {
			return "", fmt.Errorf("gemini access token expired; re-authenticate with 'ask auth gemini'")
		}
	}

	c.mu.RLock()
	model := c.model
	accessToken := c.accessToken
	project := c.projectID
	apiBase := c.apiBase
	c.mu.RUnlock()

	if project == "" {
		return "", fmt.Errorf("no Google Cloud project ID; run 'ask login gemini' or set GOOGLE_CLOUD_PROJECT")
	}

	// Build the internal-API request envelope.
	caReq := CAGenerateContentRequest{
		Model:        model,
		Project:      project,
		UserPromptID: randomID(),
		Request: VertexGenerateContentReq{
			Contents: []VertexContent{
				{
					Role: "user",
					Parts: []VertexPart{
						{Text: prompt},
					},
				},
			},
		},
	}

	body, err := json.Marshal(caReq)
	if err != nil {
		return "", fmt.Errorf("marshal gemini request: %w", err)
	}

	// POST https://cloudcode-pa.googleapis.com/v1internal:generateContent
	// Model is in the body, NOT in the URL path.
	apiURL := apiBase + ":generateContent"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create gemini request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read gemini response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var caResp CAGenerateContentResponse
	if err := json.Unmarshal(respBody, &caResp); err != nil {
		return "", fmt.Errorf("parse gemini response: %w", err)
	}

	// The response is nested under "response".
	if caResp.Response == nil {
		return "", fmt.Errorf("gemini returned empty response envelope")
	}

	if len(caResp.Response.Candidates) == 0 {
		return "", fmt.Errorf("gemini returned no candidates")
	}

	// Extract text from first candidate
	var text strings.Builder
	for _, part := range caResp.Response.Candidates[0].Content.Parts {
		text.WriteString(part.Text)
	}
	result := strings.TrimSpace(text.String())
	if result == "" {
		return "", fmt.Errorf("gemini returned empty response")
	}

	return result, nil
}
