# Grok / xAI 操作流程

## 兩種認證模式

| 模式 | 適用場景 | 驗證方式 | API 端點 |
|------|---------|----------|---------|
| **OAuth（推薦）** | SuperGrok / X Premium+ 網頁訂閱 | 瀏覽器 OAuth + PKCE | `cli-chat-proxy.grok.com/v1` |
| **API Key** | xAI 開發者 API | `XAI_API_KEY` 環境變數 | `api.x.ai/v1` |

---

## 第一步：OAuth 登入（推薦）

```powershell
.\ask login grok
```

流程：
1. 自動開啟瀏覽器到 `https://auth.x.ai/oauth2/authorize`
2. 用 X/Twitter 帳號登入
3. 授權 ask-cli 存取你的 SuperGrok / X Premium+ 訂閱
4. 頁面會自動關閉，token 自動存檔

```
╔══════════════════════════════════════════════════════╗
║   Grok / xAI OAuth Authorization                  ║
║                                                    ║
║   Opening browser to auth.x.ai...                   ║
║                                                    ║
║   1. Sign in with your X/Twitter account           ║
║   2. Grant access to ask-cli                        ║
║   3. The page will close automatically             ║
╚══════════════════════════════════════════════════════╝
```

成功後：
```
✓ Grok OAuth completed! (N models available)
  You can now use: ask --provider grok "your question"
```

### 登出 / 重新認證

刪除 token 檔案即可重新登入：
```
del ~\.ask-cli\grok_oauth.json
.\ask login grok
```

---

## 第二步（替代方案）：API Key 模式

如果你有 xAI 開發者 API key：

```powershell
$env:XAI_API_KEY = "xai-你的-api-key"
.\ask --provider grok "你的問題"
```

---

## 第三步：發送問題

### 基本用法

```powershell
.\ask --provider grok "寫一個 Go 的 HTTP server"
```

### 指定模型

```powershell
# OAuth CLI proxy models（SuperGrok / X Premium+）
.\ask --provider grok --model grok-build "你的問題"
.\ask --provider grok --model grok-build-0.1 "你的問題"
.\ask --provider grok --model grok-composer-2.5-fast "你的問題"

# API key models（xAI 開發者 API）
.\ask --provider grok --model grok-3 "你的問題"
.\ask --provider grok --model grok-3-r1 "你的問題"
```

可用的模型：

| 模型 | 模式 | 說明 |
|------|------|------|
| `grok-build` | OAuth（預設） | Grok Build，512K 上下文 |
| `grok-build-0.1` | OAuth | Grok Build v0.1 |
| `grok-composer-2.5-fast` | OAuth | Grok Composer 2.5 Fast |
| `grok-3` | API Key | Grok 3 標準 |
| `grok-3-auto` | API Key（預設） | 自動選擇 |
| `grok-3-r1` | API Key | Grok 3 推理模型 |

### 輸出到檔案

```powershell
.\ask --provider grok -o response.md "你的問題"
```

### 詳細除錯模式

```powershell
.\ask --provider grok -v "你的問題"
```

---

## 完整範例

```powershell
# 1. OAuth 登入
PS D:\ask-cli-go-chromedp-rod-docs\dist> .\ask login grok
╔══════════════════════════════════════════════════════╗
║   Grok / xAI OAuth Authorization                  ║
║                                                    ║
║   Opening browser to auth.x.ai...                   ║
║                                                    ║
║   1. Sign in with your X/Twitter account           ║
║   2. Grant access to ask-cli                        ║
║   3. The page will close automatically             ║
╚══════════════════════════════════════════════════════╝

✓ Grok OAuth completed! (N models available)
  You can now use: ask --provider grok "your question"

# 2. 提問（使用 OAuth + CLI proxy）
PS D:\ask-cli-go-chromedp-rod-docs\dist> .\ask --provider grok "給我一個武則天的線稿圖 三視圖 提示詞"

# 3. 指定 CLI proxy 模型 + 輸出檔案
PS D:\ask-cli-go-chromedp-rod-docs\dist> .\ask --provider grok --model grok-build -o prompt.md `
    "給我一個武則天的線稿圖 三視圖 提示詞"
```

---

## 技術摘要

| 項目 | OAuth 模式 | API Key 模式 |
|------|-----------|-------------|
| API 端點 | `cli-chat-proxy.grok.com/v1` | `api.x.ai/v1` |
| 認證方式 | OAuth 2.0 + PKCE | Bearer Token |
| 所需訂閱 | SuperGrok / X Premium+ | xAI 開發者帳號 |
| OAuth issuer | `auth.x.ai` | N/A |
| 用戶端 ID | `b1a00492-...`（公開） | N/A |
| 特殊 Header | `X-XAI-Token-Auth`, `x-grok-client-version`, `x-grok-client-identifier` | 無 |
| token 存檔 | `~/.ask-cli/grok_oauth.json` | `~/.ask-cli/grok-session.json` |
| 自動 refresh | 是（使用 refresh token） | N/A |

---

## 常見問題

### Q: 需要什麼訂閱？
A: OAuth 模式需要 **SuperGrok** 或 **X Premium+** 訂閱。API Key 模式需要 xAI 開發者帳號。

### Q: OAuth token 過期怎麼辦？
A: 自動 refresh。如果 refresh 失敗，執行 `.\ask login grok` 重新登入。

### Q: 無法開啟瀏覽器？
A: 確保 Chrome 已安裝。OAuth flow 會嘗試用系統預設瀏覽器開啟。

### Q: 出現 rate limit 錯誤？
A: 等待幾分鐘後重試，或改用較長 timeout：
```powershell
.\ask --provider grok --timeout 300 "你的問題"
```

### Q: 如何切換 ChatGPT/Gemini？
A: 使用 `--provider` 旗標：
```powershell
.\ask "chatgpt 問題"                              # 預設 ChatGPT
.\ask --provider gemini "gemini 問題"
.\ask --provider grok "grok 問題"
```
