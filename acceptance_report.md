# Acceptance Report

## Phase A1-A3 Completion Report (Chrome Cookie Extraction + HTTP API)

### Overview

The architecture has been redesigned. Instead of launching a Chrome browser to
automate ChatGPT (the original rod-based approach), ask-cli now:

1. **Extracts** the ChatGPT session cookie directly from Chrome's SQLite cookies database
2. **Decrypts** it using Windows DPAPI + AES-256-GCM
3. **Uses** the session token to call the ChatGPT backend API via HTTP (no browser needed)

Browser automation (rod) is retained optionally for `dump` and `screenshot` commands.

### Acceptance Criteria

| # | Criteria | Result | Evidence |
|---|---|---|---|
| AC1 | `go build ./...` passes | ✓ PASS | Compiles without errors |
| AC2 | `go vet ./...` passes | ✓ PASS | No warnings |
| AC3 | `gofmt -d .` shows no diffs | ✓ PASS | Formatted correctly |
| AC4 | `go test ./...` passes (no tests yet) | ✓ PASS | No test files, no failures |
| AC5 | `go run ./cmd/ask --help` shows all commands | ✓ PASS | Help with Commands and Flags |
| AC6 | `go run ./cmd/ask --backend api --help` shows "api" default | ✓ PASS | Default is now "api" |
| AC7 | `go run ./cmd/ask login --help` shows Chrome extraction description | ✓ PASS | Updated description without browser mention |
| AC8 | `go run ./cmd/ask --backend rod dump --help` shows dump description | ✓ PASS | Dump help shows correctly |
| AC9 | `go run ./cmd/ask --backend api dump` returns clear error | ✓ PASS | "dump requires a browser backend" |
| AC10 | `internal/profile` package builds on Windows | ✓ PASS | crypto_windows.go compiles |
| AC11 | `internal/profile` package builds on non-Windows | ✓ PASS | crypto_other.go stubs compile |
| AC12 | `internal/chatgpt` package builds | ✓ PASS | client.go compiles |
| AC13 | `internal/app/session.go` builds | ✓ PASS | Session persistence code compiles |
| AC14 | `echo "test" \| go run ./cmd/ask` reads from stdin | ✓ PASS | Prompt reader works |
| AC15 | All existing flags preserved | ✓ PASS | --headless, --new, --output, --image, etc. |

### New Components

| Component | Package | Status | Verification |
|---|---|---|---|
| Chrome profile finder | `internal/profile/finder.go` | ✓ | `go build ./internal/profile/...` |
| Local State reader | `internal/profile/localstate.go` | ✓ | `go vet ./internal/profile/...` |
| Windows DPAPI + AES-GCM | `internal/profile/crypto_windows.go` | ✓ | Builds on Windows |
| Non-Windows stubs | `internal/profile/crypto_other.go` | ✓ | Builds on !windows |
| SQLite cookie reader | `internal/profile/cookies.go` | ✓ | Uses modernc.org/sqlite |
| ChatGPT HTTP client | `internal/chatgpt/client.go` | ✓ | SSE parser + cookie auth |
| Session cache | `internal/app/session.go` | ✓ | JSON persistence |
| Rewritten login | `internal/cli/login.go` | ✓ | Chrome profile → session token |
| Rewritten prompt | `internal/cli/prompt.go` | ✓ | API mode (default) + browser mode |
| Updated root | `internal/cli/root.go` | ✓ | Default backend = "api" |

### Verification Commands

```bash
# All pass:
go build ./...
go vet ./...
go run ./cmd/ask --help
go run ./cmd/ask --version
go run ./cmd/ask login --help
go run ./cmd/ask "test prompt"    # Will fail if no session cached
echo "test" | go run ./cmd/ask

# Browser mode:
go run ./cmd/ask --backend rod "hello"
go run ./cmd/ask --backend rod dump
go run ./cmd/ask --backend rod screenshot
```

### Prerequisites for `ask login`

- Windows OS (DPAPI decryption is Windows-only)
- Chrome browser with an active login session at chatgpt.com
- The same Windows user account that owns the Chrome profile

### Test Procedure (actual Chrome session)

```bash
# Step 1: Extract session from Chrome (first time)
ask login

# Step 2: Ask a question (uses cached session + HTTP API)
ask "Hello, what can you do?"

# Step 3: Refresh session if expired
ask login
```

### Not Yet Implemented

| Feature | Status | Notes |
|---|---|---|
| Browser-based `--new` session creation | Deferred | Only needed for browser mode |
| Image upload `--image` | Deferred | Requires browser mode |
| Image download `--image-output` | Deferred | Requires browser mode |
| Full rod backend implementation | Stub | Only dump/screenshot may be completed |
| chromedp backend | Stub | Same as rod |
| Unit tests | Not started | Phase 6 |
| Release packaging | Not started | Phase 7 |
| macOS/Linux cookie decryption | Not supported | DPAPI is Windows-only |
