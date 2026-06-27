# final.md — Go + chromedp / rod 最終交付說明

## 1. 本次修改結果

原本規劃的 TypeScript + Playwright 版本已改成：

```txt
Go + rod + chromedp
```

建議實作策略：

```txt
rod 作為預設 backend
chromedp 作為可選 backend
Browser interface 統一抽象
Cobra 實作 CLI
固定 Chrome profile 保存 ChatGPT 登入狀態
```

---

## 2. 為什麼改成 Go

Go 適合這個專案，原因是：

1. 可以編譯成單一 binary。
2. Windows 使用者安裝簡單。
3. CLI 開發成熟。
4. Chrome DevTools Protocol 生態可用。
5. 比 Node.js 更容易交付免 runtime 的工具。
6. 比 Rust 更快完成瀏覽器自動化 MVP。

---

## 3. rod 與 chromedp 分工

### rod

建議作為預設 backend。

用途：

- `ask login`
- `ask open`
- `ask get`
- `ask dump`
- `ask screenshot`
- `--image`
- `--image-output`

優點：

- 寫法直覺
- 操作頁面元素方便
- 檔案上傳比較容易
- 適合快速復刻截圖中的工具

### chromedp

建議作為第二 backend。

用途：

- 文字問答
- dump HTML
- screenshot
- selector 測試
- 長期穩定備援

優點：

- CDP task model 明確
- 社群成熟
- 對底層控制細

---

## 4. 最終功能對照

| 截圖功能 | Go 版本實作方式 | 狀態 |
|---|---|---|
| `ask open` | Cobra command + Browser.Navigate / SendPrompt | 必做 |
| `ask get` | 讀取最新 assistant message | 必做 |
| `ask login` | headful Chrome + profile 保存 session | 必做 |
| `ask dump` | 取得 document HTML | 必做 |
| `ask screenshot` | CDP screenshot | 必做 |
| `ask help` | Cobra auto help | 必做 |
| `[PROMPT]` | args 或 stdin | 必做 |
| `--headless` | rod/chromedp 啟動參數 | 必做 |
| `--new` | NewChat selector / fallback | 必做 |
| `--verbose` | debug log | 必做 |
| `--output` | Markdown 寫檔 | 必做 |
| `--image-output` | 下載回覆圖片 | 建議 |
| `--image` | file input 上傳 | 建議 |
| `--version` | build-time version | 必做 |
| `--backend` | rod/chromedp 切換 | 新增建議 |
| `--profile` | 自訂 Chrome profile | 新增建議 |
| `--timeout` | 控制等待時間 | 新增建議 |

---

## 5. 建議指令體驗

```bash
ask login
ask "請用三點說明 Go 的優點"
ask --new "請開一個新對話並回答 OK"
ask open --headless=false "幫我寫一段 Go 程式"
ask get
ask dump -o debug.html
ask screenshot -o debug.png
ask --image ./photo.png "請描述這張圖片"
ask -i ./images "請生成一張黑白線稿圖"
ask --backend chromedp "請回覆 chromedp ok"
```

---

## 6. 最小可行版本 MVP

MVP 建議只先做這些：

```txt
ask login
ask "prompt"
ask open "prompt"
ask get
ask dump
ask screenshot
--headless
--new
--output
--verbose
rod backend
```

先不要同時做太多圖片功能。圖片上傳與下載受 ChatGPT 前端變動影響最大，建議等文字流程穩定後再做。

---

## 7. 實作重點

### 7.1 不能直接把 selector 寫死在 command 裡

錯誤做法：

```go
page.MustElement("#prompt-textarea")
```

建議做法：

```go
selectors.ComposerCandidates
```

集中管理 selector，方便 ChatGPT 前端改版時修復。

### 7.2 不要儲存帳密

只保存 Chrome profile。

```txt
~/.ask-cli/chrome-profile
```

### 7.3 timeout 要能回傳 partial response

長回答常見，不應 timeout 就完全失敗。建議：

```txt
warning to stderr
partial response to stdout/output
exit code 7
```

### 7.4 dump 與 screenshot 是維護核心

當 ChatGPT UI 改版時，這兩個指令是修 selector 的主要工具：

```bash
ask dump -o page.html
ask screenshot -o page.png
```

---

## 8. 建議 repo 結構

```txt
ask-cli/
  go.mod
  go.sum
  Makefile
  README.md
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

---

## 9. 開發順序建議

```txt
1. Cobra CLI skeleton
2. Path/config
3. Browser interface
4. rod Start/Navigate
5. login
6. open prompt
7. wait response
8. latest response
9. output file
10. get
11. dump
12. screenshot
13. --new
14. --image
15. --image-output
16. chromedp backend
17. release
```

---

## 10. 風險

### 10.1 ChatGPT UI 變動

最大風險是前端 DOM 改版。

對策：

- selector fallback
- dump
- screenshot
- verbose log
- backend interface

### 10.2 登入狀態失效

Session 可能過期。

對策：

- 偵測登入頁
- 提示重新執行 `ask login`

### 10.3 Headless 被限制

有些登入或功能在 headless 下可能不穩。

對策：

- `login` 永遠 headful
- 失敗時提示 `--headless=false`

### 10.4 圖片下載不穩

圖片可能是 blob URL、canvas、或受權限保護。

對策：

- 先做一般 img URL
- 再做 blob fetch
- 最後做點擊下載按鈕 fallback

---

## 11. 最終結論

本專案最適合的實作方式是：

```txt
Go + Cobra + rod-first + chromedp-compatible
```

原因：

- 最接近原截圖中的 CLI 工具體驗
- 可產出單一執行檔
- Windows 交付方便
- 開發速度比 Rust 快
- 比 Node.js 更像 native tool
- rod 適合快速完成
- chromedp 適合長期穩定化

建議先完成 rod MVP，再把 chromedp 補上，不要一開始兩套同時完整實作。
