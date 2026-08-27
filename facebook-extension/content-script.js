(() => {
  "use strict";

  const MAX_INSERT_CHARS = 50_000;
  let lastUserFocusedEditable = null;

  function isFacebookPage() {
    const host = window.location.hostname.toLowerCase();
    return host === "facebook.com" || host.endsWith(".facebook.com");
  }

  function editableFromNode(node) {
    if (!(node instanceof Element)) {
      return null;
    }
    if (
      node instanceof HTMLTextAreaElement &&
      !node.disabled &&
      !node.readOnly
    ) {
      return node;
    }
    const textarea = node.closest("textarea");
    if (
      textarea instanceof HTMLTextAreaElement &&
      !textarea.disabled &&
      !textarea.readOnly
    ) {
      return textarea;
    }
    const contentEditable = node.closest('[contenteditable]:not([contenteditable="false"])');
    return contentEditable instanceof HTMLElement && contentEditable.isContentEditable
      ? contentEditable
      : null;
  }

  function rememberUserFocus(event) {
    if (!event.isTrusted) {
      return;
    }
    const editable = editableFromNode(event.target);
    if (editable) {
      lastUserFocusedEditable = editable;
    }
  }

  // These listeners only remember an element reference. They never read or
  // transmit page text, comments, messages, customer data, or DOM contents.
  document.addEventListener("focusin", rememberUserFocus, true);
  document.addEventListener("pointerdown", rememberUserFocus, true);

  function dispatchInput(target, text) {
    target.dispatchEvent(
      new InputEvent("input", {
        bubbles: true,
        composed: true,
        inputType: "insertText",
        data: text,
      }),
    );
  }

  function insertIntoTextarea(target, text) {
    target.focus({ preventScroll: true });
    const beforeInput = new InputEvent("beforeinput", {
      bubbles: true,
      cancelable: true,
      composed: true,
      inputType: "insertText",
      data: text,
    });
    if (!target.dispatchEvent(beforeInput)) {
      throw new Error("The Facebook editor blocked the insertion");
    }

    const start = Number.isInteger(target.selectionStart) ? target.selectionStart : 0;
    const end = Number.isInteger(target.selectionEnd) ? target.selectionEnd : start;
    HTMLTextAreaElement.prototype.setRangeText.call(target, text, start, end, "end");
    dispatchInput(target, text);
  }

  function selectionInside(target) {
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) {
      return null;
    }
    const range = selection.getRangeAt(0);
    return target.contains(range.commonAncestorContainer) ? selection : null;
  }

  function placeCaretAtEnd(target) {
    const selection = window.getSelection();
    if (!selection) {
      throw new Error("The browser selection is unavailable");
    }
    const range = document.createRange();
    range.selectNodeContents(target);
    range.collapse(false);
    selection.removeAllRanges();
    selection.addRange(range);
    return selection;
  }

  function insertContentEditableFallback(target, text) {
    const selection = selectionInside(target) || placeCaretAtEnd(target);
    const beforeInput = new InputEvent("beforeinput", {
      bubbles: true,
      cancelable: true,
      composed: true,
      inputType: "insertText",
      data: text,
    });
    if (!target.dispatchEvent(beforeInput)) {
      throw new Error("The Facebook editor blocked the insertion");
    }

    const range = selection.getRangeAt(0);
    range.deleteContents();
    const textNode = document.createTextNode(text);
    range.insertNode(textNode);
    range.setStartAfter(textNode);
    range.collapse(true);
    selection.removeAllRanges();
    selection.addRange(range);
    dispatchInput(target, text);
  }

  function insertIntoContentEditable(target, text) {
    target.focus({ preventScroll: true });
    if (!selectionInside(target)) {
      placeCaretAtEnd(target);
    }

    let inputObserved = false;
    const observeInput = () => {
      inputObserved = true;
    };
    target.addEventListener("input", observeInput, { capture: true, once: true });
    let inserted = false;
    try {
      // execCommand is retained here because Chromium maps insertText through
      // the same editing path used by rich editors such as Facebook's composer.
      // The fallback below still inserts a Text node, never HTML.
      inserted = document.execCommand("insertText", false, text);
    } finally {
      target.removeEventListener("input", observeInput, true);
    }
    if (!inserted) {
      insertContentEditableFallback(target, text);
    } else if (!inputObserved) {
      dispatchInput(target, text);
    }
  }

  function validateSender(sender) {
    if (!sender || sender.id !== chrome.runtime.id) {
      return false;
    }
    if (typeof sender.url !== "string" || sender.url === "") {
      return !sender.tab;
    }
    try {
      const senderURL = new URL(sender.url);
      return senderURL.protocol === "chrome-extension:" && senderURL.hostname === chrome.runtime.id;
    } catch {
      return false;
    }
  }

  function handleInsert(message, sender) {
    if (!isFacebookPage()) {
      return { ok: false, error: { code: "WRONG_SITE", message: "Open facebook.com before inserting content" } };
    }
    if (!validateSender(sender)) {
      return { ok: false, error: { code: "UNAUTHORIZED_SENDER", message: "Insertion request was rejected" } };
    }
    if (!message || (message.type !== "facebook.insert" && message.type !== "FBP_INSERT_TEXT")) {
      return undefined;
    }
    if (
      typeof message.text !== "string" ||
      message.text.length === 0 ||
      message.text.length > MAX_INSERT_CHARS ||
      message.text.includes("\u0000")
    ) {
      return {
        ok: false,
        error: {
          code: "INVALID_TEXT",
          message: `Text must contain 1-${MAX_INSERT_CHARS} plain-text characters`,
        },
      };
    }

    const target = lastUserFocusedEditable;
    if (!target || !target.isConnected || !editableFromNode(target)) {
      return {
        ok: false,
        error: {
          code: "NO_EDITOR_FOCUSED",
          message: "Click a Facebook post or comment editor, then try Insert again",
        },
      };
    }

    try {
      if (target instanceof HTMLTextAreaElement) {
        insertIntoTextarea(target, message.text);
        return { ok: true, target: "textarea" };
      }
      insertIntoContentEditable(target, message.text);
      return { ok: true, target: "contenteditable" };
    } catch {
      return {
        ok: false,
        error: {
          code: "INSERT_FAILED",
          message: "Facebook's editor did not accept the text. Click the editor and try again",
        },
      };
    }
  }

  chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    const response = handleInsert(message, sender);
    if (response === undefined) {
      return false;
    }
    sendResponse(response);
    // Insertion is synchronous; no channel is kept open and no Post button is
    // queried or clicked anywhere in this script.
    return false;
  });
})();
