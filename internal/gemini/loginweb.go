package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"time"
)

// OAuthLoginPage is the web page shown for Gemini OAuth login.
const OAuthLoginPage = `<!DOCTYPE html>
<html lang="zh-TW">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ask-cli — Gemini OAuth Login</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; display: flex; justify-content: center; align-items: center; min-height: 100vh; }
  .card { background: white; border-radius: 12px; padding: 32px; max-width: 560px; width: 90%; box-shadow: 0 2px 12px rgba(0,0,0,0.1); }
  h1 { font-size: 20px; margin-bottom: 8px; color: #111; }
  p { color: #555; font-size: 14px; line-height: 1.6; margin-bottom: 16px; }
  ol { color: #555; font-size: 14px; line-height: 1.8; margin: 0 0 20px 20px; }
  .button { display: inline-block; background: #4285f4; color: white; text-decoration: none; border-radius: 8px; padding: 12px 24px; font-size: 15px; cursor: pointer; width: 100%; text-align: center; border: none; }
  .button:hover { background: #3367d6; }
  .error { color: #e53e3e; font-size: 13px; margin-top: 8px; }
  .success { color: #10a37f; font-size: 13px; margin-top: 8px; }
</style>
</head>
<body>
<div class="card">
  <h1>🔑 授權 Gemini</h1>
  <p>按下方按鈕前往 Google 登入並授權 ask-cli 使用 Gemini Web 訂閱：</p>
  <a href="{{.AuthURL}}" class="button" target="_blank">前往 Google 授權</a>
  <p style="margin-top:16px;font-size:13px;color:#888;">授權完成後會自動跳回此頁面。</p>
  <div id="status" style="margin-top:16px;"></div>
</div>
<script>
// Poll for completion
async function poll() {
  try {
    const r = await fetch('/status');
    const data = await r.json();
    if (data.done) {
      document.getElementById('status').innerHTML = '<div class="success">✅ 授權成功！可以關閉此頁面回到終端機。</div>';
    } else if (data.error) {
      document.getElementById('status').innerHTML = '<div class="error">❌ ' + data.error + '</div>';
    } else {
      setTimeout(poll, 500);
    }
  } catch(e) {
    setTimeout(poll, 1000);
  }
}
poll();
</script>
</body>
</html>`

// OAuthCallbackServer handles the OAuth redirect callback for Gemini login.
type OAuthCallbackServer struct {
	Port    int
	AuthURL string

	client   *Client
	done     chan struct{}
	authCode string
	authErr  string
	server   *http.Server
	listener net.Listener
}

// StartOAuthServer starts a local HTTP server for the Gemini OAuth flow.
// It returns the server instance and the URL the user should open.
func StartOAuthServer(client *Client) (*OAuthCallbackServer, string, error) {
	// Google's registered redirect URI is http://localhost:8085/oauth2callback,
	// so we MUST listen on port 8085 for the callback.
	listener, err := net.Listen("tcp", "127.0.0.1:8085")
	if err != nil {
		return nil, "", fmt.Errorf("cannot start local server on :8085: %w", err)
	}

	port := 8085

	authURL, _, err := client.AuthURL()
	if err != nil {
		listener.Close()
		return nil, "", fmt.Errorf("generate auth URL: %w", err)
	}

	ocs := &OAuthCallbackServer{
		Port:     port,
		AuthURL:  authURL,
		client:   client,
		done:     make(chan struct{}),
		listener: listener,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ocs.handleIndex)
	mux.HandleFunc("/oauth2callback", ocs.handleCallback)
	mux.HandleFunc("/status", ocs.handleStatus)

	ocs.server = &http.Server{
		Handler: mux,
	}

	go ocs.server.Serve(listener)

	loginURL := fmt.Sprintf("http://127.0.0.1:%d/", port)
	return ocs, loginURL, nil
}

// WaitForAuth blocks until the OAuth flow completes or the context is cancelled.
func (ocs *OAuthCallbackServer) WaitForAuth(ctx context.Context) error {
	select {
	case <-ocs.done:
		if ocs.authErr != "" {
			return fmt.Errorf("oauth failed: %s", ocs.authErr)
		}
		if ocs.authCode == "" {
			return fmt.Errorf("oauth callback received no code")
		}
		return ocs.client.ExchangeCode(ctx, ocs.authCode)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop shuts down the HTTP server.
func (ocs *OAuthCallbackServer) Stop() {
	if ocs.server != nil {
		ocs.server.Close()
	}
}

func (ocs *OAuthCallbackServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tpl, _ := template.New("oauth").Parse(OAuthLoginPage)
	tpl.Execute(w, map[string]string{"AuthURL": ocs.AuthURL})
}

func (ocs *OAuthCallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errStr := q.Get("error"); errStr != "" {
		ocs.authErr = errStr
		http.Error(w, "Authorization denied: "+errStr, http.StatusBadRequest)
		close(ocs.done)
		return
	}

	code := q.Get("code")
	if code == "" {
		ocs.authErr = "no authorization code received"
		http.Error(w, "No authorization code", http.StatusBadRequest)
		close(ocs.done)
		return
	}

	ocs.authCode = code
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html><html><body><p>✅ Authorization successful! You can close this tab and return to the terminal.</p><script>window.close()</script></body></html>`))

	go func() {
		time.Sleep(100 * time.Millisecond)
		close(ocs.done)
	}()
}

func (ocs *OAuthCallbackServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	select {
	case <-ocs.done:
		if ocs.authErr != "" {
			json.NewEncoder(w).Encode(map[string]any{"done": true, "error": ocs.authErr})
		} else {
			json.NewEncoder(w).Encode(map[string]any{"done": true})
		}
	default:
		json.NewEncoder(w).Encode(map[string]any{"done": false})
	}
}
