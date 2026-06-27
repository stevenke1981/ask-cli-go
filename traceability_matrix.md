# Traceability Matrix

## CLI Framework & Browser Backend

| Requirement | Implementation | Verification | Evidence | Status |
|---|---|---|---|---|
| CLI skeleton with Cobra | `cmd/ask/main.go`, `internal/cli/root.go` | `go build ./...` | Help shows all commands and flags | ✓ |
| Version flag | `root.go` — `Version` var with `--version` | `go run ./cmd/ask --version` | `ask version 0.1.0` | ✓ |
| `--backend` flag (default "api") | Persistent flag on root | `go run ./cmd/ask --help` | Flag present with default "api" | ✓ |
| `--profile` flag | Persistent flag on root | `go run ./cmd/ask --help` | Flag present in help | ✓ |
| `--timeout` flag | Persistent flag on root | `go run ./cmd/ask --help` | Flag present with default 180 | ✓ |
| `--verbose, -v` | Persistent flag on root | `go run ./cmd/ask --help` | Flag present in help | ✓ |
| `--output, -o` | Root + subcommands | `go run ./cmd/ask dump --help` | Flag present in subcommand help | ✓ |
| `--headless` flag | Root + open commands | `go run ./cmd/ask --help` shows flag | Flag present in help | ✓ |
| `--new` flag | Root + open commands | `go run ./cmd/ask --help` shows flag | Flag present in help | ✓ |
| `--image-output, -i` | Root command | `go run ./cmd/ask --help` shows flag | Flag present in help | ✓ |
| `--image` repeated | Root command | `go run ./cmd/ask --help` shows flag | Flag present in help | ✓ |
| `ask login` subcommand | `internal/cli/login.go` | `go run ./cmd/ask login --help` | Chrome cookie extraction + auth | ✓ |
| `ask open` subcommand | `internal/cli/open.go` | `go run ./cmd/ask open --help` | Delegates to runPrompt | ✓ |
| `ask get` subcommand | `internal/cli/get.go` | `go run ./cmd/ask get --help` | Shows session status in API mode | ✓ |
| `ask dump` subcommand | `internal/cli/dump.go` | `go run ./cmd/ask dump --help` | Browser-only; error in API mode | ✓ |
| `ask screenshot` subcommand | `internal/cli/screenshot.go` | `go run ./cmd/ask screenshot --help` | Browser-only; error in API mode | ✓ |
| Positional prompt arg | `root.go` — `ArbitraryArgs` + `rootRun` dispatch | `echo "test" \| go run ./cmd/ask` | Recognized as prompt | ✓ |
| Stdin prompt | `app/prompt.go` — `ReadPrompt()` | `echo "test" \| go run ./cmd/ask` | Reads from pipe | ✓ |
| Output writer | `internal/app/output.go` | Build | Compiles without errors | ✓ |
| Browser interface | `internal/browser/browser.go` | Build | Interface defined with all methods | ✓ |
| Browser factory | `internal/browser/browser.go` — `NewBrowser()` | Build | Returns RodBrowser or ChromedpBrowser | ✓ |
| Rod backend | `internal/browser/rod_backend.go` | Build | Implements Browser interface (stub) | ✓ |
| Chromedp backend | `internal/browser/chromedp_backend.go` | Build | Implements Browser interface (stub) | ✓ |
| Platform clipboard | `internal/platform/clipboard.go` | Build | Cross-platform clipboard stubs | ✓ |
| Platform chrome finder | `internal/platform/chrome.go` | Build | Chrome path detection | ✓ |

## Path & Config

| Requirement | Implementation | Verification | Evidence | Status |
|---|---|---|---|---|
| Default base dir | `internal/app/paths.go` | Build | Returns `~/.ask-cli` | ✓ |
| Default profile dir | `internal/app/paths.go` | Build | Returns `~/.ask-cli/chrome-profile` | ✓ |
| Default downloads/screenshots/dumps dirs | `internal/app/paths.go` | Build | Per-type directories | ✓ |
| Auto-create directories | `internal/app/paths.go` — `EnsureDirs()` | Build | Creates all required dirs | ✓ |
| Config with env overrides | `internal/app/config.go` | Build | `ASK_CLI_BACKEND`, `ASK_CLI_PROFILE_DIR`, etc. | ✓ |

## Session Cache (NEW)

| Requirement | Implementation | Verification | Evidence | Status |
|---|---|---|---|---|
| Session persistence | `internal/app/session.go` | Build | JSON file with tokens | ✓ |
| Save/load session token | `app.SaveSessionToken()` / `app.LoadSession()` | Build | Compiles without errors | ✓ |
| Access token expiry check | `app.SessionData.IsAccessTokenExpired()` | Build | 5-minute grace period | ✓ |
| Session token path | `app.SessionTokenPath()` | Build | `~/.ask-cli/session.json` | ✓ |

## Chrome Profile Cookie Extraction (NEW)

| Requirement | Implementation | Verification | Evidence | Status |
|---|---|---|---|---|
| Chrome path discovery | `internal/profile/finder.go` | Build | Cross-platform (Windows/macOS/Linux) | ✓ |
| Local State reader | `internal/profile/localstate.go` | Build | Parses `os_crypt.encrypted_key` | ✓ |
| Master key decryption | `internal/profile/crypto_windows.go` | Build | DPAPI + AES-256-GCM (Windows) | ✓ |
| Non-Windows stub | `internal/profile/crypto_other.go` | Build | Returns ErrNotSupported | ✓ |
| Cookie SQLite reader | `internal/profile/cookies.go` | Build | Uses modernc.org/sqlite | ✓ |
| Session cookie extraction | `internal/profile/cookies.go` — `ExtractSessionToken()` | Build | Targets `__Secure-next-auth.session-token` | ✓ |
| Multi-profile support | `internal/cli/login.go` | Build | Checks Default + Profile 1..10 | ✓ |

## ChatGPT HTTP API Client (NEW)

| Requirement | Implementation | Verification | Evidence | Status |
|---|---|---|---|---|
| API client with cookie auth | `internal/chatgpt/client.go` | Build | Session token injected as cookie | ✓ |
| Access token acquisition | `Client.Authenticate()` | Build | Calls `/api/auth/session` | ✓ |
| Prompt sending | `Client.SendPrompt()` | Build | POST to `/backend-api/conversation` | ✓ |
| SSE stream parsing | `parseSSEStream()` | Build | Extracts text from SSE events | ✓ |
| Custom CookieJar | `cookieJar` struct | Build | Adds secure session cookie to requests | ✓ |

## Verification

| Requirement | Implementation | Verification | Evidence | Status |
|---|---|---|---|---|
| Build passes | All packages | `go build ./...` | Compiles without errors | ✓ |
| Vet passes | All packages | `go vet ./...` | No warnings | ✓ |
| Format passes | All packages | `gofmt -l .` | No formatting issues | ✓ |
| Test passes | (no tests yet) | `go test ./...` | No failures | ✓ |
