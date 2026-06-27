"use strict";

const COMPOSER_SELECTORS = [
  "#prompt-textarea",
  "[data-testid='prompt-textarea']",
  "textarea",
  "[contenteditable='true']",
];

const SEND_BUTTON_SELECTORS = [
  "button[data-testid='send-button']",
  "button[aria-label='Send prompt']",
  "button[aria-label='Send']",
];

const ASSISTANT_SELECTOR = "[data-message-author-role='assistant']";
const STOP_BUTTON_SELECTORS = [
  "button[data-testid='stop-button']",
  "button[data-testid='stop-generating-button']",
  "button[aria-label='Stop generating']",
];

startBridgeFromFragment();

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type !== "ask-cli-task") {
    return false;
  }

  runTask(message.task)
    .then((content) => sendResponse({ id: message.task.id, content }))
    .catch((error) =>
      sendResponse({
        id: message.task?.id || "",
        error: error instanceof Error ? error.message : String(error),
      }),
    );
  return true;
});

function startBridgeFromFragment() {
  const values = new URLSearchParams(window.location.hash.slice(1));
  const token = values.get("ask-cli-token") || "";
  const port = Number(values.get("ask-cli-port"));
  if (!/^[a-f0-9]{64}$/.test(token) || !Number.isInteger(port)) {
    return;
  }

  history.replaceState(null, document.title, window.location.pathname + window.location.search);
  chrome.runtime
    .sendMessage({
      type: "ask-cli-start",
      token,
      port,
    })
    .catch((error) => console.error("ask-cli extension bridge failed", error));
}

async function runTask(task) {
  if (!task || typeof task.prompt !== "string" || task.prompt.trim() === "") {
    throw new Error("ask-cli supplied an empty prompt");
  }

  const timeoutMs = clamp(Number(task.timeout_ms) || 180000, 1000, 600000);
  const composer = await waitForElement(COMPOSER_SELECTORS, Math.min(timeoutMs, 30000));
  if (!composer) {
    throw new Error("ChatGPT composer not found; confirm that this Chrome profile is signed in");
  }

  const assistantCount = document.querySelectorAll(ASSISTANT_SELECTOR).length;
  setComposerText(composer, task.prompt);

  const sendButton = await waitForEnabledButton(SEND_BUTTON_SELECTORS, 5000);
  if (sendButton) {
    sendButton.click();
  } else {
    composer.dispatchEvent(
      new KeyboardEvent("keydown", {
        key: "Enter",
        code: "Enter",
        bubbles: true,
        cancelable: true,
      }),
    );
  }

  return waitForAssistantResponse(assistantCount, timeoutMs);
}

function setComposerText(composer, prompt) {
  composer.focus();

  if (composer instanceof HTMLTextAreaElement || composer instanceof HTMLInputElement) {
    const prototype =
      composer instanceof HTMLTextAreaElement
        ? HTMLTextAreaElement.prototype
        : HTMLInputElement.prototype;
    const setter = Object.getOwnPropertyDescriptor(prototype, "value")?.set;
    if (setter) {
      setter.call(composer, prompt);
    } else {
      composer.value = prompt;
    }
    composer.dispatchEvent(new InputEvent("input", { bubbles: true, data: prompt }));
    composer.dispatchEvent(new Event("change", { bubbles: true }));
    return;
  }

  const inserted = document.execCommand("selectAll", false, null) &&
    document.execCommand("insertText", false, prompt);
  if (!inserted) {
    composer.replaceChildren();
    const paragraph = document.createElement("p");
    paragraph.textContent = prompt;
    composer.appendChild(paragraph);
    composer.dispatchEvent(
      new InputEvent("input", {
        bubbles: true,
        inputType: "insertText",
        data: prompt,
      }),
    );
  }
}

async function waitForAssistantResponse(previousCount, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  let lastText = "";
  let stableSince = 0;

  while (Date.now() < deadline) {
    const messages = [...document.querySelectorAll(ASSISTANT_SELECTOR)];
    if (messages.length > previousCount) {
      const text = (messages.at(-1)?.innerText || "").trim();
      if (text && text === lastText) {
        if (!hasVisibleElement(STOP_BUTTON_SELECTORS) && Date.now() - stableSince >= 1500) {
          return text;
        }
      } else {
        lastText = text;
        stableSince = Date.now();
      }
    }
    await sleep(250);
  }

  throw new Error("timed out waiting for the ChatGPT response");
}

async function waitForElement(selectors, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const element = findVisibleElement(selectors);
    if (element) {
      return element;
    }
    await sleep(250);
  }
  return null;
}

async function waitForEnabledButton(selectors, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const button = findVisibleElement(selectors);
    if (button && !button.disabled && button.getAttribute("aria-disabled") !== "true") {
      return button;
    }
    await sleep(100);
  }
  return null;
}

function hasVisibleElement(selectors) {
  return Boolean(findVisibleElement(selectors));
}

function findVisibleElement(selectors) {
  for (const selector of selectors) {
    for (const element of document.querySelectorAll(selector)) {
      if (element.getClientRects().length > 0) {
        return element;
      }
    }
  }
  return null;
}

function clamp(value, minimum, maximum) {
  return Math.min(maximum, Math.max(minimum, value));
}

function sleep(milliseconds) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}
