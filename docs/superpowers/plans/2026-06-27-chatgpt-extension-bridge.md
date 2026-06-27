# ChatGPT Extension Bridge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `ask "prompt"` interact with the authenticated ChatGPT page in regular Google Chrome.

**Architecture:** A token-authenticated, one-shot loopback HTTP bridge exchanges one task and result with a Manifest V3 extension. The CLI opens a normal Chrome URL containing ephemeral bridge coordinates in the fragment; the extension performs DOM interaction inside the trusted page.

**Tech Stack:** Go 1.26 standard library HTTP, Cobra, Chrome Manifest V3, vanilla JavaScript

---

### Task 1: One-shot bridge protocol

**Files:**
- Create: `internal/webbridge/server.go`
- Test: `internal/webbridge/server_test.go`

- [ ] Write failing tests for authenticated task retrieval, client claim, result delivery, wrong token, wrong result ID, and context timeout.
- [ ] Run `go test ./internal/webbridge -v` and confirm expected missing-symbol failures.
- [ ] Implement an ephemeral loopback server, random token, trigger URL, and result waiting.
- [ ] Re-run the focused tests and confirm they pass.

### Task 2: Web prompt CLI routing

**Files:**
- Modify: `internal/cli/prompt.go`
- Modify: `internal/cli/root.go`
- Create: `internal/cli/prompt_web.go`
- Test: `internal/cli/prompt_web_test.go`

- [ ] Write failing tests proving `web` dispatch, normal-Chrome URL opening, response output, image rejection, and missing-extension guidance.
- [ ] Run focused CLI tests and confirm expected failures.
- [ ] Implement `runPromptWeb` with injectable bridge/open functions.
- [ ] Make `web` the default while preserving explicit `api`, `rod`, and `chromedp`.
- [ ] Re-run focused tests and confirm they pass.

### Task 3: Chrome extension

**Files:**
- Create: `extension/chatgpt-bridge/manifest.json`
- Create: `extension/chatgpt-bridge/background.js`
- Create: `extension/chatgpt-bridge/content.js`
- Create: `internal/webbridge/extension_test.go`

- [ ] Write a failing Go test for manifest version, permissions, host permissions, service worker, and content-script match.
- [ ] Add the minimal Manifest V3 extension.
- [ ] Implement token handoff, task fetch, DOM prompt submission, stable-response detection, and result POST.
- [ ] Run `node --check` for both JavaScript files and run bridge tests.

### Task 4: One-time extension setup

**Files:**
- Create: `internal/app/extension.go`
- Test: `internal/app/extension_test.go`
- Create: `internal/cli/extension.go`
- Modify: `internal/cli/auth.go`

- [ ] Write failing tests for extension-directory discovery.
- [ ] Implement discovery beside the executable, beside `dist`, from the workspace, and via `ASK_CLI_EXTENSION_DIR`.
- [ ] Add `ask extension` setup instructions and include the path in `ask auth` output.
- [ ] Verify `ask extension --help`.

### Task 5: Documentation and final verification

**Files:**
- Modify: `README.md`
- Modify: `docs/superpowers/specs/2026-06-27-web-authorization-design.md`

- [ ] Document installation, auth-to-prompt flow, backend selection, and troubleshooting.
- [ ] Run `gofmt` on all changed Go files.
- [ ] Run `go test -count=1 ./...`.
- [ ] Run `go vet ./...`.
- [ ] Run `node --check extension/chatgpt-bridge/background.js`.
- [ ] Run `node --check extension/chatgpt-bridge/content.js`.
- [ ] Build `dist/ask.exe` with `-buildvcs=false`.
- [ ] Verify CLI help and normal-Chrome launch behavior.
