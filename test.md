# test.md — Go + chromedp / rod 測試計畫

## 1. 測試目標

確保 `ask` CLI 可穩定復刻截圖中的功能：

- login
- open
- get
- dump
- screenshot
- prompt from args
- prompt from stdin
- output Markdown
- image upload
- image output
- headless/headful
- new session
- rod backend
- chromedp backend

測試分成四層：

1. Unit tests：不啟動 Chrome。
2. Integration tests：測 CLI、檔案、fixture。
3. Browser integration tests：啟動 Chrome，但不一定需要登入。
4. E2E tests：真的登入 ChatGPT 後測完整流程，預設關閉。

---

## 2. 測試指令

```bash
go test ./...
go test ./... -race
go test ./... -run TestMarkdown
ASK_E2E=1 go test ./... -run TestE2E
```

建議 Makefile：

```makefile
fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

race:
	go test ./... -race

build:
	go build -o dist/ask ./cmd/ask
```

---

## 3. Unit tests

### 3.1 Path tests

檔案：`internal/app/paths_test.go`

測試：

- `DefaultBaseDir()` 不為空
- `DefaultProfileDir()` 指向 base dir 下的 `chrome-profile`
- Windows path 不使用硬寫 `/`
- 可建立 dumps/screenshots/downloads 目錄

驗收：

```bash
go test ./internal/app -run TestPaths
```

---

### 3.2 Prompt reader tests

檔案：`internal/app/prompt_test.go`

案例：

| Case | Input | Expected |
|---|---|---|
| args prompt | `ask hello` | `hello` |
| multi args | `ask hello world` | `hello world` |
| stdin | `echo hi | ask` | `hi` |
| multiline stdin | file content | 保留換行 |
| empty | no args no stdin | error |

---

### 3.3 Output writer tests

檔案：`internal/app/output_test.go`

案例：

- stdout output
- write `result.md`
- parent dir auto create
- invalid path returns error
- UTF-8 content preserved

---

### 3.4 Markdown converter tests

檔案：`internal/chatgpt/markdown_test.go`

使用 fixture：

```txt
testdata/assistant_message.html
testdata/assistant_message_code.html
testdata/assistant_message_table.html
```

案例：

- paragraph 轉換
- unordered list 轉換
- ordered list 轉換
- code block 轉換
- inline code 轉換
- table 轉換
- nested elements fallback

---

### 3.5 Selector fallback tests

檔案：`internal/browser/selectors_test.go`

案例：

- composer candidates 順序正確
- assistant candidates 順序正確
- file input candidates 包含 `input[type=file]`
- empty selector set returns error

---

## 4. CLI integration tests

### 4.1 Help output

測試：

```bash
go run ./cmd/ask -h
```

預期包含：

```txt
Usage:
Commands:
open
get
login
dump
screenshot
--headless
--new
--image-output
--image
```

### 4.2 Version output

```bash
go run ./cmd/ask --version
```

預期：

```txt
ask version x.y.z
```

### 4.3 Invalid command

```bash
go run ./cmd/ask unknown
```

預期：

- exit code != 0
- stderr 有 unknown command

---

## 5. Browser integration tests

這層會啟動 Chrome，但不要求 ChatGPT 登入。

### 5.1 rod 啟動測試

```bash
ASK_BROWSER_TEST=1 go test ./internal/browser -run TestRodStartStop
```

測試：

- headless 啟動成功
- navigate 到 static test page
- dump html 包含 fixture text
- screenshot bytes 是 PNG

### 5.2 chromedp 啟動測試

```bash
ASK_BROWSER_TEST=1 go test ./internal/browser -run TestChromedpStartStop
```

測試：

- headless 啟動成功
- navigate 到 static test page
- dump html 成功
- screenshot 成功

### 5.3 selector 測試頁

建立 local fixture：

```html
<div data-message-author-role="assistant">
  <div class="markdown">Hello from assistant</div>
</div>
<div id="prompt-textarea" contenteditable="true"></div>
<button data-testid="send-button">Send</button>
```

驗收：

- backend 能找到 composer
- backend 能取得 latest assistant text

---

## 6. E2E tests

E2E 需要使用者已執行：

```bash
ask login
```

並手動確認 ChatGPT 已登入。

E2E 預設不在 CI 執行，必須明確設定：

```bash
ASK_E2E=1 go test ./e2e -timeout 5m
```

### 6.1 login smoke test

手動測試：

```bash
ask login
```

驗收：

- Chrome 開啟
- 可以手動登入
- 按 Enter 後 CLI 結束
- profile 目錄存在

---

### 6.2 文字 prompt 測試

```bash
ask --new "請只回覆：ASK_E2E_OK"
```

驗收：

- stdout 包含 `ASK_E2E_OK`
- exit code 0

---

### 6.3 output file 測試

```bash
ask --new -o /tmp/ask-answer.md "請用 Markdown 列出三點"
```

驗收：

- 檔案存在
- 檔案非空
- 檔案包含 Markdown list

---

### 6.4 stdin 測試

```bash
echo "請只回覆 STDIN_OK" | ask --new
```

驗收：

- stdout 包含 `STDIN_OK`

---

### 6.5 dump 測試

```bash
ask dump -o /tmp/chatgpt.html
```

驗收：

- HTML 檔存在
- 檔案包含 `<html`
- 檔案大小大於 1KB

---

### 6.6 screenshot 測試

```bash
ask screenshot -o /tmp/chatgpt.png
```

驗收：

- PNG 檔存在
- 前 8 bytes 為 PNG magic bytes
- 檔案大小大於 1KB

---

### 6.7 image upload 測試

```bash
ask --new --image testdata/sample.png "請描述這張圖片，並只用一句話回答"
```

驗收：

- 不報上傳錯誤
- 有 assistant 回覆

---

### 6.8 image output 測試

```bash
ask --new -i /tmp/ask-images "請生成一張簡單黑白線稿圖"
```

驗收：

- `/tmp/ask-images` 有圖片檔
- 圖片副檔名正確
- 檔案大小大於 1KB

此測試可能受帳號功能、模型功能、前端權限影響，因此標為 optional。

---

## 7. Headless/headful 測試

### 7.1 headless true

```bash
ask --headless=true --new "請回覆 HEADLESS_OK"
```

預期：

- 不開可見視窗
- stdout 包含 `HEADLESS_OK`

### 7.2 headless false

```bash
ask --headless=false --new "請回覆 HEADFUL_OK"
```

預期：

- 開啟可見 Chrome
- stdout 包含 `HEADFUL_OK`

---

## 8. Backend parity tests

目標：rod 與 chromedp 至少在基本功能上等價。

```bash
ask --backend rod --new "請只回覆 BACKEND_OK"
ask --backend chromedp --new "請只回覆 BACKEND_OK"
```

驗收：

- 兩者 stdout 都包含 `BACKEND_OK`
- dump 都可用
- screenshot 都可用

可允許差異：

- image upload 初期只有 rod 支援
- image download 初期只有 rod 支援

---

## 9. 錯誤測試

### 9.1 未登入

使用全新 profile：

```bash
ask --profile /tmp/ask-empty-profile get
```

預期：

- exit code 4
- stderr 提示先執行 `ask login`

### 9.2 圖片不存在

```bash
ask --image missing.png "描述圖片"
```

預期：

- exit code 6 或 2
- stderr 顯示 file not found

### 9.3 timeout

```bash
ask --timeout 1s "請寫一篇非常長的文章"
```

預期：

- exit code 7 或 warning
- 如果已有 partial response，應輸出 partial

### 9.4 output 不可寫

```bash
ask -o /root/no-permission.md "test"
```

預期：

- exit code 9
- stderr 顯示 permission denied

---

## 10. Windows 測試

PowerShell：

```powershell
.\ask.exe --help
.\ask.exe login
.\ask.exe --new "請回覆 WINDOWS_OK"
.\ask.exe screenshot -o .\debug.png
.\ask.exe dump -o .\debug.html
Get-Content .\prompt.txt | .\ask.exe -o .\answer.md
```

需要確認：

- path separator 正確
- 中文 prompt 不亂碼
- UTF-8 output 正確
- Chrome profile 可寫入
- PowerShell pipe 可用

---

## 11. macOS 測試

```bash
./ask --help
./ask login
./ask --new "請回覆 MAC_OK"
./ask screenshot -o debug.png
```

需要確認：

- Chrome path 自動偵測
- headless 可用
- clipboard fallback 可用

---

## 12. Linux 測試

```bash
./ask --help
./ask login
./ask --new "請回覆 LINUX_OK"
./ask screenshot -o debug.png
```

需要確認：

- 有安裝 Chrome / Chromium
- headless 環境可用
- CI 需要 xvfb 時可執行

---

## 13. CI 建議

CI 不跑真 ChatGPT E2E，只跑：

- go fmt check
- go vet
- go test ./...
- fixture browser tests if chromium available
- build matrix

GitHub Actions matrix：

```yaml
os:
  - ubuntu-latest
  - windows-latest
  - macos-latest
go:
  - '1.22'
  - '1.23'
```

---

## 14. 測試完成標準

可視為完成的最低標準：

- [ ] `go test ./...` 通過
- [ ] Windows 手動 E2E 通過
- [ ] `ask login` 可保存 session
- [ ] `ask --new "請回覆 OK"` 可回覆
- [ ] `ask get` 可取得最新回覆
- [ ] `ask dump` 可輸出 HTML
- [ ] `ask screenshot` 可輸出 PNG
- [ ] `--output` 可寫 Markdown
- [ ] rod backend 可用
- [ ] chromedp backend 可執行基本文字流程
