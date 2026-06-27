# ChatGPT Normal-Chrome Extension Bridge Design

## Goal

After `ask auth chatgpt` opens and the user signs into ChatGPT in regular Google Chrome, `ask "prompt"` sends the prompt through that normal ChatGPT web page and prints the response. The interaction must not use Playwright, Rod, chromedp, CDP, headless Chrome, remote debugging, or a separate browser profile.

## Chosen architecture

A Manifest V3 unpacked Chrome extension runs a content script on `https://chatgpt.com/*`. For each prompt, the CLI starts a single-task HTTP bridge on a random `127.0.0.1` port with a random 256-bit token, then opens:

```text
https://chatgpt.com/#ask-cli-token=<token>&ask-cli-port=<port>
```

The content script removes the hash immediately and asks its extension service worker to fetch the task. The content script enters the prompt into ChatGPT's real composer, submits it, waits until the newest assistant response stops changing, and returns text through the service worker to the loopback bridge.

## Security boundaries

- The bridge binds only to `127.0.0.1` on an ephemeral port.
- Every endpoint requires a cryptographically random, single-run token.
- The task accepts one matching result ID.
- Request bodies have a fixed size limit.
- No cookie, access token, page storage, or browser profile data leaves Chrome.
- The extension only has host permissions for `chatgpt.com` and loopback HTTP.
- URL fragments are not sent to the ChatGPT server and are removed by the content script.

## CLI behavior

- The default backend becomes `web`.
- `ask "prompt"` starts the bridge, opens regular Chrome, waits briefly for the extension to claim the task, then waits up to `--timeout` for a response.
- `--backend api` preserves the existing direct backend API flow.
- `--backend rod` and `--backend chromedp` preserve existing explicit browser modes.
- Images are rejected in the extension bridge until a separately tested upload protocol exists.
- If the extension is absent, the CLI returns an actionable error with the unpacked extension path.

## Extension setup

Chrome requires a one-time user action:

1. Open `chrome://extensions`.
2. Enable Developer mode.
3. Choose **Load unpacked**.
4. Select `extension/chatgpt-bridge`.

`ask extension` opens the extensions page and the extension folder, then prints these steps.

## DOM behavior

The content script:

1. waits for `#prompt-textarea`, `textarea`, or a compatible contenteditable composer;
2. records the current assistant-message count;
3. inserts text and dispatches input/change events;
4. clicks a specific send button, falling back to an Enter event;
5. waits for a new assistant message;
6. returns once no stop-generating button exists and the response text is stable;
7. returns a clear error for login pages, missing selectors, timeout, or runtime disconnect.

## Verification

- Go bridge protocol tests cover authentication, claim, result, invalid ID, body limits, timeout, and cleanup.
- CLI tests prove `web` routing and actionable missing-extension errors.
- Extension manifest tests verify minimal permissions and matches.
- `node --check` validates extension JavaScript syntax.
- Full `go test ./...`, `go vet ./...`, build, CLI help, and native Chrome smoke checks run before delivery.
