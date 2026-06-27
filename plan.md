# plan.md — Go + chromedp / rod 實作計畫

## 1. 實作總方向

本專案改成 **Go native CLI**，核心策略是：

1. 先以 **rod** 完成可用 MVP。
2. 保留 `Browser` 介面，讓 chromedp 可在第二階段接上。
3. 先復刻截圖中的 CLI 使用體驗，再逐步強化 selector、圖片、Markdown 與下載能力。
4. 使用固定 Chrome profile 保存 ChatGPT 登入狀態。
5. 避免儲存密碼、避免繞過登入驗證。

---

## 2. 階段規劃

### Phase 0 — 專案初始化

目標：建立 Go CLI 專案骨架。

工作：

- 建立 `go.mod`
- 建立 `cmd/ask/main.go`
- 加入 Cobra CLI
- 建立 `internal/cli`
- 定義 root command、version、global flags
- 建立 repo 文件

驗收：

```bash
go run ./cmd/ask --help
go run ./cmd/ask --version
```

可看到接近截圖的 help 與 commands。

---

### Phase 1 — Browser interface 與 rod backend

目標：完成可啟動 Chrome 的 backend。

工作：

- 建立 `internal/browser/browser.go`
- 定義 `Browser` interface
- 定義 `BrowserOptions`
- 建立 `rod_backend.go`
- 建立 `Start()` / `Stop()` / `Navigate()`
- 建立 path helper：`~/.ask-cli/chrome-profile`

驗收：

```bash
ask login
```

可以打開 Chrome 並進入 ChatGPT。

---

### Phase 2 — login command

目標：讓使用者手動登入一次後保存 session。

工作：

- `ask login` 強制 headful
- 開啟 ChatGPT 首頁
- 終端機提示：登入完成後按 Enter
- 不讀取、不要求帳密
- 關閉 Chrome 或保持可設定

驗收：

```bash
ask login
ask open --headless=false
```

第二次開啟時應保留登入狀態。

---

### Phase 3 — prompt 傳送與回覆取得

目標：實作 `ask "prompt"`、`ask open "prompt"`、`ask get`。

工作：

- 從 args 讀 prompt
- 若 prompt 為空，從 stdin 讀取
- 找 composer selector
- 填入 prompt
- 按 Enter / 點 send
- 等待 assistant response 穩定
- 取得最後一則 assistant response 的 innerText
- stdout 輸出
- `--output result.md` 寫入檔案

驗收：

```bash
ask "用三點說明 Go 的優點"
cat prompt.txt | ask -o answer.md
ask get
```

可輸出最新回覆。

---

### Phase 4 — dump / screenshot

目標：完成除錯工具。

工作：

- `ask dump`
- 取得目前頁面 HTML
- 預設寫入 `~/.ask-cli/dumps/`
- 可用 `--output` 指定檔案
- `ask screenshot`
- 預設寫入 `~/.ask-cli/screenshots/`
- 可用 `--output` 指定 PNG

驗收：

```bash
ask dump
ask dump -o page.html
ask screenshot
ask screenshot -o page.png
```

會產生可查看檔案。

---

### Phase 5 — `--new` 新對話

目標：支援新 ChatGPT session。

工作：

- 優先嘗試 URL：`https://chatgpt.com/`
- 找 New Chat button
- 點擊新對話
- fallback：直接導航到首頁或新 conversation URL
- 若同時有 prompt，先新對話再送 prompt

驗收：

```bash
ask --new "請用一句話介紹 Rust"
```

應建立新對話並取得回覆。

---

### Phase 6 — 圖片上傳 `--image`

目標：支援一張或多張本機圖片。

工作：

- repeated flag：`--image path`
- 驗證檔案存在
- 驗證副檔名：png/jpg/jpeg/webp/gif
- 找 file input
- 使用 rod `SetFiles()` 上傳
- 等待 preview 出現
- 再送 prompt

驗收：

```bash
ask --image ./input.png "描述這張圖片"
ask --image ./a.png --image ./b.png "比較兩張圖片"
```

ChatGPT 頁面應出現圖片並回覆。

---

### Phase 7 — Markdown 輸出強化

目標：提升 `--output` 品質。

工作：

- 取 assistant message HTML
- 用 goquery 轉 Markdown
- 支援：
  - paragraph
  - heading
  - unordered list
  - ordered list
  - code block
  - inline code
  - table
  - blockquote
- fallback innerText

驗收：

```bash
ask -o result.md "給我一段 Go 程式碼與表格"
```

`result.md` 可保留 code fence 與表格格式。

---

### Phase 8 — 圖片下載 `--image-output`

目標：下載 ChatGPT 回覆中的圖片。

工作：

- 找最新 assistant response 中的 image elements
- 取得 src / blob / download link
- 下載圖片
- 如果 `--image-output` 是資料夾，依序輸出 `image-001.png`
- 如果是檔案且只有一張圖，輸出到指定檔
- 多張圖但指定檔名時，自動加 suffix

驗收：

```bash
ask -i ./images "生成一張黑白線稿圖"
```

圖片下載到指定位置。

---

### Phase 9 — chromedp backend

目標：讓 CLI 可切換 backend。

工作：

- 建立 `chromedp_backend.go`
- 完成與 rod 相同的 Browser interface
- 支援：
  - Start
  - Stop
  - Navigate
  - SendPrompt
  - WaitForResponseDone
  - LatestResponseMarkdown
  - DumpHTML
  - Screenshot
- 暫時可不支援 image upload / image download，標示 unsupported

驗收：

```bash
ask --backend chromedp "請回覆 OK"
ask --backend chromedp screenshot -o page.png
```

可正常執行。

---

### Phase 10 — Release 與文件

目標：交付可下載的 CLI。

工作：

- Makefile
- GitHub Actions build matrix
- Windows/macOS/Linux binary
- README usage
- Troubleshooting
- selector 更新指南

驗收：

```bash
make test
make build
./dist/ask --help
```

---

## 3. 建議里程碑

| Milestone | 內容 | 狀態 |
|---|---|---|
| M1 | CLI skeleton + help + version | 必做 |
| M2 | rod login + profile 保存 | 必做 |
| M3 | prompt send + latest response | 必做 |
| M4 | dump + screenshot | 必做 |
| M5 | stdin + output file | 必做 |
| M6 | --new | 必做 |
| M7 | --image upload | 建議 |
| M8 | --image-output download | 建議 |
| M9 | chromedp backend | 第二階段 |
| M10 | release packaging | 最終 |

---

## 4. MVP 開發順序

最推薦順序：

```txt
Cobra CLI
  -> paths/config
  -> Browser interface
  -> rod Start/Navigate
  -> login
  -> open
  -> SendPrompt
  -> WaitForResponseDone
  -> LatestResponse
  -> output/stdin
  -> dump/screenshot
  -> --new
  -> --image
  -> chromedp
```

不要一開始就同時寫 rod 與 chromedp。先讓 rod 完整可用，再把穩定經驗抽到 interface。

---

## 5. 重要設計決策

### 決策 1：rod 作為預設 backend

原因：

- Go 原生
- 控制 Chrome 方便
- 適合快速 MVP
- 上傳檔案、等待 DOM、截圖實作較直覺

### 決策 2：chromedp 作為可選 backend

原因：

- 社群成熟
- CDP task 模型明確
- 適合長期穩定維護
- 可作為 rod 失效時的替代方案

### 決策 3：固定 profile 保存登入

原因：

- 不處理帳號密碼
- 不處理 2FA
- 不存取敏感認證
- 使用者只需 `ask login` 一次

### 決策 4：selector 集中管理

原因：

ChatGPT 網頁改版時，只需更新 selector 檔案。

### 決策 5：timeout 回傳 partial result

長回答可能超時。建議 timeout 時不要直接丟失內容，而是：

- stderr 顯示 warning
- stdout / output 寫入目前已抓到的回覆
- exit code 可設為 7

---

## 6. 非目標

本專案不做：

- 自動註冊帳號
- 自動輸入密碼
- 自動解 CAPTCHA
- 繞過地區、風控、付費限制
- 模擬 API token
- 長期保證 ChatGPT DOM selector 不變

---

## 7. 交付內容

最終交付應包含：

```txt
ask-cli/
  source code
  README.md
  install.md
  troubleshooting.md
  docs/
    plan.md
    spec.md
    todos.md
    test.md
    final.md
  dist/
    ask-windows-amd64.exe
    ask-linux-amd64
    ask-darwin-arm64
```
