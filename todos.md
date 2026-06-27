# todos.md — ask-cli 任務清單

## v2 Architecture: Chrome Cookie Extraction + Direct HTTP API

The main flow no longer requires browser automation. ChatGPT session cookies are
extracted directly from the Chrome browser profile, decrypted using Windows DPAPI
+ AES-256-GCM, and then used to call the ChatGPT backend API via HTTP.

Browser automation (rod) is retained only for `dump` and `screenshot` commands.

---

## Phase 0: 專案初始化 (Completed)

- [x] 建立 repo：`ask-cli`
- [x] 初始化 Go module：`go mod init github.com/ask-cli/ask-cli`
- [x] 建立 `cmd/ask/main.go`
- [x] 建立 `internal/cli`
- [x] 建立 `internal/app`
- [x] 建立 `internal/browser`
- [x] 建立 `internal/chatgpt`
- [x] 建立 `internal/platform`
- [x] 加入 `.gitignore`
- [x] 加入 `Makefile`

---

## Phase 1: CLI skeleton (Completed)

- [x] 加入 Cobra
- [x] 建立 root command：`ask`
- [x] 設定 short description
- [x] 加入 `--headless`
- [x] 加入 `--new`
- [x] 加入 `--verbose`
- [x] 加入 `--output, -o`
- [x] 加入 `--image-output, -i`
- [x] 加入 repeated `--image`
- [x] 加入 `--backend` (default: `"api"`)
- [x] 加入 `--profile`
- [x] 加入 `--timeout`
- [x] 加入 `--version, -V`
- [x] 調整 help 排版

---

## Phase 2: Path 與 config (Completed)

- [x] 實作 `DefaultBaseDir()`
- [x] 實作 `DefaultProfileDir()`
- [x] 實作 `DefaultDownloadsDir()`
- [x] 實作 `DefaultScreenshotsDir()`
- [x] 實作 `DefaultDumpsDir()`
- [x] 啟動時自動建立資料夾
- [x] 支援 `ASK_CLI_PROFILE_DIR` 環境變數
- [x] 支援 `ASK_CLI_BACKEND` 環境變數
- [x] 支援 `ASK_CLI_BROWSER_PATH` 環境變數

---

## Phase 3: Browser interface (Completed, kept for dump/screenshot only)

- [x] 建立 `Browser` interface
- [x] 建立 `BrowserOptions`
- [x] 建立 `NewBrowser(backend string)` factory
- [x] 定義所有 error types

---

## Phase 4: rod backend (Stub only — no browser launch needed for main flow)

- [x] 加入 rod dependency
- [x] 實作 `RodBrowser` struct (stub)
- [x] 實作所有 Browser interface methods (stub)
- [x] 支援自訂 Chrome path
- [x] 支援 headless / headful

---

## Phase A1: Chrome Profile Cookie Extraction (Completed)

- [x] 建立 `internal/profile` package
- [x] `profile.go` — 共用 types 與 errors
- [x] `finder.go` — Chrome profile 路徑發現 (跨平台)
- [x] `localstate.go` — 讀取 Chrome Local State JSON
- [x] `crypto_windows.go` — Windows DPAPI + AES-256-GCM 解密
- [x] `crypto_other.go` — 非 Windows 平台的 stub
- [x] `cookies.go` — SQLite cookies DB 讀取與解密
- [x] 支援多個 Chrome profile (`Default`, `Profile 1`, ...)
- [x] 目標 cookie: `__Secure-next-auth.session-token` for `chatgpt.com`
- [x] build + vet 通過

---

## Phase A2: ChatGPT HTTP API Client (Completed)

- [x] 建立 `internal/chatgpt/client.go`
- [x] `Client` struct with session cookie auth
- [x] `Authenticate()` — 使用 session token 取得 access token
- [x] `SendPrompt()` — 透過 `/backend-api/conversation` 發送 prompt
- [x] SSE stream 解析
- [x] 客製 CookieJar 自動附加 session cookie
- [x] build + vet 通過

---

## Phase A3: 會話快取與 CLI 整合 (Completed)

- [x] `internal/app/session.go` — Session 快取 (JSON 檔案)
- [x] `SaveSession()` / `LoadSession()` / `DeleteSession()`
- [x] `SaveSessionToken()` convenience function
- [x] `IsAccessTokenExpired()` check
- [x] `login.go` — 重寫為 Chrome cookie 提取 + auth 驗證
- [x] `prompt.go` — 支援 API mode (`--backend api`)
- [x] `root.go` — 預設 backend 改為 `"api"`
- [x] `get.go` — API mode 顯示會話狀態
- [x] `dump.go` / `screenshot.go` — API mode 回傳清楚錯誤
- [x] build + vet 通過

---

## Phase 5b: Browser mode (optional — rod / chromedp)

Browser mode remains available via `--backend rod` for:
- `ask --backend rod "prompt"` — browser automation
- `ask --backend rod dump` — page HTML dump
- `ask --backend rod screenshot` — page screenshot

Remaining items (deferred unless browser mode is needed):

- [ ] 實作 `FindFirstVisible()`（需 rod 套件）
- [ ] 實作完整的 `SendPrompt()`（需 rod 套件）
- [ ] 實作 `WaitForResponseDone()`（需 rod 套件）
- [ ] 實作 `LatestResponseMarkdown()`（需 rod 套件）
- [ ] 實作 `NewChat()`（需 rod 套件）
- [ ] 實作 `AttachImages()`（需 rod 套件）
- [ ] 實作完整的 `DumpHTML()`（需 rod 套件）
- [ ] 實作 `Screenshot()`（需 rod 套件）

---

## Phase 6: 測試

- [ ] unit tests：path
- [ ] unit tests：prompt reader
- [ ] unit tests：session 快取
- [ ] unit tests：profile cookie extraction（需要 mock）
- [ ] unit tests：output writer
- [ ] integration tests：CLI help
- [ ] integration tests：login flow (requires Chrome)
- [ ] Windows 測試

---

## Phase 7: Release

- [x] `make fmt`
- [x] `make vet`
- [x] `make test`
- [x] `make build`
- [ ] 建立 GitHub Actions
- [ ] 產出 Windows amd64 binary
- [ ] 產出 Linux amd64 binary
- [ ] 產出 macOS arm64 binary

---

## Phase 8: 完成定義

專案完成時需符合：

- [x] CLI help 完整
- [x] `ask login` 可從 Chrome 提取 session token（Windows 限定）
- [x] `ask login` 可驗證會話
- [x] `ask "prompt"` 可直接透過 API 取得回覆
- [x] `ask get` 顯示會話狀態
- [ ] `ask --backend rod dump` 可 dump HTML
- [ ] `ask --backend rod screenshot` 可截圖
- [ ] `ask --backend rod "prompt"` 可用 browser 模式
- [ ] 文件完整
- [ ] 測試通過
