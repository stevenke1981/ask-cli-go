# ChatGPT、Gemini、Grok 網頁授權設計

## 目標

新增 `ask auth`，把三家服務的官方 HTTPS URL 交給使用者平常使用的 Google Chrome。授權流程不得使用 Playwright、Rod、chromedp、CDP、自動化旗標或獨立測試 profile，避免觸發 Cloudflare 的自動化瀏覽器判定。

## 架構

- `internal/webauth` 只保存 provider 名稱、別名與官方 URL。
- `internal/platform.OpenChromeURLs` 尋找正式安裝的 Google Chrome，並只以 URL 作為程序參數。
- `internal/cli/auth.go` 解析 provider，呼叫平台層開啟頁籤後立即返回。
- 登入、Cloudflare challenge 與 session 保存全部由正常 Chrome 處理。

## 禁止的授權路徑

`ask auth` 不得加入下列旗標或元件：

- `--headless`
- `--remote-debugging-port`
- `--user-data-dir`
- `--enable-automation`
- Playwright、Rod、chromedp 或任何 CDP cookie probe

全域 `--backend` 與 `--profile` 對 `ask auth` 無效，避免意外切回測試瀏覽器。

## CLI

```text
ask auth
ask auth chatgpt
ask auth gemini
ask auth grok
ask auth chatgpt gemini
ask auth all
ask auth openai google xai
```

未提供 provider 時一次開啟 ChatGPT、Gemini、Grok。CLI 不偵測登入完成，使用者直接在正常 Chrome 完成授權。

## 測試

- provider 名稱、別名、排序、去重與錯誤輸入。
- 三個官方 URL 正確傳給原生 Chrome opener。
- Chrome process arguments 只包含 HTTPS URL，並明確排除自動化旗標。
- 全套 `go test ./...`、`go vet ./...`、build 與 CLI help smoke test。

## 安全與範圍

本功能不安裝 MITM CA、不攔截 HTTPS、不讀取 cookie 或 token，也不新增 API proxy。它只負責在正常 Chrome 開啟官方登入頁面。ChatGPT 登入後的 prompt/response 互動由 `chatgpt-extension-bridge-design.md` 定義的本地擴充功能完成，仍不啟動測試瀏覽器。
