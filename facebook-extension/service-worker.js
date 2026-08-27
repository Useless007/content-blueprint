import {
  CONTENT_PACK_SCHEMA,
  ValidationError,
  buildPrompt,
  normalizeGroundingUrl,
  parseInteractionResponse,
  validateBrief,
  validateContentPack,
} from "./src/core.js";
import { GrowthValidationError, validateGrowthSnapshot } from "./src/growth.js";

const GEMINI_ENDPOINT = "https://generativelanguage.googleapis.com/v1beta/interactions";
const GEMINI_MODEL = "gemini-3.7-flash";
const REQUEST_TIMEOUT_MS = 120_000;
const MAX_RESPONSE_BYTES = 4 * 1024 * 1024;
const MAX_REQUEST_BYTES = 512 * 1024;
const MAX_LOCAL_STATE_BYTES = 500 * 1024;
const NATIVE_HOST = "com.contentblueprint.facebook";

const SESSION_API_KEY = "geminiApiKey";
const LOCAL_DRAFT_KEY = "facebookContentDraft";
const LOCAL_SETTINGS_KEY = "facebookContentSettings";

const DEFAULT_SETTINGS = Object.freeze({ useGrounding: true });

const MESSAGE_ALIASES = Object.freeze({
  FBP_GET_STATE: "state.get",
  FBP_SAVE_API_KEY: "settings.save",
  FBP_CLEAR_API_KEY: "settings.clear",
  FBP_SAVE_LOCAL_STATE: "state.save",
  FBP_GENERATE: "generate.facebookPack",
  "settings.clearApiKey": "settings.clear",
  FBP_COMPANION_HEALTH: "companion.health",
  FBP_COMPANION_SYNC_BRIEF: "companion.syncBrief",
  FBP_COMPANION_GET_PACK: "companion.getLatestPack",
  FBP_COMPANION_GET_GROWTH_PACK: "companion.getLatestGrowthPack",
});

// Clicking the toolbar icon opens this extension's side panel. Failures are
// intentionally ignored here; the panel can still be opened from Chrome UI.
if (chrome.sidePanel?.setPanelBehavior) {
  try {
    const behavior = chrome.sidePanel.setPanelBehavior({ openPanelOnActionClick: true });
    if (behavior && typeof behavior.catch === "function") {
      behavior.catch(() => {});
    }
  } catch {
    // Older Chromium builds can expose the API before supporting the promise form.
  }
}

function isPlainObject(value) {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    return false;
  }
  const prototype = Object.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}

function utf8Length(value) {
  return new TextEncoder().encode(value).byteLength;
}

function storageGet(area, keys) {
  return new Promise((resolve, reject) => {
    area.get(keys, (result) => {
      const problem = chrome.runtime.lastError;
      if (problem) {
        reject(new ExtensionError("STORAGE_ERROR", problem.message || "Could not read extension storage"));
      } else {
        resolve(result);
      }
    });
  });
}

function storageSet(area, values) {
  return new Promise((resolve, reject) => {
    area.set(values, () => {
      const problem = chrome.runtime.lastError;
      if (problem) {
        reject(new ExtensionError("STORAGE_ERROR", problem.message || "Could not save extension storage"));
      } else {
        resolve();
      }
    });
  });
}

function storageRemove(area, keys) {
  return new Promise((resolve, reject) => {
    area.remove(keys, () => {
      const problem = chrome.runtime.lastError;
      if (problem) {
        reject(new ExtensionError("STORAGE_ERROR", problem.message || "Could not clear extension storage"));
      } else {
        resolve();
      }
    });
  });
}

function sendNativeMessage(message) {
  return new Promise((resolve, reject) => {
    chrome.runtime.sendNativeMessage(NATIVE_HOST, message, (response) => {
      const problem = chrome.runtime.lastError;
      if (problem) {
        reject(
          new ExtensionError(
            "COMPANION_UNAVAILABLE",
            "ไม่พบ Content Blueprint Companion กรุณาเปิดโปรแกรมหรือติดตั้ง Native Host ก่อน",
          ),
        );
        return;
      }
      if (!isPlainObject(response)) {
        reject(new ExtensionError("COMPANION_INVALID_RESPONSE", "Companion ส่งคำตอบที่ไม่ถูกต้อง"));
        return;
      }
      if (response.ok !== true) {
        reject(
          new ExtensionError(
            typeof response.errorCode === "string" ? response.errorCode : "COMPANION_ERROR",
            cleanProviderMessage(response.message, "", 600) || "Companion ทำงานไม่สำเร็จ",
          ),
        );
        return;
      }
      resolve(response);
    });
  });
}

async function companionHealth() {
  const response = await sendNativeMessage({ action: "health" });
  return {
    connected: true,
    protocolVersion: typeof response.protocolVersion === "string" ? response.protocolVersion : "",
  };
}

async function syncCompanionBrief(message) {
  const result = validateBrief(message.brief);
  if (!result.valid) {
    throw new ExtensionError("VALIDATION_ERROR", "Content brief is invalid", result.errors);
  }
  const response = await sendNativeMessage({ action: "saveBrief", brief: result.value });
  return {
    briefRevision: typeof response.briefRevision === "string" ? response.briefRevision : "",
    updatedAt: typeof response.updatedAt === "string" ? response.updatedAt : "",
  };
}

function normalizeCompanionSources(value) {
  if (!Array.isArray(value)) {
    return [];
  }
  const sources = [];
  const seen = new Set();
  for (const candidate of value.slice(0, 30)) {
    if (!isPlainObject(candidate)) {
      continue;
    }
    const url = normalizeGroundingUrl(candidate.url);
    if (url === null || seen.has(url)) {
      continue;
    }
    seen.add(url);
    sources.push({
      title: cleanProviderMessage(candidate.title, "", 500),
      url,
    });
  }
  return sources;
}

async function getLatestCompanionPack(message) {
  const request = { action: "getLatestPack" };
  if (typeof message.briefRevision === "string" && message.briefRevision.trim() !== "") {
    request.briefRevision = message.briefRevision.trim();
  }
  const response = await sendNativeMessage(request);
  if (response.found !== true || !isPlainObject(response.snapshot)) {
    return { found: false, stale: false };
  }
  if (response.stale === true) {
    return {
      found: true,
      stale: true,
      briefRevision: typeof response.briefRevision === "string" ? response.briefRevision : "",
    };
  }
  const pack = validateContentPack(response.snapshot.pack);
  const sources = normalizeCompanionSources(response.snapshot.groundingSources);
  return {
    found: true,
    stale: false,
    briefRevision: typeof response.briefRevision === "string" ? response.briefRevision : "",
    updatedAt: typeof response.updatedAt === "string" ? response.updatedAt : "",
    pack,
    content: pack,
    sources,
    groundingSources: sources,
    model:
      cleanProviderMessage(response.snapshot.generatedBy, "", 120) ||
      "Content Blueprint Companion",
  };
}

async function getLatestGrowthPack(message) {
  const request = { action: "getLatestGrowthPack" };
  if (typeof message.briefRevision === "string" && message.briefRevision.trim() !== "") {
    request.briefRevision = message.briefRevision.trim();
  }
  const response = await sendNativeMessage(request);
  if (response.action !== "getLatestGrowthPack" || response.protocolVersion !== "1.0") {
    throw new ExtensionError("COMPANION_INVALID_RESPONSE", "Growth Companion protocol response is invalid");
  }
  if (response.found !== true) {
    return { found: false, stale: false };
  }
  if (!isPlainObject(response.growthSnapshot)) {
    throw new ExtensionError("COMPANION_INVALID_RESPONSE", "Growth Companion did not return a typed snapshot");
  }
  try {
    const snapshot = validateGrowthSnapshot(response.growthSnapshot);
    return {
      found: true,
      stale: response.stale === true,
      briefRevision: typeof response.briefRevision === "string" ? response.briefRevision : "",
      updatedAt: typeof response.updatedAt === "string" ? response.updatedAt : "",
      snapshot,
    };
  } catch (error) {
    if (error instanceof GrowthValidationError) {
      throw new ExtensionError("COMPANION_INVALID_RESPONSE", "Growth Companion snapshot failed strict validation");
    }
    throw error;
  }
}

class ExtensionError extends Error {
  constructor(code, message, details = []) {
    super(message);
    this.name = "ExtensionError";
    this.code = code;
    this.details = Array.isArray(details) ? details : [];
  }
}

function restrictStorageArea(area) {
  if (!area || typeof area.setAccessLevel !== "function") {
    return Promise.resolve();
  }
  return new Promise((resolve) => {
    area.setAccessLevel({ accessLevel: "TRUSTED_CONTEXTS" }, () => {
      // Older Chromium builds may not support this method. The API key still
      // remains in storage.session, whose default access is trusted contexts.
      void chrome.runtime.lastError;
      resolve();
    });
  });
}

const storageAccessReady = Promise.all([
  restrictStorageArea(chrome.storage.session),
  restrictStorageArea(chrome.storage.local),
]);

function trustedExtensionSender(sender) {
  if (!sender || sender.id !== chrome.runtime.id) {
    return false;
  }
  const senderURL = typeof sender.url === "string" ? sender.url : "";
  if (senderURL === "") {
    // Extension service workers do not carry a tab. Reject content scripts
    // and web-page senders when Chrome omits their URL.
    return !sender.tab;
  }
  try {
    const parsed = new URL(senderURL);
    return parsed.protocol === "chrome-extension:" && parsed.hostname === chrome.runtime.id;
  } catch {
    return false;
  }
}

function normalizeSettings(value) {
  const settings = isPlainObject(value) ? value : {};
  return {
    useGrounding:
      typeof settings.useGrounding === "boolean"
        ? settings.useGrounding
        : DEFAULT_SETTINGS.useGrounding,
  };
}

function validateAPIKey(value) {
  if (typeof value !== "string") {
    throw new ExtensionError("INVALID_API_KEY", "API key must be text");
  }
  const key = value.trim();
  if (key.length < 10 || key.length > 512 || /[\u0000-\u0020\u007F]/u.test(key)) {
    throw new ExtensionError("INVALID_API_KEY", "API key has an invalid format");
  }
  return key;
}

function containsSecretField(value, depth = 0) {
  if (depth > 12 || value === null || typeof value !== "object") {
    return depth > 12;
  }
  if (Array.isArray(value)) {
    return value.some((item) => containsSecretField(item, depth + 1));
  }
  for (const [key, child] of Object.entries(value)) {
    if (/(?:api.?key|access.?token|authorization|password|secret)/iu.test(key)) {
      return true;
    }
    if (containsSecretField(child, depth + 1)) {
      return true;
    }
  }
  return false;
}

function normalizeDraft(value) {
  if (!isPlainObject(value)) {
    throw new ExtensionError("INVALID_DRAFT", "Draft must be an object");
  }
  if (containsSecretField(value)) {
    throw new ExtensionError(
      "SECRET_IN_LOCAL_STATE",
      "Draft was not saved because it contains a field that may hold a secret",
    );
  }
  let serialized;
  try {
    serialized = JSON.stringify(value);
  } catch {
    throw new ExtensionError("INVALID_DRAFT", "Draft could not be encoded");
  }
  if (serialized === undefined || utf8Length(serialized) > MAX_LOCAL_STATE_BYTES) {
    throw new ExtensionError("INVALID_DRAFT", "Draft is too large to save locally");
  }
  const clone = JSON.parse(serialized);
  if (!isPlainObject(clone)) {
    throw new ExtensionError("INVALID_DRAFT", "Draft must be a plain object");
  }
  return clone;
}

async function getStoredState() {
  const [local, session] = await Promise.all([
    storageGet(chrome.storage.local, [LOCAL_DRAFT_KEY, LOCAL_SETTINGS_KEY]),
    storageGet(chrome.storage.session, [SESSION_API_KEY]),
  ]);
  return {
    draft: isPlainObject(local[LOCAL_DRAFT_KEY]) ? local[LOCAL_DRAFT_KEY] : {},
    settings: normalizeSettings(local[LOCAL_SETTINGS_KEY]),
    apiKeyConfigured:
      typeof session[SESSION_API_KEY] === "string" && session[SESSION_API_KEY].trim() !== "",
  };
}

async function saveSettings(message) {
  const current = await storageGet(chrome.storage.local, [LOCAL_SETTINGS_KEY]);
  const incoming = isPlainObject(message.settings) ? { ...message.settings } : {};
  if (typeof message.useGrounding === "boolean") {
    incoming.useGrounding = message.useGrounding;
  }
  const settings = normalizeSettings({
    ...normalizeSettings(current[LOCAL_SETTINGS_KEY]),
    ...incoming,
  });
  await storageSet(chrome.storage.local, { [LOCAL_SETTINGS_KEY]: settings });

  if (Object.prototype.hasOwnProperty.call(message, "apiKey")) {
    const apiKey = validateAPIKey(message.apiKey);
    await storageSet(chrome.storage.session, { [SESSION_API_KEY]: apiKey });
  }
  const session = await storageGet(chrome.storage.session, [SESSION_API_KEY]);
  return {
    settings,
    apiKeyConfigured:
      typeof session[SESSION_API_KEY] === "string" && session[SESSION_API_KEY].trim() !== "",
  };
}

async function saveDraft(draft) {
  const normalized = normalizeDraft(draft);
  await storageSet(chrome.storage.local, { [LOCAL_DRAFT_KEY]: normalized });
  return normalized;
}

function cleanProviderMessage(value, apiKey, limit = 600) {
  let text = typeof value === "string" ? value : "";
  if (apiKey) {
    text = text.split(apiKey).join("[REDACTED]");
  }
  text = text.replace(/[\u0000-\u001F\u007F]+/gu, " ").replace(/\s+/gu, " ").trim();
  return text.length <= limit ? text : `${text.slice(0, limit)}…`;
}

function providerError(status, responseText, apiKey) {
  let providerMessage = "";
  try {
    const envelope = JSON.parse(responseText);
    providerMessage = cleanProviderMessage(envelope?.error?.message, apiKey);
  } catch {
    // Provider HTML and other unstructured bodies are intentionally not shown.
  }

  if (status === 401 || status === 403) {
    return new ExtensionError(
      "AUTH_FAILED",
      "Gemini rejected the API key. Open Settings and save a valid key for this browser session.",
    );
  }
  if (status === 429) {
    return new ExtensionError(
      "RATE_LIMITED",
      "Gemini rate limit or quota was reached. Wait briefly, then try again or check the key quota.",
    );
  }
  const suffix = providerMessage ? `: ${providerMessage}` : "";
  return new ExtensionError(
    "PROVIDER_ERROR",
    `Gemini returned HTTP ${status || "redirect"}${suffix}`,
  );
}

async function readResponseText(response) {
  const declaredLength = Number(response.headers.get("content-length"));
  if (Number.isFinite(declaredLength) && declaredLength > MAX_RESPONSE_BYTES) {
    throw new ExtensionError("RESPONSE_TOO_LARGE", "Gemini response exceeded the 4 MiB safety limit");
  }
  if (!response.body || typeof response.body.getReader !== "function") {
    const text = await response.text();
    if (utf8Length(text) > MAX_RESPONSE_BYTES) {
      throw new ExtensionError("RESPONSE_TOO_LARGE", "Gemini response exceeded the 4 MiB safety limit");
    }
    return text;
  }

  const reader = response.body.getReader();
  const chunks = [];
  let received = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      received += value.byteLength;
      if (received > MAX_RESPONSE_BYTES) {
        await reader.cancel();
        throw new ExtensionError("RESPONSE_TOO_LARGE", "Gemini response exceeded the 4 MiB safety limit");
      }
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }
  const bytes = new Uint8Array(received);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return new TextDecoder("utf-8", { fatal: false }).decode(bytes);
}

async function generateFacebookPack(message) {
  if (Object.prototype.hasOwnProperty.call(message, "apiKey")) {
    throw new ExtensionError(
      "SECRET_IN_MESSAGE",
      "Save the API key in Settings before generating; generation messages must not contain secrets",
    );
  }
  const prompt = buildPrompt(message.brief);
  const [session, local] = await Promise.all([
    storageGet(chrome.storage.session, [SESSION_API_KEY]),
    storageGet(chrome.storage.local, [LOCAL_SETTINGS_KEY]),
  ]);
  const apiKey = session[SESSION_API_KEY];
  if (typeof apiKey !== "string" || apiKey.trim() === "") {
    throw new ExtensionError(
      "API_KEY_MISSING",
      "Add a Gemini API key in Settings. It will be kept only for this browser session.",
    );
  }

  const savedSettings = normalizeSettings(local[LOCAL_SETTINGS_KEY]);
  const useGrounding =
    typeof message.useGrounding === "boolean"
      ? message.useGrounding
      : isPlainObject(message.settings) && typeof message.settings.useGrounding === "boolean"
        ? message.settings.useGrounding
        : savedSettings.useGrounding;

  const requestPayload = {
    model: GEMINI_MODEL,
    input: prompt.input,
    system_instruction: prompt.systemInstruction,
    store: false,
    response_format: {
      type: "text",
      mime_type: "application/json",
      schema: CONTENT_PACK_SCHEMA,
    },
  };
  if (useGrounding) {
    requestPayload.tools = [{ type: "google_search" }, { type: "url_context" }];
  }
  const body = JSON.stringify(requestPayload);
  if (utf8Length(body) > MAX_REQUEST_BYTES) {
    throw new ExtensionError("REQUEST_TOO_LARGE", "The content brief is too large to send safely");
  }

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  let response;
  let responseText;
  try {
    response = await fetch(GEMINI_ENDPOINT, {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
        "x-goog-api-key": apiKey,
      },
      body,
      signal: controller.signal,
      redirect: "manual",
      credentials: "omit",
      cache: "no-store",
      referrerPolicy: "no-referrer",
    });
    responseText = await readResponseText(response);
  } catch (error) {
    if (error instanceof ExtensionError) {
      throw error;
    }
    if (error?.name === "AbortError") {
      throw new ExtensionError("TIMEOUT", "Gemini took longer than 2 minutes. Try again with a shorter brief.");
    }
    const detail = cleanProviderMessage(error?.message, apiKey, 240);
    throw new ExtensionError(
      "NETWORK_ERROR",
      detail ? `Could not reach Gemini: ${detail}` : "Could not reach Gemini. Check the network and try again.",
    );
  } finally {
    clearTimeout(timer);
  }

  // A manual redirect is never followed, so the authenticated header cannot
  // be replayed to a different origin.
  if (response.type === "opaqueredirect" || response.status < 200 || response.status >= 300) {
    throw providerError(response.status, responseText, apiKey);
  }

  let result;
  try {
    result = parseInteractionResponse(responseText);
  } catch (error) {
    if (error instanceof ValidationError) {
      throw new ExtensionError(
        "INVALID_RESPONSE",
        cleanProviderMessage(error.message, apiKey, 800) || "Gemini returned an invalid response",
        error.errors.map((item) => cleanProviderMessage(String(item), apiKey, 300)),
      );
    }
    throw error;
  }
  return {
    pack: result.content,
    content: result.content,
    sources: result.groundingSources,
    groundingSources: result.groundingSources,
    usage: result.usage,
    model: result.model || GEMINI_MODEL,
  };
}

async function handleMessage(message, sender) {
  if (!trustedExtensionSender(sender)) {
    throw new ExtensionError("UNAUTHORIZED_SENDER", "This request did not come from an extension page");
  }
  if (!isPlainObject(message) || typeof message.type !== "string") {
    throw new ExtensionError("INVALID_MESSAGE", "Extension message must include a type");
  }
  await storageAccessReady;
  const type = MESSAGE_ALIASES[message.type] || message.type;

  switch (type) {
    case "state.get":
    case "settings.get":
      return getStoredState();

    case "settings.save":
      return saveSettings(message);

    case "settings.clear": {
      await storageRemove(chrome.storage.session, [SESSION_API_KEY]);
      const state = await getStoredState();
      return { settings: state.settings, apiKeyConfigured: false };
    }

    case "draft.load": {
      const local = await storageGet(chrome.storage.local, [LOCAL_DRAFT_KEY]);
      return { draft: isPlainObject(local[LOCAL_DRAFT_KEY]) ? local[LOCAL_DRAFT_KEY] : {} };
    }

    case "draft.save":
      await saveDraft(message.draft);
      return {};

    case "state.save": {
      const operations = [];
      if (Object.prototype.hasOwnProperty.call(message, "draft")) {
        operations.push(saveDraft(message.draft));
      }
      if (
        Object.prototype.hasOwnProperty.call(message, "settings") ||
        Object.prototype.hasOwnProperty.call(message, "useGrounding") ||
        Object.prototype.hasOwnProperty.call(message, "apiKey")
      ) {
        operations.push(saveSettings(message));
      }
      await Promise.all(operations);
      return getStoredState();
    }

    case "generate.facebookPack":
      return generateFacebookPack(message);

    case "companion.health":
      return companionHealth();

    case "companion.syncBrief":
      return syncCompanionBrief(message);

    case "companion.getLatestPack":
      return getLatestCompanionPack(message);

    case "companion.getLatestGrowthPack":
      return getLatestGrowthPack(message);

    default:
      throw new ExtensionError("UNSUPPORTED_MESSAGE", `Unsupported extension message: ${type.slice(0, 80)}`);
  }
}

function publicError(error) {
  if (error instanceof ExtensionError) {
    return {
      code: error.code,
      message: cleanProviderMessage(error.message, "", 800) || "Extension operation failed",
      ...(error.details.length > 0
        ? { details: error.details.map((item) => cleanProviderMessage(String(item), "", 300)).slice(0, 20) }
        : {}),
    };
  }
  if (error instanceof ValidationError) {
    return {
      code: "VALIDATION_ERROR",
      message: cleanProviderMessage(error.message, "", 800) || "Content brief is invalid",
      ...(error.errors.length > 0
        ? { details: error.errors.map((item) => cleanProviderMessage(String(item), "", 300)).slice(0, 20) }
        : {}),
    };
  }
  return { code: "INTERNAL_ERROR", message: "The extension could not complete this request" };
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  handleMessage(message, sender)
    .then((result) => sendResponse({ ok: true, ...result }))
    .catch((error) => sendResponse({ ok: false, error: publicError(error) }));
  return true;
});

export const SERVICE_CONSTANTS = Object.freeze({
  endpoint: GEMINI_ENDPOINT,
  model: GEMINI_MODEL,
  timeoutMs: REQUEST_TIMEOUT_MS,
  maxResponseBytes: MAX_RESPONSE_BYTES,
});
