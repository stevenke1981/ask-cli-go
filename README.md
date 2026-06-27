# Ask CLI — Go + chromedp / rod

使用 **web 訂閱額度** 在終端機中與 ChatGPT、Gemini、Grok 對話。

核心選型：

- Go
- Cobra
- rod as default browser backend
- chromedp as optional backend
- regular installed Google Chrome for ChatGPT, Gemini, and Grok web authorization

## 多 Provider 支援

ask-cli 現在支援三個 AI 服務提供者，讓你用 web 訂閱直接在終端機中使用：

| Provider | 授權方式 | 使用指令 |
|----------|---------|---------|
| **ChatGPT** | Chrome extension bridge / Cookie API | `ask "prompt"` (預設) |
| **Gemini** | OAuth 2.0 + PKCE | `ask --provider gemini "prompt"` |
| **Grok** | xAI API Key | `ask --provider grok "prompt"` |

## Web authorization

Use `ask auth` to open the providers' official websites in your regular
installed Google Chrome:

```powershell
# Open ChatGPT, Gemini, and Grok login pages
go run ./cmd/ask auth

# Authorize one or more providers
go run ./cmd/ask auth chatgpt
go run ./cmd/ask auth gemini grok

# Aliases are accepted
go run ./cmd/ask auth openai google xai
```

`ask auth` does not use Playwright, Rod, chromedp, CDP, headless mode, remote
debugging, or a separate `--user-data-dir`. It passes only the official HTTPS
URLs to `chrome.exe`, so Chrome reuses its normal process and user profile.

## 登入 (Authentication)

### ChatGPT

```powershell
# 從 Chrome profile 自動提取 session cookie（Windows）
.\dist\ask.exe login

# 或透過瀏覽器手動複製 token
.\dist\ask.exe login --web

# 或直接提供 token
.\dist\ask.exe login --token <your-session-token>
```

### Gemini (OAuth 2.0 + PKCE)

```powershell
# 啟動 OAuth 授權流程（自動開啟瀏覽器）
.\dist\ask.exe login gemini

# 使用 Google 帳號登入並授權
# Token 會自動保存到 ~/.ask-cli/gemini-tokens.json
```

Gemini 使用 **Google OAuth 2.0 + PKCE**（Proof Key for Code Exchange），
這是最安全的 OAuth 流程。Refresh token 可自動更新 access token，
無需手動重新授權。

### Grok (xAI API)

```powershell
# 設定環境變數
$env:XAI_API_KEY="your-api-key"

# 或使用 login 指令儲存
.\dist\ask.exe login grok
# 會提示輸入 API key
```

## 使用方式

### ChatGPT（預設）

```powershell
# 使用 extension bridge（需先安裝擴充功能）
.\dist\ask.exe "請用三點說明量子糾纏"

# 使用 direct HTTP API
.\dist\ask.exe --backend api "prompt"

# 使用 browser automation
.\dist\ask.exe --backend rod "prompt"
```

### Gemini

```powershell
# 使用 Gemini Advanced 訂閱
.\dist\ask.exe --provider gemini "解釋量子糾纏"

# 指定模型
.\dist\ask.exe --provider gemini --model gemini-2.5-pro "prompt"
```

### Grok

```powershell
# 使用 SuperGrok 訂閱（需 xAI API key）
.\dist\ask.exe --provider grok "解釋量子糾纏"

# 指定模型
.\dist\ask.exe --provider grok --model grok-3 "prompt"
```

## ChatGPT extension bridge setup

Chrome requires one manual extension installation before `ask` can interact
with the authenticated page:

```powershell
.\dist\ask.exe extension
```

This opens `chrome://extensions` and the local
`extension/chatgpt-bridge` folder. In Chrome:

1. Enable **Developer mode**.
2. Click **Load unpacked**.
3. Select the opened `chatgpt-bridge` folder.
4. Keep the extension enabled.

Then authorize and ask:

```powershell
.\dist\ask.exe auth chatgpt
.\dist\ask.exe "請用三點說明量子糾纏"
```

## Backend 切換

| Backend | 說明 | 適用 Provider |
|---------|------|--------------|
| `web` (default) | Chrome extension bridge | ChatGPT |
| `api` | Direct HTTP API | ChatGPT |
| `bridge` | API Bridge server (OpenAI-compatible) | All |
| `rod`/`chromedp` | Browser automation | ChatGPT |

## 架構

```
                     ┌──────────────────────┐
                     │   ask CLI (Cobra)     │
                     │  --provider flag      │
                     └──────┬───────┬───────┘
                            │       │
              ┌─────────────┘       └─────────────┐
              ▼                                     ▼
   ┌────────────────────┐           ┌────────────────────┐
   │   ChatGPT          │           │   Gemini           │
   │   (extension/api)  │           │   (OAuth + PKCE)   │
   └────────────────────┘           └────────────────────┘
                                           │
                              ┌─────────────▼──────────┐
                              │   Grok (xAI API)       │
                              │   OpenAI-compatible    │
                              └────────────────────────┘
```

## 研究文件

- `web-ai-api-bridge-research.md`：三家服務（ChatGPT、Gemini、Grok）的
  認證架構、Token 管理、API 協定轉換層的完整研究報告。
- `internal/apibridge/`：根據研究報告 §5/§8 實作的 OpenAI-compatible API Bridge。
- `internal/gemini/`：根據研究報告 §6.3 實作的 Gemini OAuth + PKCE 客戶端。
- `internal/grok/`：根據研究報告 §3.2 實作的 Grok/xAI 客戶端。

## 文件

- `spec.md`：完整規格
- `plan.md`：階段計畫
- `todos.md`：實作任務清單
- `test.md`：測試計畫
- `final.md`：最終交付說明
