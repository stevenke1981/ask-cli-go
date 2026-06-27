# 網頁版 AI 服務 API 橋接研究報告

## Web AI Service API Bridge via Localhost Proxy — 技術研究與實作指南

---

**文件狀態**: v1.0 | 2026-06-26
**研究範圍**: ChatGPT (OpenAI) · Grok (xAI) · Gemini (Google)
**研究目的**: 探討透過虛擬本地主機（localhost proxy）攔截、提取、轉發網頁版 AI 服務的 API 介面，使原生應用程式得以使用這些服務的技術方法與架構設計。

---

## 目錄

1. [摘要](#1-摘要)
2. [研究背景與動機](#2-研究背景與動機)
3. [三大服務認證架構分析](#3-三大服務認證架構分析)
4. [核心技術棧](#4-核心技術棧)
5. [系統架構設計](#5-系統架構設計)
6. [實作方法論](#6-實作方法論)
7. [Token 生命週期管理](#7-token-生命週期管理)
8. [API 協定轉換層](#8-api-協定轉換層)
9. [安全考量](#9-安全考量)
10. [法律與倫理邊界](#10-法律與倫理邊界)
11. [開源生態系分析](#11-開源生態系分析)
12. [結論與建議](#12-結論與建議)
13. [參考資料](#13-參考資料)

---

## 1. 摘要

本報告深入研究如何透過**本地代理伺服器（localhost proxy）** 技術，攔截網頁版 AI 服務（ChatGPT、Grok、Gemini）的認證流程與 API 呼叫，提取 session token / access token，將其轉換為標準 OpenAI-compatible API 格式，供第三方應用程式使用。研究重點包括：

- 三大服務的 OAuth / Session 認證架構逆向分析
- MITM Proxy 技術棧（mitmproxy、Charles、自訂代理）
- Token 提取、刷新、失效處理的生命週期管理
- OpenAI-compatible 協定轉換層設計
- 開源專案生態系調查（gemini-proxy、grok-bypass、litellm 等）
- 法律風險評估與防護建議

**核心發現**: 三項服務均存在成熟的開源逆向工程實作，技術上可行但法律風險顯著。Gemini 因官方 CLI 使用 OAuth + PKCE，最易實現橋接；ChatGPT 需處理 Cloudflare 挑戰與 Arkose 驗證，複雜度最高；Grok 因官方已提供 OpenAI-compatible API，橋接需求最低。

---

## 2. 研究背景與動機

### 2.1 問題陳述

| 痛點 | 說明 |
|------|------|
| **無 API Key 的使用者** | 一般使用者擁有 ChatGPT Plus / SuperGrok 訂閱，但無法透過 API 使用已付費服務 |
| **API 費用高昂** | 官方 API 按 token 計費，大量使用成本驚人（GPT-5.4: $2.50/$15 per MTok; Grok 4: $3/$15 per MTok） |
| **Web 版與 API 分離** | ChatGPT Plus 訂閱無法用於 API；Gemini Advanced 訂閱無法用於 Gemini API |
| **缺少靈活整合** | 開發者需要將 AI 服務整合到自有工具鏈，但僅有 Web 介面可用 |

### 2.2 研究目標

1. **映射** 三大服務的 Web API 端點與認證機制
2. **提取** 瀏覽器 session 中的 access token / session token
3. **轉換** 私有 API 協定為標準 OpenAI-compatible 格式
4. **轉發** 請求到原生應用或 CLI 工具
5. **維持** Token 自動刷新以確保持續可用

---

## 3. 三大服務認證架構分析

### 3.1 ChatGPT (OpenAI)

| 元件 | 規格 |
|------|------|
| **登入端點** | `chat.openai.com/auth/login` → Auth0 OAuth2 流程 |
| **API Base** | `https://chat.openai.com/backend-api/` |
| **主要端點** | `POST /backend-api/conversation` (SSE streaming) |
| **認證方式** | Bearer Token (`__Secure-next-auth.session-token` + access token) |
| **Token 類型** | JWT access token (短效, ~1h) + session cookie (長效) |
| **額外驗證** | Cloudflare Challenge + Arkose Labs captcha (不定時觸發) |
| **模型 ID** | `text-davinci-002-render-sha` (GPT-3.5), `gpt-4-*` (GPT-4系列) |
| **回應格式** | SSE (`data: {...}\n\ndata: [DONE]`) |

**認證流程**:
```
Browser → chat.openai.com/auth/login
         → Redirect: auth0.openai.com (OAuth2)
         → User login (email/password)
         → Auth0 callback → access_token
         → Frontend stores in sessionStorage
         → Subsequent API calls: Authorization: Bearer <token>
```

**逆向難度**: ⚠️ **高** — 需處理 Auth0 OAuth2 流程、Session token 提取、不定期 Cloudflare 驗證、Arkose captcha。

### 3.2 Grok (xAI)

| 元件 | 規格 |
|------|------|
| **登入端點** | `grok.com` (X.com 帳號整合登入) |
| **API Base** | `https://api.x.ai/v1/` (官方公有 API) / 另有私有 WebSocket 端點 |
| **認證方式** | Bearer token / API key |
| **Token 類型** | JWT access token (OAuth 2.0) |
| **官方程式** | Grok Build CLI (Rust, 開源) 使用同樣 OAuth 流程 |
| **回應格式** | SSE / NDJSON |
| **特色** | 已提供完整 OpenAI-compatible REST API (`api.x.ai/v1`) |

**認證流程**:
```
Browser → grok.com
         → X.com SSO login
         → OAuth2 token issuance
         → WebSocket connection for chat
```

**逆向難度**: 🟢 **低** — xAI 已提供官方 OpenAI-compatible API (`docs.x.ai`)，Bridge 需求最低。開源 `grok-bypass` 專案仍存在以繞過 API 付費。

### 3.3 Gemini (Google)

| 元件 | 規格 |
|------|------|
| **登入端點** | `gemini.google.com/app` (Google 帳號 OAuth) |
| **API Base** | `https://cloudcode-pa.googleapis.com` (內部端點) |
| **認證方式** | OAuth 2.0 + PKCE (同 Gemini CLI) |
| **Token 類型** | Google OAuth access token (~1h) + refresh token (長效) |
| **CLI 整合** | Gemini CLI (`gemini-cli`) 使用公開 OAuth client ID |
| **模型 ID** | `gemini-2.5-pro`, `gemini-2.5-flash` 等 |
| **回應格式** | gRPC-web / SSE streaming |

**認證流程**:
```
Browser → gemini.google.com
         → Google 帳號 OAuth 2.0 + PKCE
         → OAuth token stored in cookie + indexDB
         → API calls via gRPC-web (Authorization: Bearer <token>)
```

**逆向難度**: 🟡 **中** — Google 官方 CLI 使用公開 OAuth 2.0 + PKCE 流程，最易橋接。開源專案 `gemini-proxy` 已提供完整實作。

### 3.4 對照總結

| 面向 | ChatGPT | Grok | Gemini |
|------|---------|------|--------|
| OAuth Provider | Auth0 | X.com | Google Identity |
| Token 提取難度 | 高 (session cookie) | 低 (官方 API) | 中 (OAuth PKCE) |
| 需破解 Captcha | 是 (Arkose) | 否 | 否 |
| 官方 API 存在 | 是 (付費) | 是 (付費) | 是 (付費) |
| Web API Bridge 需求 | 高 | 低 | 中 |
| 開源 Bridge 專案 | `webchatgpt` (PyPI) | `grok-bypass` | `gemini-proxy`, `gemini-web-to-api` |
| 帳號風險 | 中 (違反 ToS) | 中 (違反 ToS) | 高 (Google 明確禁止) |

---

## 4. 核心技術棧

### 4.1 MITM Proxy (中間人代理)

| 工具 | 類型 | 優點 | 缺點 |
|------|------|------|------|
| **mitmproxy** | Open source / Python | 腳本化 (Python addon)、支援 HTTP/2/3、WebSocket、mitmweb GUI | 需手動安裝 CA 憑證 |
| **Charles Proxy** | 商業軟體 | GUI 友善、SSL 解密即開即用 | 非開源、需授權費 |
| **Fiddler** | 免費 (Windows) | .NET 生態整合、自動配置系統代理 | 不支援 HTTP/2 (早期版本) |
| **Proxyman** | 商業 (macOS) | 原生 macOS 體驗、自動 CA 配置 | 僅 macOS |

**推薦**: **mitmproxy** (開源、腳本化、跨平台、社群活躍)

### 4.2 瀏覽器自動化 / Cookie 提取

| 工具 | 適用場景 |
|------|----------|
| **Puppeteer** (Node.js) | 無頭瀏覽器操作、Cookie/Token 提取 |
| **Playwright** (Python/Node.js) | 跨瀏覽器自動化、支援 Chromium/Firefox/WebKit |
| **Selenium** | 傳統瀏覽器自動化 |
| **Browser Extension (userscript)** | 透過 `GM_cookie` API 提取 cookie (暴力破解) |

### 4.3 Cookie → Token 提取方式

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  瀏覽器登入      │ ──► │  MITM Proxy     │ ──► │  Token 提取器    │
│  (Puppeteer)    │     │  (mitmproxy)    │     │  (Python addon) │
└─────────────────┘     └─────────────────┘     └────────┬────────┘
                                                         │
                                                         ▼
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  原生 App / CLI  │ ◄── │  API Bridge      │ ◄── │  Token Store     │
│  (OpenAI SDK)   │     │  (FastAPI)       │     │  (redis/記憶體)  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

### 4.4 Token 提取方法比較

| 方法 | 可靠性 | 複雜度 | 自動化程度 | 適用服務 |
|------|--------|--------|-----------|----------|
| **mitmproxy addon 攔截** | 🟢 高 | 🟡 中 | 🟢 全自動 | ChatGPT, Grok, Gemini |
| **Puppeteer CDP (Chrome DevTools Protocol)** | 🟢 高 | 🟡 中 | 🟢 全自動 | 所有 Web 服務 |
| **手動複製 Cookie (userscript)** | 🟡 中 | 🟢 低 | 🔴 半自動 | 開發者腳本測試 |
| **CLI OAuth 流程** | 🟢 高 | 🟢 低 | 🟢 全自動 | Gemini (官方 CLI) |

---

## 5. 系統架構設計

### 5.1 高階架構

```
┌─────────────────────────────────────────────────────────────────────┐
│                       本地主機 (localhost)                           │
│                                                                     │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────────────┐ │
│  │ 瀏覽器    │──►│ MITM     │──►│ Token    │──►│ API Bridge       │ │
│  │ (Chrome) │   │ Proxy    │   │ Extractor│   │ (FastAPI)        │ │
│  └──────────┘   └──────────┘   └────┬─────┘   └──────┬───────────┘ │
│       │                              │                 │            │
│       │                              ▼                 ▼            │
│       │                       ┌──────────┐   ┌──────────────────┐ │
│       │                       │ Token    │   │ OpenAI-Compatible│ │
│       │                       │ Store    │   │ REST API         │ │
│       │                       │ (Redis)  │   │ localhost:8080   │ │
│       │                       └──────────┘   └──────────────────┘ │
│       │                                                           │
│       ▼                                                           │
│  ┌───────────────────────────────────────────────────────────┐    │
│  │  AI 服務端點 (chat.openai.com / api.x.ai / gemini.google) │    │
│  └───────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       應用程式層                                     │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────────────┐ │
│  │ Open Web │   │ CLI Tool │   │ Mobile   │   │ Custom App       │ │
│  │ UI       │   │ (curl)   │   │ App      │   │ (Python/JS/Rust) │ │
│  └──────────┘   └──────────┘   └──────────┘   └──────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.2 元件分解

| 元件 | 職責 | 技術選擇 |
|------|------|----------|
| **MITM Proxy** | 攔截 HTTP/HTTPS 流量，提取認證資訊 | mitmproxy (Python addon) |
| **Token Extractor** | 從攔截流量解析 OAuth/API Token | mitmproxy addon + regex/JWT decode |
| **Token Store** | 儲存 Token、管理刷新邏輯 | Redis / 記憶體 dict |
| **API Bridge** | 轉換私有 API 為標準格式 | FastAPI + OpenAI-compatible schema |
| **Session Manager** | 管理瀏覽器 session 生命週期 | Playwright / Puppeteer |
| **Health Monitor** | 檢查 token 有效性，觸發自動刷新 | 背景定時任務 |

### 5.3 資料流 (Data Flow)

**正向流 (請求)**:
```
App Request → API Bridge (localhost:8080/v1/chat/completions)
            → 查找對應上游端點
            → 附加 Bearer Token
            → 轉換請求格式 (OpenAI → 私有格式)
            → 發送到 AI 服務
```

**反向流 (回應)**:
```
AI Service Response (SSE/JSON)
            → 轉換回應格式 (私有格式 → OpenAI)
            → 串流/非串流回傳
            → App 接收到標準回應
```

---

## 6. 實作方法論

### 6.1 方法一: mitmproxy + Token Extractor (通用方案)

**適合**: ChatGPT、Grok 網頁版、所有 Web-based 服務

```python
# mitmproxy addon: token_extractor.py
"""
從反向代理流量中提取 OAuth/API Token
"""
import json
import re
from mitmproxy import http
from mitmproxy import ctx

class TokenExtractor:
    """攔截 API 請求並提取 Bearer Token"""

    # 定義各服務的特徵
    SERVICE_PATTERNS = {
        "chatgpt": {
            "domain": "chat.openai.com",
            "token_header": r"Bearer\s+([\w\-\.]+)",
            "token_endpoint": "/backend-api/",
        },
        "grok": {
            "domain": "grok.com",
            "token_header": r"Bearer\s+([\w\-\.]+)",
            "token_endpoint": "/v1/",
        },
        "gemini": {
            "domain": "gemini.google.com",
            "token_header": r"Bearer\s+([\w\-\.]+)",
            "token_endpoint": "/v1alpha/",
        },
    }

    def __init__(self):
        self.tokens = {}
        self.store = {}  # Token 儲存（可注入外部 Redis）

    def request(self, flow: http.HTTPFlow):
        """攔截請求出 Authorization header"""
        auth = flow.request.headers.get("Authorization", "")
        host = flow.request.pretty_host

        if not auth.startswith("Bearer "):
            return

        token = auth.replace("Bearer ", "").strip()

        # 根據 domain 分類
        for service, config in self.SERVICE_PATTERNS.items():
            if config["domain"] in host and config["token_endpoint"] in flow.request.path:
                old = self.tokens.get(service)
                if token != old:
                    self.tokens[service] = token
                    self.store[service] = token
                    ctx.log.info(f"[{service}] Token updated: {token[:20]}...")
                break

    def get_token(self, service: str) -> str:
        """供 API Bridge 查詢 Token"""
        return self.tokens.get(service, "")

addons = [TokenExtractor()]
```

### 6.2 方法二: Playwright + Cookie 提取 (適合自動化登入)

**適合**: 需要保持瀏覽器 session 的情境

```python
# browser_session.py — 使用 Playwright 維持登入會話
import asyncio
from playwright.async_api import async_playwright

class BrowserSession:
    """管理瀏覽器 session 以獲得 API token"""

    def __init__(self, service_url: str):
        self.service_url = service_url
        self.browser = None
        self.context = None
        self.page = None

    async def start(self):
        p = await async_playwright().start()
        # 使用 persistent context 保持 cookie
        self.browser = await p.chromium.launch_persistent_context(
            user_data_dir="./browser-data",
            headless=False,  # 首次登入需要手動操作
        )
        self.page = await self.browser.new_page()
        await self.page.goto(self.service_url)

    async def get_access_token(self) -> str:
        """從 localStorage / sessionStorage 提取 token"""
        token = await self.page.evaluate("""
            () => {
                // ChatGPT: sessionStorage 中的 accessToken
                const parts = [
                    sessionStorage.getItem('accessToken'),
                    localStorage.getItem('accessToken'),
                ];
                for (const p of parts) {
                    if (p) return p;
                }
                return null;
            }
        """)
        return token

    async def get_cookies(self) -> dict:
        """提取所有 cookies"""
        cookies = await self.context.cookies()
        return {c["name"]: c["value"] for c in cookies}

    async def refresh_if_needed(self):
        """定期檢查 token 有效性"""
        token = await self.get_access_token()
        if not token:
            # 重新導向登入頁
            await self.page.goto(self.service_url)
            # 等待使用者手動登入
            await self.page.wait_for_url(f"{self.service_url}/**")
        return await self.get_access_token()
```

### 6.3 方法三: Gemini OAuth + PKCE (純 CLI，不需瀏覽器)

**適合**: Gemini（最乾淨的方式）

```python
# gemini_oauth.py — 使用 Google OAuth 2.0 + PKCE
"""
使用公開的 Gemini CLI OAuth 憑證進行認證
無需模擬瀏覽器或手動提取 Cookie
"""
import os
import requests
from authlib.integrations.requests_client import OAuth2Session

# 來自 Gemini CLI 的公開 Client ID
GEMINI_CLIENT_ID = (
    "44752326432-q6k3s6k3s6k3s6k3s6k3s6k3s6k3s6k3.apps."
    "googleusercontent.com"
)
GEMINI_AUTH_URL = "https://accounts.google.com/o/oauth2/v2/auth"
GEMINI_TOKEN_URL = "https://oauth2.googleapis.com/token"
GEMINI_API_BASE = "https://cloudcode-pa.googleapis.com"

SCOPES = [
    "openid",
    "https://www.googleapis.com/auth/cloud-platform",
    "https://www.googleapis.com/auth/generative-language.retriever",
]

class GeminiOAuthBridge:
    """Gemini OAuth 橋接器"""

    def __init__(self):
        self.session = None
        self.token = None

    def login(self):
        """啟動 OAuth + PKCE 流程（首次需要瀏覽器）"""
        client = OAuth2Session(
            client_id=GEMINI_CLIENT_ID,
            scope=SCOPES,
            redirect_uri="http://localhost:8080/oauth/callback",
            pkce_challenge_method="S256",
        )

        # 產生授權 URL
        uri, state = client.create_authorization_url(GEMINI_AUTH_URL)
        print(f"請在瀏覽器中開啟此 URL 並登入:\n{uri}")

        # 啟動本地回呼伺服器
        from http.server import HTTPServer, BaseHTTPRequestHandler
        import threading

        auth_code = [None]

        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                from urllib.parse import urlparse, parse_qs
                query = parse_qs(urlparse(self.path).query)
                auth_code[0] = query.get("code", [None])[0]
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b"Authorization complete! You can close this tab.")

        server = HTTPServer(("localhost", 8080), Handler)
        thread = threading.Thread(target=server.handle_request)
        thread.start()
        thread.join()

        # 交換 Token
        token = client.fetch_token(
            GEMINI_TOKEN_URL,
            code=auth_code[0],
        )
        self.token = token
        return token

    def refresh_token(self):
        """自動刷新 access token"""
        if not self.token or "refresh_token" not in self.token:
            return self.login()

        client = OAuth2Session(
            client_id=GEMINI_CLIENT_ID,
            token=self.token,
        )
        self.token = client.refresh_token(GEMINI_TOKEN_URL)
        return self.token

    def chat(self, messages: list, model: str = "gemini-2.5-pro"):
        """呼叫 Gemini API"""
        headers = {
            "Authorization": f"Bearer {self.token['access_token']}",
            "Content-Type": "application/json",
        }
        payload = {
            "contents": [{"role": m["role"], "parts": [{"text": m["content"]}]}
                         for m in messages],
        }
        resp = requests.post(
            f"{GEMINI_API_BASE}/v1alpha/models/{model}:generateContent",
            headers=headers,
            json=payload,
        )
        return resp.json()
```

---

## 7. Token 生命週期管理

### 7.1 Token 類型與生命週期

| 服務 | Token 類型 | 生命週期 | 刷新機制 |
|------|-----------|----------|----------|
| **ChatGPT** | JWT access token | ~1 小時 | Session token (cookie) 自動換新 |
| **ChatGPT** | Session cookie (`__Secure-next-auth.session-token`) | ~30 天 | 需重新登入 |
| **Grok (xAI)** | JWT access token | ~2 小時 | Refresh token (OAuth2) |
| **Gemini** | Google OAuth access token | ~1 小時 | Refresh token (OAuth2 PKCE) |

### 7.2 Token Store 設計

```python
# token_store.py — Token 生命週期管理
import time
import threading
from dataclasses import dataclass

@dataclass
class TokenRecord:
    access_token: str
    refresh_token: str = ""
    expires_at: float = 0
    token_type: str = "Bearer"

class TokenStore:
    """執行緒安全的 Token 儲存與刷新管理"""

    def __init__(self):
        self._tokens: dict[str, TokenRecord] = {}
        self._lock = threading.Lock()
        self._refresh_callbacks: dict[str, callable] = {}

    def set_refresh_callback(self, service: str, callback: callable):
        """設定 token 刷新回呼函數"""
        self._refresh_callbacks[service] = callback

    def get_token(self, service: str) -> str:
        """取得有效 token，必要時自動刷新"""
        with self._lock:
            record = self._tokens.get(service)
            if not record:
                return ""

            # 檢查是否將要過期 (預留 5 分鐘緩衝)
            if record.expires_at and time.time() >= record.expires_at - 300:
                callback = self._refresh_callbacks.get(service)
                if callback:
                    # 非同步刷新 (另開執行緒避免阻塞)
                    threading.Thread(target=self._do_refresh,
                                     args=(service, callback),
                                     daemon=True).start()
            return record.access_token

    def _do_refresh(self, service: str, callback: callable):
        """執行 token 刷新"""
        try:
            new_token = callback(self._tokens.get(service))
            if new_token:
                self._tokens[service] = new_token
        except Exception as e:
            print(f"[TokenStore] {service} refresh failed: {e}")

    def set_token(self, service: str, record: TokenRecord):
        """更新 token"""
        with self._lock:
            self._tokens[service] = record

    def is_valid(self, service: str) -> bool:
        """檢查 token 是否有效"""
        token = self.get_token(service)
        return bool(token)
```

### 7.3 Refresh 策略

```
Token 有效時段:
├───────────────┤
  ↑ 取得 Token       ↑ Auto-refresh threshold    ↑ 過期
  (time=0)           (time=expires_at-5min)       (time=expires_at)

策略:
1. Lazy Refresh: 僅在 API 呼叫發現 401 時觸發刷新
2. Proactive Refresh: 定時檢查 + threshold 觸發刷新
3. Race Condition 保護: 使用鎖避免同時多個刷新請求

建議:
- Gemini: 使用 OAuth refresh token (可靠)
- ChatGPT: 監聽 mitmproxy 中的 cookie 更新事件
- Grok: 使用 xAI 官方 API key (不需要 refresh)
```

---

## 8. API 協定轉換層

### 8.1 OpenAI-Compatible 格式

目標轉換格式是 OpenAI Chat Completions API，這是業界事實標準：

```http
POST /v1/chat/completions
Host: localhost:8080
Authorization: Bearer <any-key>
Content-Type: application/json

{
  "model": "gpt-4o",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello!"}
  ],
  "stream": true,
  "max_tokens": 1024
}
```

### 8.2 協定轉換引擎

```python
# api_bridge.py — 協定轉換伺服器 (FastAPI)
from fastapi import FastAPI, HTTPException
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
import httpx
import json
from typing import Optional, Literal

app = FastAPI(title="AI API Bridge")

# ── 上游服務設定 ──
UPSTREAM_CONFIG = {
    "chatgpt": {
        "base_url": "https://chat.openai.com/backend-api",
        "chat_endpoint": "/conversation",
        "model_map": {
            "gpt-4o": "gpt-4o-2024-11-20",
            "gpt-4": "text-davinci-002-render-sha",
        }
    },
    "gemini": {
        "base_url": "https://cloudcode-pa.googleapis.com/v1alpha",
        "chat_endpoint": "/models/gemini-2.5-pro:streamGenerateContent",
        "model_map": {
            "gemini-2.5-pro": "gemini-2.5-pro",
            "gemini-2.5-flash": "gemini-2.5-flash",
        }
    },
    "grok": {
        "base_url": "https://api.x.ai/v1",
        "chat_endpoint": "/chat/completions",
        "model_map": {
            "grok-4": "grok-4-0709",
            "grok-3": "grok-3-auto",
        }
    }
}

# ── 要求模型 ──
class ChatMessage(BaseModel):
    role: Literal["system", "user", "assistant"]
    content: str

class ChatRequest(BaseModel):
    model: str = "gpt-4o"
    messages: list[ChatMessage]
    stream: bool = False
    max_tokens: Optional[int] = None
    temperature: Optional[float] = None

# ── 路由 ──
@app.post("/v1/chat/completions")
async def chat_completions(req: ChatRequest):
    """OpenAI-compatible 聊天完成端點"""

    # 自動路由到對應上游
    upstream = _resolve_upstream(req.model)
    if not upstream:
        raise HTTPException(400, f"Unknown model: {req.model}")

    # 取得 Token
    token = token_store.get_token(upstream["service"])
    if not token:
        raise HTTPException(401, f"Token not available for {upstream['service']}")

    # 轉換請求格式
    upstream_request = _convert_request(req, upstream)

    if req.stream:
        return StreamingResponse(
            _stream_upstream(upstream, token, upstream_request),
            media_type="text/event-stream",
        )
    else:
        return await _call_upstream(upstream, token, upstream_request)


def _resolve_upstream(model: str) -> dict | None:
    """根據模型名稱解析上游服務"""
    for service, config in UPSTREAM_CONFIG.items():
        if model in config["model_map"]:
            return {
                "service": service,
                "config": config,
                "model": config["model_map"][model],
            }
    return None


def _convert_request(req: ChatRequest, upstream: dict) -> dict:
    """轉換 OpenAI 格式 → 上游私有格式"""
    service = upstream["service"]

    if service == "chatgpt":
        # ChatGPT 需要 message tree 格式
        return {
            "action": "next",
            "messages": [{
                "id": _uuid(),
                "role": m.role,
                "content": {
                    "content_type": "text",
                    "parts": [m.content],
                },
            } for m in req.messages],
            "model": upstream["model"],
            "parent_message_id": _uuid(),
        }

    elif service == "gemini":
        return {
            "contents": [
                {"role": m.role, "parts": [{"text": m.content}]}
                for m in req.messages
            ],
        }

    elif service == "grok":
        # Grok 原生支援 OpenAI 格式
        return {
            "model": upstream["model"],
            "messages": [{"role": m.role, "content": m.content}
                        for m in req.messages],
            "stream": req.stream,
        }


async def _stream_upstream(upstream: dict, token: str, body: dict):
    """串流轉換回 OpenAI SSE 格式"""
    service = upstream["service"]
    config = upstream["config"]

    async with httpx.AsyncClient() as client:
        url = f"{config['base_url']}{config['chat_endpoint']}"
        headers = {"Authorization": f"Bearer {token}"}

        async with client.stream("POST", url, headers=headers, json=body) as resp:
            async for line in resp.aiter_lines():
                if not line:
                    continue

                # 轉換為 OpenAI SSE 格式
                openai_chunk = _convert_stream_chunk(service, line)
                if openai_chunk:
                    yield f"data: {json.dumps(openai_chunk)}\n\n"

    yield "data: [DONE]\n\n"


def _convert_stream_chunk(service: str, raw_line: str) -> dict | None:
    """轉換上游串流 chunk → OpenAI 格式"""
    if service == "chatgpt":
        # ChatGPT SSE: data: {"message": {...}}
        if raw_line.startswith("data: "):
            data = json.loads(raw_line[6:])
            # OpenAI 格式
            return {
                "choices": [{
                    "delta": {"content": data.get("message", {})
                              .get("content", {})
                              .get("parts", [""])[0]},
                    "index": 0,
                }]
            }
    elif service == "gemini":
        # Gemini: 轉換 gRPC-web 格式為 OpenAI
        try:
            data = json.loads(raw_line)
            text = data.get("candidates", [{}])[0].get("content", {})\
                       .get("parts", [{}])[0].get("text", "")
            return {
                "choices": [{"delta": {"content": text}, "index": 0}]
            }
        except json.JSONDecodeError:
            return None
    return None
```

### 8.3 支援的 API 端點

完整的 Bridge API 應實作：

| 端點 | 方法 | OpenAI 規格 | 本 Bridge |
|------|------|-------------|-----------|
| `/v1/chat/completions` | POST | ✅ | ✅ |
| `/v1/models` | GET | ✅ | ✅ |
| `/v1/completions` | POST | ✅ | ⬜ 選擇性 |
| `/v1/embeddings` | POST | ✅ | ⬜ 選擇性 |

---

## 9. 安全考量

### 9.1 風險矩陣

| 風險 | 嚴重性 | 可能性 | 緩解措施 |
|------|--------|--------|----------|
| Token 洩漏 | 🔴 高 | 🟡 中 | Token 僅存記憶體，不寫入磁碟 |
| MITM CA 憑證遭濫用 | 🔴 高 | 🟢 低 | 使用一次性 CA，用完即銷毀 |
| SQL Injection | 🟡 中 | 🟢 低 | 使用參數化查詢 |
| 未授權訪問 Bridge | 🟡 中 | 🟡 中 | 綁定 localhost，加入 API key 驗證 |
| 帳號被 Ban | 🔴 高 | 🟡 中 | 使用 Token 池輪換，控制請求頻率 |

### 9.2 安全建議

1. **僅繫結 localhost** — 禁止外部網路訪問 Bridge
   ```python
   import uvicorn
   uvicorn.run(app, host="127.0.0.1", port=8080)  # 非 0.0.0.0
   ```
2. **使用 Virtual API Key** — 在 Bridge 前再加一層 API Key 驗證
3. **不持久化 Token** — Token 僅儲存於記憶體，避免檔案洩漏
4. **限制請求速率** — 避免觸發上游服務的異常檢測
   ```python
   from slowapi import Limiter
   limiter = Limiter(key_func=lambda: "global")
   @limiter.limit("10/second")
   ```
5. **定期更換 MITM CA 憑證** — 降低憑證被提取的損害

---

## 10. 法律與倫理邊界

### 10.1 服務條款對照

| 服務 | ToS 相關條款 | 逆向工程禁令 |
|------|-------------|-------------|
| **OpenAI** | 禁止未經授權的 API 存取、禁止繞過存取控制 | ✅ 明確禁止 |
| **xAI (Grok)** | 禁止逆向工程、禁止自動化存取 Web 版 | ✅ 明確禁止 |
| **Google (Gemini)** | 禁止逆向工程、禁止未經授權的 API 存取 | ✅ 明確禁止（2026/02 更新後更嚴格） |

### 10.2 已知法律事件

- **2026 年 2 月**: Google 更新 ToS，明確禁止使用 Gemini CLI OAuth 流程為第三方 API 代理，並已 Ban 部分違規付費帳號 ([source](https://developer.puter.com/tutorials/gemini-oauth/))
- **2023 年**: OpenAI 多次更新 Cloudflare 驗證機制，對抗逆向工程
- **xAI**: 官方 API 已夠便宜 ($3/$15 per MTok)，Bypass 需求低

### 10.3 建議使用方針

| 使用情境 | 建議 |
|----------|------|
| 個人開發測試 | 🟡 可接受，但需承擔帳號風險 |
| 商業產品使用 | 🔴 強烈不建議 — 會產生法律責任 |
| 學術研究 | 🟢 合理使用 — 僅限逆向工程研究 |
| 取代官方付費 API | 🔴 不建議 — 破壞商業模式 |
| 備援方案 (emergency) | 🟡 短期可接受，長期應轉官方 API |

**免責聲明**: 本報告純粹為技術研究目的，不鼓勵違反任何服務條款的行為。讀者應自行評估法律風險。

---

## 11. 開源生態系分析

### 11.1 主要專案比較

| 專案 | 目標服務 | 語言 | Star | 授權 | 最後更新 | 說明 |
|------|---------|------|------|------|---------|------|
| **gemini-proxy** ([KashifKhn](https://github.com/KashifKhn/gemini-proxy)) | Gemini | TypeScript (Bun/Hono) | ~30 | MIT | 2026-03 | OAuth + PKCE 認證，OpenAI-compatible，自動 token 刷新、SSE 串流、function calling |
| **gemini-web-to-api** ([ntthanh2603](https://github.com/ntthanh2603/gemini-web-to-api)) | Gemini | Go | — | MIT | 2025-12 | Cookie-based auth，REST API wrapper，Docker 支援 |
| **grok-bypass** ([saraansx](https://github.com/saraansx/grok-bypass)) | Grok (xAI) | Python | — | — | 2026 | 匿名 session pool、fingerprint rotation、Cloudflare 規避、image generation |
| **webchatgpt** (PyPI) | ChatGPT | Python | — | GPLv3 | 2024-04 | Reverse engineering of ChatGPT web-version，CLI + SDK |
| **re-gpt** (PyPI) | ChatGPT | Python | — | Apache-2.0 | 2024 | Session token based ChatGPT client |
| **LiteLLM** ([BerriAI](https://github.com/BerriAI/litellm)) | 100+ LLM | Python | 51.4k | MIT | 2026 (活躍) | 官方 API Proxy，非逆向工程，但支援 Gemini/Grok/OpenAI |
| **OpenAI HTTP Proxy** ([Nayjest](https://github.com/Nayjest/lm-proxy)) | Multi-provider | Python | — | MIT | 2026-06 | OpenAI-compatible gateway，正規 API 代理 |
| **mitmproxy2swagger** | 通用 | Python | — | MIT | 2025 | 將 mitmproxy 流量轉換為 OpenAPI 規格 |

### 11.2 生態系趨勢

```
2023          2024          2025          2026
├─────────────┼─────────────┼─────────────┼─────────────┤
ChatGPT Reverse Engineering 高峰期
│
├── acheong08/ChatGPT (archived)
├── webchatgpt
├── re-gpt
│
Gemini Reverse Engineering 崛起
│            ├── gemini-web-to-api
│                        ├── gemini-proxy     ← 最具架構完整性
│
Grok 官方 API 開放
│                        ├── grok-bypass      ← 匿名池模擬
│
趨向: OpenAI-Compatible 協議統一
│                        ├── LiteLLM 51.4k★
│                        ├── openai-http-proxy
│                        └── 正規代理取代逆向工程
```

**關鍵觀察**: 2025-2026 年趨勢是從「逆向工程私有 API」轉向「使用正規 API + OpenAI-compatible 代理」。原因：
1. 官方 API 價格逐年下降（GPT-5.4 mini: $0.75/$4.50 per MTok）
2. 服務商加強反制（Google ToS 更新、Account Ban）
3. 統一協定（OpenAI Chat Completions）成為業界標準
4. LiteLLM 等工具讓多 Provider 切換零成本

---

## 12. 結論與建議

### 12.1 核心結論

1. **技術上可行**: 三項服務均可透過不同方法實現 API 橋接，最成熟的路徑是 Gemini OAuth + PKCE
2. **ChatGPT 最困難**: Arkose captcha + Cloudflare + message tree 格式，維護成本最高
3. **Grok 最不需要**: xAI 已有官方 OpenAI-compatible API，價格具競爭力
4. **法律風險遞增**: Google 已開始 Ban 違規帳號，OpenAI 持續強化反制
5. **趨勢轉向正規 API**: 2026 年的主流方案已轉向 LiteLLM 等多 Provider 代理

### 12.2 建議策略

```
情境 1: 個人開發、研究用途
→ 推薦: Gemini OAuth PKCE (方法三) + FastAPI Bridge
→ Token 管理: OAuth refresh token (自動刷新)
→ 風險: 低 (少量使用，Google 不易偵測)

情境 2: 已有訂閱、想在自有工具使用
→ 推薦: mitmproxy (方法一) + 選擇性橋接 ChatGPT/Grok
→ Token 管理: Browser session 提取 (手動)
→ 風險: 中 (依賴手動操作、低頻率)

情境 3: 團隊多模型整合
→ 推薦: LiteLLM + 正規 API Key
→ 費用: 官方 API 計費
→ 風險: 零 (完全合規)
```

### 12.3 未來展望

| 方向 | 預測時程 | 影響 |
|------|----------|------|
| OpenAI 推出「Web 版 API 附加方案」 | 2026-2027 | 🟢 Bridge 需求大幅降低 |
| Google 全面鎖定 OAuth 流程 | 已開始 (2026/02) | 🔴 Gemini Bridge 不可行 |
| xAI API 持續降價 | 2026 H2 | 🟢 Grok Bypass 不再必要 |
| Cloudflare 挑戰強化 | 持續進化 | 🔴 ChatGPT Bridge 需頻繁更新 |
| 標準化 API 協議 | 進行中 | 🟢 Bridge 維護成本降低 |

---

## 13. 參考資料

### 文獻與研究

1. Zai-Kun. (2023). *Reverse Engineered ChatGPT API*. GitHub. https://github.com/Zai-Kun/reverse-engineered-chatgpt
2. acheong08. (2023). *ChatGPT: Reverse engineered ChatGPT API*. GitHub. https://github.com/acheong08/ChatGPT
3. 0xdevalias. (2023). *Notes on reverse engineering ChatGPT's frontend web app*. GitHub Gist. https://gist.github.com/0xdevalias/4ac297ee3f794c17d0997b4673a2f160
4. Aaron. (2023). *ChatGPT Web Reverse Engineering: Building an OpenAI-Compatible API*. Hideaway Blog. https://niceboy.org/en/posts/2023/03/chatgpt-web-to-api
5. Simatwa. (2024). *WebChatGPT: Reverse Engineering of ChatGPT Web-Version*. PyPI. https://pypi.org/project/webchatgpt
6. Kin Lane. (2025). *Reverse Engineering the APIs Behind Everything With Mitmproxy*. API Evangelist. https://apievangelist.com/2025/04/21/reverse-engineering-the-apis-behind-everything-with-mitmproxy
7. KashifKhn. (2026). *gemini-proxy: Self-hosted OpenAI-compatible API proxy for Google Gemini*. GitHub. https://github.com/KashifKhn/gemini-proxy
8. ntthanh2603. (2025). *gemini-web-to-api: Transforms Google Gemini web interface into a standardized REST API*. GitHub. https://github.com/ntthanh2603/gemini-web-to-api
9. saraansx. (2026). *grok-bypass: Reverse-engineered Grok API with OpenAI-compatible endpoints*. GitHub. https://github.com/saraansx/grok-bypass
10. BerriAI. (2026). *LiteLLM: Python SDK, Proxy Server to call 100+ LLM APIs*. GitHub. https://github.com/BerriAI/litellm
11. Nayjest. (2026). *OpenAI HTTP Proxy / LM-Proxy*. PyPI. https://pypi.org/project/openai-http-proxy
12. Simon Willison. (2025). *Reverse engineering Codex CLI to get GPT-5-Codex-Mini*. https://simonwillison.net/2025/Nov/9/gpt-5-codex-mini
13. Reynaldi Chernando. (2026). *How to do OAuth with Gemini*. Puter.js Tutorials. https://developer.puter.com/tutorials/gemini-oauth
14. xAI. (2026). *xAI API Documentation*. https://docs.x.ai
15. Google AI. (2026). *Gemini API Ephemeral Tokens*. https://ai.google.dev/gemini-api/docs/live-api/ephemeral-tokens
16. OpenAI. (2026). *Codex Authentication*. https://developers.openai.com/codex/auth
17. mitmproxy Project. (2026). *mitmproxy Documentation*. https://docs.mitmproxy.org
18. Markaicode. (2026). *OpenAI API Production Architecture: Caching, Rate Limiting, and Fallback*. https://markaicode.com/architecture/openai-api-system-design-architecture-999
19. AppSec Santa. (2026). *mitmproxy Review 2026: Free HTTPS Intercepting Proxy*. https://appsecsanta.com/mitmproxy

### 技術標準

20. OpenAI. (2026). *Chat Completions API Reference*. https://platform.openai.com/docs/api-reference/chat
21. NVIDIA. (2024). *NVIDIA NIM: Optimized Inference Microservices*. https://build.nvidia.com/nim

---

**報告結束** — 本文件由 Research Team 於 2026-06-26 產出，所有資訊以學術研究為目的。

*免責聲明: 本報告中的技術資訊僅供教育和研究參考。使用者應遵守各服務的條款與條件，並自行承擔使用風險。*
