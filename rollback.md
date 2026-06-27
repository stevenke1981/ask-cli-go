# Rollback Plan

## Architecture Change: Browser Automation → Chrome Cookie Extraction + HTTP API

### Summary

The main flow was migrated from launching a Chrome browser (via go-rod) to
directly reading ChatGPT session cookies from Chrome's SQLite profile database,
decrypting them with Windows DPAPI + AES-256-GCM, and calling the ChatGPT
backend API via HTTP. The rod browser backend is preserved for `dump` and
`screenshot` commands via `--backend rod`.

### Rollback Triggers

| Trigger | Action | Impact |
|---|---|---|
| DPAPI decryption fails on newer Chrome versions | Fall back to browser-based login | Cookie extraction broken; user uses `--backend rod login` |
| ChatGPT API endpoints change | Update client.go endpoint URLs | Prompt sending broken until updated |
| Cookie DB format changes (Chrome version bump) | Update cookies.go parser | Login broken until updated |
| Windows-specific code blocks cross-platform use | Already handled via `crypto_other.go` stub | Non-Windows platforms must use browser mode |

### Rollback Path: Restore Browser-Only Mode

If the API mode proves unreliable, the project can be rolled back to
browser-only mode with these steps:

**1. Revert `internal/cli/root.go`**

Change default backend back to `"rod"`:

```go
rootCmd.PersistentFlags().StringVar(&backend, "backend", "rod", "Browser backend (rod, chromedp)")
```

Revert help text to original browser-focused description.

**2. Revert `internal/cli/login.go`**

Restore original browser-based login that opens Chrome and waits for manual login.

**3. Revert `internal/cli/prompt.go`**

Remove the API mode path and restore sole dependency on `withBrowser()`.

### Files That Can Be Removed

| File | Purpose | Re-create cost |
|---|---|---|
| `internal/profile/` | Chrome cookie extraction (5 files) | Medium — 2-3 days |
| `internal/chatgpt/client.go` | HTTP API client + SSE parsing | Medium — 1-2 days |
| `internal/app/session.go` | Session token caching | Low — 1 hour |

### Files That Stay Regardless

| File | Purpose |
|---|---|
| `internal/browser/rod_backend.go` | Browser automation for dump/screenshot |
| `internal/browser/browser.go` | Browser interface |
| `internal/app/paths.go` | Directory helpers |
| `internal/app/config.go` | Config with env overrides |

### Reversible Points

| Git commit / state | What was changed | How to revert |
|---|---|---|
| Current state (Phase A1-A3) | Full API mode + profile extraction | `git revert` or manual removal of profile/ package + new client code |

### Test Rollback

```
# Verify browser mode still works after API changes:
go build ./... && go test ./...
go run ./cmd/ask --backend rod --help
```
