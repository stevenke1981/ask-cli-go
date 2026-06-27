package chatgpt

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"time"
)

// loginPage is the HTML page shown to the user for paste-login.
const loginPage = `<!DOCTYPE html>
<html lang="zh-TW">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ask-cli — ChatGPT Session Login</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; display: flex; justify-content: center; align-items: center; min-height: 100vh; }
  .card { background: white; border-radius: 12px; padding: 32px; max-width: 560px; width: 90%; box-shadow: 0 2px 12px rgba(0,0,0,0.1); }
  h1 { font-size: 20px; margin-bottom: 8px; color: #111; }
  p { color: #555; font-size: 14px; line-height: 1.6; margin-bottom: 16px; }
  ol { color: #555; font-size: 14px; line-height: 1.8; margin: 0 0 20px 20px; }
  textarea { width: 100%; padding: 12px; border: 1px solid #ddd; border-radius: 8px; font-family: monospace; font-size: 13px; resize: vertical; min-height: 80px; }
  textarea:focus { border-color: #10a37f; outline: none; box-shadow: 0 0 0 3px rgba(16,163,127,0.15); }
  button { background: #10a37f; color: white; border: none; border-radius: 8px; padding: 10px 24px; font-size: 15px; cursor: pointer; margin-top: 8px; width: 100%; }
  button:hover { background: #0e8c6b; }
  .error { color: #e53e3e; font-size: 13px; margin-top: 8px; display: none; }
  .success { color: #10a37f; font-size: 13px; margin-top: 8px; display: none; }
  .step { font-weight: 600; color: #10a37f; }
</style>
</head>
<body>
<div class="card">
  <h1>🔑 登入 ChatGPT</h1>
  <p>請從 Chrome 開發者工具複製你的 ChatGPT 登入憑證：</p>
  <ol>
    <li>在 Chrome 中打開 <strong>chatgpt.com</strong> 並確認已登入</li>
    <li>按 <strong>F12</strong> 開啟開發者工具</li>
    <li>切換到 <strong>Application</strong> (應用程式) 分頁</li>
    <li>左側選 <strong>Cookies</strong> → <strong>https://chatgpt.com</strong></li>
    <li>找到 <strong><code>__Secure-next-auth.session-token</code></strong></li>
    <li>雙擊 <strong>Value</strong> 欄位 → 全選複製 (Ctrl+C)</li>
    <li>貼到下方輸入框並按「儲存」</li>
  </ol>
  <textarea id="token" placeholder="貼上 __Secure-next-auth.session-token 的值..."></textarea>
  <div id="error" class="error"></div>
  <div id="success" class="success">✅ 登入成功！可以關閉此頁面回到終端機。</div>
  <button onclick="submitToken()">儲存並驗證</button>
</div>
<script>
async function submitToken() {
  const token = document.getElementById('token').value.trim();
  const errEl = document.getElementById('error');
  const okEl = document.getElementById('success');
  errEl.style.display = 'none';
  okEl.style.display = 'none';
  if (!token) {
    errEl.textContent = '請先貼上 session token';
    errEl.style.display = 'block';
    return;
  }
  try {
    const r = await fetch('/save', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token })
    });
    const data = await r.json();
    if (data.ok) {
      okEl.style.display = 'block';
    } else {
      errEl.textContent = data.error || '驗證失敗';
      errEl.style.display = 'block';
    }
  } catch(e) {
    errEl.textContent = '連線錯誤: ' + e.message;
    errEl.style.display = 'block';
  }
}
</script>
</body>
</html>`

// LoginServer starts a local HTTP server that provides a web UI for
// entering a ChatGPT session token, verifies it, and returns the token.
type LoginServer struct {
	Port     int
	Token    string
	Verified bool
	Error    string
	server   *http.Server
}

// StartLoginServer starts a local HTTP server on a random available port
// and returns the server instance and the URL to open.
func StartLoginServer() (*LoginServer, string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", fmt.Errorf("cannot start local server: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	ls := &LoginServer{Port: port}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ls.handleIndex)
	mux.HandleFunc("/save", ls.handleSave)

	ls.server = &http.Server{
		Handler: mux,
	}

	go ls.server.Serve(listener)

	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	return ls, url, nil
}

// WaitForToken blocks until a token is received (via the web form) or
// the context is cancelled, then returns the token (verified or not).
func (ls *LoginServer) WaitForToken(ctx context.Context) (string, bool, error) {
	for {
		select {
		case <-ctx.Done():
			return "", false, ctx.Err()
		default:
			if ls.Token != "" {
				return ls.Token, ls.Verified, nil
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// Stop shuts down the HTTP server.
func (ls *LoginServer) Stop() {
	if ls.server != nil {
		ls.server.Close()
	}
}

func (ls *LoginServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tpl, _ := template.New("login").Parse(loginPage)
	tpl.Execute(w, nil)
}

func (ls *LoginServer) handleSave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "method not allowed"})
		return
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}

	body.Token = trimSessionToken(body.Token)
	if body.Token == "" {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "token is empty"})
		return
	}

	// Verify the token by attempting to authenticate
	client := NewClient(body.Token)
	verifyCtx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := client.Authenticate(verifyCtx); err != nil {
		ls.Token = body.Token
		ls.Verified = false
		ls.Error = err.Error()
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("token 驗證失敗 (但仍可儲存): %v", err),
		})
		return
	}

	ls.Token = body.Token
	ls.Verified = true
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// trimSessionToken cleans up a session token value.
func trimSessionToken(s string) string {
	// Remove quotes, whitespace, and common wrappers
	result := s
	if len(result) > 0 && result[0] == '"' {
		result = result[1:]
	}
	if len(result) > 0 && result[len(result)-1] == '"' {
		result = result[:len(result)-1]
	}
	// Remove 'sid=' or 'session=' prefix if accidentally included
	for _, prefix := range []string{"sid=", "session=", "token="} {
		if len(result) > len(prefix) && result[:len(prefix)] == prefix {
			result = result[len(prefix):]
		}
	}
	return result
}
