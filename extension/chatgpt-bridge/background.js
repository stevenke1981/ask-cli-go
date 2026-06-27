"use strict";

const activeBridges = new Set();

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message?.type !== "ask-cli-start") {
    return false;
  }

  const tabId = sender.tab?.id;
  const token = String(message.token || "");
  const port = Number(message.port);
  if (
    !Number.isInteger(tabId) ||
    !/^[a-f0-9]{64}$/.test(token) ||
    !Number.isInteger(port) ||
    port < 1 ||
    port > 65535
  ) {
    sendResponse({ accepted: false });
    return false;
  }

  const bridgeKey = `${tabId}:${port}:${token}`;
  if (activeBridges.has(bridgeKey)) {
    sendResponse({ accepted: true });
    return false;
  }

  activeBridges.add(bridgeKey);
  runBridge(tabId, port, token)
    .then(() => sendResponse({ accepted: true, complete: true }))
    .catch((error) => {
      console.error("ask-cli bridge failed", error);
      sendResponse({
        accepted: true,
        complete: false,
        error: error instanceof Error ? error.message : String(error),
      });
    })
    .finally(() => activeBridges.delete(bridgeKey));
  return true;
});

async function runBridge(tabId, port, token) {
  const baseURL = `http://127.0.0.1:${port}`;
  const taskURL = `${baseURL}/v1/task?token=${encodeURIComponent(token)}`;
  const resultURL = `${baseURL}/v1/result?token=${encodeURIComponent(token)}`;

  const taskResponse = await fetchWithRetry(taskURL, 20, 250);
  if (!taskResponse.ok) {
    throw new Error(`task request failed with HTTP ${taskResponse.status}`);
  }
  const task = await taskResponse.json();

  let result;
  try {
    result = await chrome.tabs.sendMessage(tabId, {
      type: "ask-cli-task",
      task,
    });
    if (!result || result.id !== task.id) {
      throw new Error("content script returned an invalid result");
    }
  } catch (error) {
    result = {
      id: task.id,
      error: error instanceof Error ? error.message : String(error),
    };
  }

  const resultResponse = await fetch(resultURL, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(result),
    cache: "no-store",
  });
  if (!resultResponse.ok) {
    throw new Error(`result request failed with HTTP ${resultResponse.status}`);
  }
}

async function fetchWithRetry(url, attempts, delayMs) {
  let lastError;
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    try {
      return await fetch(url, { cache: "no-store" });
    } catch (error) {
      lastError = error;
      await sleep(delayMs);
    }
  }
  throw lastError || new Error("bridge request failed");
}

function sleep(milliseconds) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}
