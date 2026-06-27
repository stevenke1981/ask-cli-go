// Package grokauth provides xAI Grok OAuth authentication and token management
// for cli-chat-proxy.grok.com (SuperGrok / X Premium+ web subscription).
//
// Architecture:
//
//	OAuth 2.0 + PKCE  →  auth.x.ai  →  OAuth tokens
//	         ↓
//	cli-chat-proxy.grok.com/v1  (OpenAI-compatible, with special headers)
//
// The OAuth client ID is the same public PKCE client used by the official Grok CLI.
package grokauth

const (
	// IssuerURL is the xAI OAuth issuer.
	IssuerURL = "https://auth.x.ai"

	// AuthorizeURL is the OAuth authorization endpoint.
	AuthorizeURL = IssuerURL + "/oauth2/authorize"

	// TokenURL is the OAuth token endpoint.
	TokenURL = IssuerURL + "/oauth2/token"

	// DefaultClientID is the public Grok CLI OAuth client (PKCE, no secret).
	// Same as the official Grok Build CLI.
	DefaultClientID = "b1a00492-073a-47ea-816f-4c329264a828"

	// DefaultScope is the OAuth scope for SuperGrok / X Premium+ access.
	DefaultScope = "openid profile email offline_access grok-cli:access api:access"

	// CLIProxyAccessScope is required for cli-chat-proxy.grok.com.
	CLIProxyAccessScope = "api:access"

	// CLIProxyBaseURL is the Grok CLI proxy endpoint.
	CLIProxyBaseURL = "https://cli-chat-proxy.grok.com/v1"

	// CLITokenAuthHeader is the value for X-XAI-Token-Auth header.
	CLITokenAuthHeader = "xai-grok-cli"

	// CLIClientIdentifier matches the official Grok shell.
	CLIClientIdentifier = "grok-shell"

	// DefaultRedirectURI is the loopback callback URI.
	DefaultRedirectURI = "http://localhost:8080/callback"

	// DefaultCLIProxyClientVersion is used when no override or install info is available.
	DefaultCLIProxyClientVersion = "0.2.22"
)
