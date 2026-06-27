# spec.md — Ask CLI Replica：Go + chromedp / rod 規格書

## 1. 專案目標

建立一個可在終端機使用的 `ask` CLI，復刻截圖中的主要功能：

```txt
ChatGPT CLI - Ask ChatGPT from your Terminal with your subscription

Usage: ask [OPTIONS] [PROMPT] [COMMAND]

Commands:
  open        Open Chrome browser, optionally navigate to a URL, and copy the latest response
  get         Retrieve the latest response from ChatGPT, defaults to headless
  login       Open Chrome browser and wait for manual login
  dump        Dump the current browser tab HTML for debugging
  screenshot  Take a screenshot of the current browser tab for debugging
  help        Print help or subcommand help

Arguments:
  [PROMPT]    The prompt to send to ChatGPT. If empty, reads from standard input

Options:
      --headless[=<HEADLESS>]  Run Chrome in headless mode. Defaults to true
      --new                    Create a brand new ChatGPT session by opening a new tab and closing old ones
  -v, --verbose                Print verbose debugging status messages
  -o, --output <FILE>          Write the final response in Markdown format to the specified file
  -i, --image-output <PATH>    Write downloaded images to the specified folder or file path
      --image <IMAGE_FILE>     Attach one or more local image files to the prompt
  -h, --help                   Print help
  -V, --version                Print version
```

此版本改成 **Go native CLI**，並支援兩種 Chrome DevTools Protocol backend：

1. **rod backend**：建議預設，API 高階、瀏覽器操作簡潔、適合快速實作。
2. **chromedp backend**：可選 backend，較貼近 CDP task model，適合穩定化、細節控制與企業內部維護。

專案不是呼叫官方 OpenAI API，而是透過本機 Chrome profile 控制 ChatGPT 網頁介面。因此它的穩定性會受到 ChatGPT 前端 DOM、登入流程、檔案上傳 UI、回覆渲染方式變動影響。文件會同時保留 `Backend` 抽象層，日後可增加 `openai-api` backend。

---

## 2. 技術選型

### 2.1 語言

- Go 1.22+
- 產出單一可執行檔：`ask.exe` / `ask`
- Windows、macOS、Linux 跨平台

### 2.2 CLI 框架

建議：

- `spf13/cobra`：子命令、help、version、flag 解析
- `spf13/viper`：可選，用於設定檔與環境變數

不想依賴太多套件時，也可以只使用 Go 標準庫 `flag`，但為了復刻截圖中完整 help 格式，建議使用 Cobra。

### 2.3 Browser automation backend

預設使用：

- `github.com/go-rod/rod`
- `github.com/go-rod/rod/lib/launcher`

可選實作：

- `github.com/chromedp/chromedp`
- `github.com/chromedp/cdproto`

### 2.4 HTML / DOM 處理

可選：

- `github.com/PuerkitoBio/goquery`：解析 dump HTML 或回覆區塊
- `regexp` / `strings`：簡單 fallback

### 2.5 Clipboard

可選：

- Windows：PowerShell `Set-Clipboard`
- macOS：`pbcopy`
- Linux：`xclip` / `wl-copy`
- 或 `golang.design/x/clipboard`

### 2.6 檔案與設定路徑

建議預設：

```txt
~/.ask-cli/
  config.toml
  chrome-profile/
  downloads/
  screenshots/
  dumps/
  logs/
```

Windows 範例：

```txt
%USERPROFILE%\.ask-cli\
```

---

## 3. 使用者介面規格

### 3.1 指令總覽

```bash
ask login
ask open
ask open "幫我寫一個 Go HTTP server"
ask get
ask dump
ask screenshot
ask --image ./cat.png "請描述這張圖片"
ask --image ./a.png --image ./b.png -o result.md "分析圖片差異"
ask --image-output ./images "幫我生成一張線稿圖"
ask --headless=false open "打開瀏覽器並送出提示詞"
ask --new "開一個新對話並問問題"
```

### 3.2 全域 options

| Option | 型別 | 預設 | 說明 |
|---|---:|---:|---|
| `--headless[=true|false]` | bool | true | 是否以 headless 模式執行 Chrome |
| `--new` | bool | false | 建立全新 ChatGPT 對話 |
| `-v, --verbose` | bool | false | 顯示詳細 debug log |
| `-o, --output <FILE>` | string | empty | 將最終 Markdown 回覆寫入檔案 |
| `-i, --image-output <PATH>` | string | empty | 下載回覆中的圖片到資料夾或指定檔名 |
| `--image <IMAGE_FILE>` | repeated string | empty | 將本機圖片附加到 prompt |
| `-h, --help` | bool | false | 顯示 help |
| `-V, --version` | bool | false | 顯示版本 |
| `--backend <rod|chromedp>` | string | rod | 本版新增，選擇 browser backend |
| `--profile <DIR>` | string | `~/.ask-cli/chrome-profile` | Chrome user data dir |
| `--timeout <DURATION>` | duration | 180s | 等待回覆完成的最大時間 |

> 截圖中沒有 `--backend`、`--profile`、`--timeout`，但 Go 版本建議加入，方便開發與除錯。

### 3.3 positional prompt

```bash
ask "請幫我整理這段文字"
```

如果 prompt 為空，從 stdin 讀取：

```bash
cat input.txt | ask
```

### 3.4 command 行為

#### `ask login`

用途：開啟 Chrome，讓使用者手動登入 ChatGPT。

流程：

1. 啟動非 headless Chrome。
2. 使用固定 profile directory。
3. 前往 `https://chatgpt.com/`。
4. 等待使用者登入。
5. 使用者按 Enter 後結束。
6. Chrome profile 保留登入 session。

建議：`login` 永遠使用 `headless=false`。

#### `ask open [URL|PROMPT]`

用途：開啟 Chrome browser。可選擇：

- 沒有參數：開 ChatGPT 首頁。
- 參數像 URL：導航到該 URL。
- 參數不是 URL：視為 prompt，送到 ChatGPT，等待回覆，複製最新回覆。

流程：

1. 啟動 Chrome。
2. 如果 `--new`，新建 ChatGPT 對話。
3. 如果有圖片，先 attach。
4. 送出 prompt。
5. 等待回覆完成。
6. 抽取最新 assistant response。
7. 寫入 output / clipboard。
8. 如有 `--image-output`，下載回覆中的圖片。

#### `ask get`

用途：取得目前或最近一次 ChatGPT 頁面的最新回覆。

預設 headless。

流程：

1. 啟動 Chrome，使用同一個 profile。
2. 開 ChatGPT。
3. 找到目前 conversation 中最後一個 assistant message。
4. 抽取 Markdown / plain text。
5. 輸出 stdout，或寫入 `--output`。

#### `ask dump`

用途：輸出目前頁面 HTML，協助除錯 DOM selector。

輸出：

```txt
~/.ask-cli/dumps/chatgpt-YYYYMMDD-HHMMSS.html
```

若提供 `--output`，寫入指定檔案。

#### `ask screenshot`

用途：擷取目前瀏覽器頁面。

輸出：

```txt
~/.ask-cli/screenshots/chatgpt-YYYYMMDD-HHMMSS.png
```

若提供 `--output`，寫入指定檔案。

#### `ask help`

Cobra 自動生成。

---

## 4. 系統架構

```txt
cmd/ask/main.go
        |
        v
internal/cli/
  root.go
  login.go
  open.go
  get.go
  dump.go
  screenshot.go
        |
        v
internal/app/
  runner.go
  config.go
  paths.go
  prompt.go
  output.go
        |
        v
internal/browser/
  browser.go          # interface
  rod_backend.go      # rod implementation
  chromedp_backend.go # chromedp implementation
  selectors.go
  wait.go
  upload.go
  download.go
        |
        v
internal/chatgpt/
  session.go
  composer.go
  response.go
  markdown.go
  images.go
        |
        v
internal/platform/
  clipboard.go
  os.go
  chrome.go
```

---

## 5. Browser interface

所有 CLI command 不直接依賴 rod 或 chromedp，而是依賴共同介面：

```go
type Browser interface {
    Start(ctx context.Context, opts BrowserOptions) error
    Stop(ctx context.Context) error
    Navigate(ctx context.Context, url string) error
    NewChat(ctx context.Context) error
    AttachImages(ctx context.Context, paths []string) error
    SendPrompt(ctx context.Context, prompt string) error
    WaitForResponseDone(ctx context.Context) error
    LatestResponseMarkdown(ctx context.Context) (string, error)
    DumpHTML(ctx context.Context) (string, error)
    Screenshot(ctx context.Context) ([]byte, error)
    DownloadResponseImages(ctx context.Context, outPath string) ([]string, error)
}
```

### 5.1 BrowserOptions

```go
type BrowserOptions struct {
    Headless bool
    NewSession bool
    Verbose bool
    ProfileDir string
    Timeout time.Duration
    BrowserPath string
}
```

---

## 6. DOM selector 策略

因為 ChatGPT 網頁 DOM 可能變動，selector 不應硬寫在各 backend 中，而要集中於 `internal/browser/selectors.go`。

建議 selector 多層 fallback：

```go
type SelectorSet struct {
    ComposerCandidates []string
    SendButtonCandidates []string
    AssistantMessageCandidates []string
    FileInputCandidates []string
    StopGeneratingCandidates []string
    NewChatCandidates []string
}
```

### 6.1 Composer selector fallback

候選：

```txt
textarea
[contenteditable="true"]
#prompt-textarea
[data-testid="prompt-textarea"]
```

### 6.2 Assistant message selector fallback

候選：

```txt
[data-message-author-role="assistant"]
article
.markdown
```

### 6.3 回覆完成判斷

優先策略：

1. 等待「停止生成」按鈕消失。
2. 連續 N 次輪詢最新 assistant text 長度不變。
3. timeout 後回傳 partial result 並標示 warning。

建議：

```txt
poll interval: 500ms
stable count: 4
minimum wait: 2s
max timeout: --timeout，預設 180s
```

---

## 7. rod backend 實作規格

### 7.1 啟動 Chrome

```go
u := launcher.New().
    UserDataDir(profileDir).
    Headless(headless).
    MustLaunch()

browser := rod.New().ControlURL(u).MustConnect()
page := browser.MustPage("https://chatgpt.com/")
```

### 7.2 發送 prompt

rod 建議流程：

1. 找 composer element。
2. focus。
3. input text。
4. press Enter 或點擊 send button。

多行 prompt 要注意：

- 直接設定 contenteditable text 比逐字輸入穩定。
- fallback 使用 clipboard paste。

### 7.3 上傳圖片

流程：

1. 找 `<input type="file">`。
2. 如不可見，點擊 attach button 後再找 input。
3. 使用 rod element `SetFiles()`。
4. 等待圖片 preview 出現。

### 7.4 取得 Markdown

優先使用 DOM innerText。

若需要保留 code block，流程：

1. 取最後一個 assistant message HTML。
2. 用 goquery 解析：
   - `pre code` 保留 fenced code block。
   - `p` 轉段落。
   - `ul/li` 轉 bullet。
   - `ol/li` 轉 numbered list。
   - `table` 轉 Markdown table。
3. fallback 為 innerText。

---

## 8. chromedp backend 實作規格

### 8.1 啟動 Chrome

使用 `ExecAllocator`：

```go
opts := append(chromedp.DefaultExecAllocatorOptions[:],
    chromedp.UserDataDir(profileDir),
    chromedp.Flag("headless", headless),
)
allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
ctx, cancel = chromedp.NewContext(allocCtx)
```

### 8.2 發送 prompt

使用 task chain：

```go
chromedp.Run(ctx,
    chromedp.Navigate("https://chatgpt.com/"),
    chromedp.WaitVisible(selector, chromedp.ByQuery),
    chromedp.Click(selector, chromedp.ByQuery),
    chromedp.SendKeys(selector, prompt, chromedp.ByQuery),
    chromedp.KeyEvent("\r"),
)
```

### 8.3 取回最新回覆

使用 `chromedp.Nodes` 找所有 assistant nodes，再取最後一個：

```go
var nodes []*cdp.Node
chromedp.Nodes(assistantSelector, &nodes, chromedp.ByQueryAll)
```

再用 JS 取 innerText / innerHTML。

---

## 9. Session 與登入規格

### 9.1 Chrome profile

`ask login` 使用：

```txt
~/.ask-cli/chrome-profile
```

所有後續 command 也使用同一 profile。

這可保存：

- ChatGPT login cookies
- localStorage
- indexedDB
- session 狀態

### 9.2 手動登入

不可在 CLI 中要求或儲存使用者帳號密碼。

流程：

```txt
Open browser -> user logs in manually -> user presses Enter in terminal -> done
```

---

## 10. 安全與風險

### 10.1 不儲存密碼

本專案不得提供 `--username` / `--password`。

### 10.2 不繞過驗證

不得自動處理 CAPTCHA、2FA、風控頁面。

### 10.3 Browser UI 不穩定

ChatGPT 前端改版可能導致 selector 失效。

對策：

- selector fallback
- `ask dump`
- `ask screenshot`
- verbose log
- backend abstraction
- test fixture

### 10.4 建議補充 API backend

穩定商用時建議另外提供：

```bash
ask --backend openai-api "prompt"
```

但此需求目前目標是復刻圖片中的 subscription browser CLI，不強制實作 API backend。

---

## 11. 錯誤碼

| Code | 說明 |
|---:|---|
| 0 | 成功 |
| 1 | 一般錯誤 |
| 2 | CLI 參數錯誤 |
| 3 | Chrome 啟動失敗 |
| 4 | 尚未登入 |
| 5 | 找不到 composer |
| 6 | 上傳圖片失敗 |
| 7 | 等待回覆 timeout |
| 8 | 讀取回覆失敗 |
| 9 | 寫入輸出檔失敗 |

---

## 12. MVP 範圍

MVP 必做：

- `ask login`
- `ask "prompt"`
- `ask open "prompt"`
- `ask get`
- `ask dump`
- `ask screenshot`
- `--headless`
- `--new`
- `--output`
- `--verbose`
- rod backend

MVP 可延後：

- chromedp backend
- `--image`
- `--image-output`
- HTML to Markdown 完整轉換
- clipboard 跨平台完整支援
- 自動下載生成圖片

---

## 13. 建議最終 repo 結構

```txt
ask-cli/
  go.mod
  go.sum
  Makefile
  README.md
  LICENSE
  cmd/
    ask/
      main.go
  internal/
    cli/
      root.go
      login.go
      open.go
      get.go
      dump.go
      screenshot.go
    app/
      runner.go
      config.go
      paths.go
      prompt.go
      output.go
    browser/
      browser.go
      selectors.go
      rod_backend.go
      chromedp_backend.go
      wait.go
      upload.go
      download.go
    chatgpt/
      markdown.go
      response.go
      images.go
    platform/
      clipboard.go
      chrome.go
      os.go
  testdata/
    assistant_message.html
    conversation.html
  docs/
    plan.md
    spec.md
    todos.md
    test.md
    final.md
```
