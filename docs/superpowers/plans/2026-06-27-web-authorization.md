# Native Chrome Web Authorization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Open ChatGPT, Gemini, and Grok authorization pages in the user's regular Google Chrome without browser automation or test profiles.

**Architecture:** Provider metadata supplies official URLs. The platform layer launches the installed Google Chrome with URL-only arguments, while the Cobra command performs validation and reports what was opened.

**Tech Stack:** Go 1.26, Cobra, `os/exec`, standard library tests

---

### Task 1: Reproduce and lock the Cloudflare-safe contract

**Files:**
- Modify: `internal/cli/auth_test.go`
- Create: `internal/platform/chrome_test.go`

- [x] Add a failing test requiring `runWebAuth` to call a native URL opener.
- [x] Add failing tests requiring Chrome arguments to contain only valid HTTPS URLs.
- [x] Assert that automation polling flags are absent from the auth command.

### Task 2: Native Chrome launcher

**Files:**
- Modify: `internal/platform/chrome.go`

- [x] Locate regular Google Chrome, including the Windows user-level install path.
- [x] Start Chrome with official URL arguments only.
- [x] Exclude Chromium fallback and all automation/profile flags.
- [x] Validate URLs before process launch.

### Task 3: Remove automated authorization

**Files:**
- Modify: `internal/cli/auth.go`
- Modify: `internal/webauth/provider.go`
- Modify: `internal/browser/browser.go`
- Modify: `internal/browser/rod_backend.go`
- Modify: `internal/browser/chromedp_backend.go`
- Delete: `internal/webauth/authorize.go`
- Delete: `internal/webauth/authorize_test.go`
- Delete: `internal/browser/session_test.go`

- [x] Replace Rod startup and polling with the native Chrome opener.
- [x] Remove CDP session/cookie probing from authorization.
- [x] Keep provider aliases and official URLs.
- [x] Preserve the existing legacy `ask login` command.

### Task 4: Documentation and verification

**Files:**
- Modify: `README.md`
- Modify: `docs/superpowers/specs/2026-06-27-web-authorization-design.md`

- [x] Document normal Chrome behavior and forbidden automation paths.
- [x] Run `gofmt` on changed Go files.
- [x] Run `go test -count=1 ./...`.
- [x] Run `go vet ./...`.
- [x] Build `dist/ask.exe` with `-buildvcs=false`.
- [x] Verify `dist/ask.exe auth --help` contains no automation polling options.
- [x] Smoke-test `dist/ask.exe auth chatgpt` against the installed `C:\Program Files\Google\Chrome\Application\chrome.exe`.
