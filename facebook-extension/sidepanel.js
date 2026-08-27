import { growthBlockToPlainText, growthInsertGuard, growthPackToPlainText } from "./src/growth.js";

const OUTPUT_SECTIONS = [
  { key: "hooks", aliases: ["hooks"], label: "Hooks", itemLabel: "Hook" },
  { key: "longPost", aliases: ["longPost", "long_post"], label: "โพสต์ยาว", itemLabel: "โพสต์ยาว" },
  { key: "shortPost", aliases: ["shortPost", "short_post"], label: "โพสต์สั้น", itemLabel: "โพสต์สั้น" },
  { key: "reelScript", aliases: ["reelScript", "reel_script"], label: "สคริปต์ Reel", itemLabel: "สคริปต์ Reel" },
  {
    key: "carousel",
    aliases: ["carousel", "carouselSlides", "carousel_slides"],
    label: "Carousel",
    itemLabel: "สไลด์",
  },
  { key: "cta", aliases: ["cta", "CTA", "ctas"], label: "CTA", itemLabel: "CTA" },
  {
    key: "firstComment",
    aliases: ["firstComment", "first_comment"],
    label: "คอมเมนต์แรก",
    itemLabel: "คอมเมนต์แรก",
  },
  {
    key: "replyBank",
    aliases: ["replyBank", "reply_bank", "replies"],
    label: "คลังคำตอบ",
    itemLabel: "คำตอบ",
  },
  {
    key: "complianceNotes",
    aliases: ["complianceNotes", "compliance_notes"],
    label: "ความเสี่ยง",
    itemLabel: "ข้อสังเกต",
  },
];

const elements = {
  status: document.querySelector("#appStatus"),
  settingsToggle: document.querySelector("#settingsToggle"),
  settingsPanel: document.querySelector("#settingsPanel"),
  closeSettings: document.querySelector("#closeSettings"),
  keyState: document.querySelector("#keyState"),
  apiKey: document.querySelector("#apiKey"),
  saveApiKey: document.querySelector("#saveApiKey"),
  clearApiKey: document.querySelector("#clearApiKey"),
  companionState: document.querySelector("#companionState"),
  fetchCompanionPack: document.querySelector("#fetchCompanionPack"),
  syncCompanionBrief: document.querySelector("#syncCompanionBrief"),
  growthFetch: document.querySelector("#growthFetch"),
  growthState: document.querySelector("#growthState"),
  growthResults: document.querySelector("#growthResults"),
  growthWarning: document.querySelector("#growthWarning"),
  growthMeta: document.querySelector("#growthMeta"),
  growthConfirmWrap: document.querySelector("#growthConfirmWrap"),
  growthPendingConfirm: document.querySelector("#growthPendingConfirm"),
  growthCopyAll: document.querySelector("#growthCopyAll"),
  growthInsertAll: document.querySelector("#growthInsertAll"),
  growthBlocks: document.querySelector("#growthBlocks"),
  briefForm: document.querySelector("#briefForm"),
  campaignProduct: document.querySelector("#campaignProduct"),
  audience: document.querySelector("#audience"),
  objective: document.querySelector("#objective"),
  offer: document.querySelector("#offer"),
  brandVoice: document.querySelector("#brandVoice"),
  evidence: document.querySelector("#evidence"),
  constraints: document.querySelector("#constraints"),
  useGrounding: document.querySelector("#useGrounding"),
  saveDraft: document.querySelector("#saveDraft"),
  loadDraft: document.querySelector("#loadDraft"),
  generateButton: document.querySelector("#generateButton"),
  generateSpinner: document.querySelector("#generateSpinner"),
  generateLabel: document.querySelector("#generateLabel"),
  resultsSection: document.querySelector("#resultsSection"),
  resultMeta: document.querySelector("#resultMeta"),
  outputTabList: document.querySelector("#outputTabList"),
  outputPanels: document.querySelector("#outputPanels"),
  sourcesDetails: document.querySelector("#sourcesDetails"),
  sourceCount: document.querySelector("#sourceCount"),
  sourceList: document.querySelector("#sourceList"),
};

let isGenerating = false;
let companionBusy = false;
let companionRevision = "";
let companionBriefDirty = false;
let growthSnapshot = null;
let growthStale = false;
let growthBusy = false;

/**
 * Side-panel runtime contracts (no method may return the API key):
 * - settings.get -> {ok:true, settings?:{apiKeyConfigured:boolean}, apiKeyConfigured?:boolean}
 * - settings.save {apiKey} -> {ok:true, settings?:{apiKeyConfigured:true}}
 * - settings.clear -> {ok:true}
 * - draft.save {draft} -> {ok:true}; draft excludes API credentials
 * - draft.load -> {ok:true, draft?:object}
 * - generate.facebookPack {brief,useGrounding}
 *   -> {ok:true,result:{pack,sources,usage,model}} or {ok:false,error}
 * - companion.syncBrief {brief} -> {ok:true,briefRevision,updatedAt}
 * - companion.getLatestPack {briefRevision?} -> {ok:true,found,stale,pack?,sources?,model?}
 * - companion.getLatestGrowthPack -> {ok:true,found,stale,snapshot?}; snapshot is strictly validated by the worker
 */

function setStatus(message, tone = "info") {
  elements.status.textContent = String(message || "");
  elements.status.className = `status status--${tone}`;
}

function errorMessage(error, fallback = "เกิดข้อผิดพลาดที่ไม่ทราบสาเหตุ") {
  if (typeof error === "string" && error.trim()) return error.trim();
  if (error && typeof error.message === "string" && error.message.trim()) return error.message.trim();
  if (error && typeof error.error === "string" && error.error.trim()) return error.error.trim();
  if (error && error.error && typeof error.error.message === "string") return error.error.message;
  return fallback;
}

function sendRuntimeMessage(message) {
  return new Promise((resolve, reject) => {
    if (!globalThis.chrome?.runtime?.sendMessage) {
      reject(new Error("เปิดหน้านี้ผ่าน Chrome Extension เพื่อใช้งาน"));
      return;
    }

    let settled = false;
    const succeed = (value) => {
      if (settled) return;
      settled = true;
      resolve(value);
    };
    const fail = (error) => {
      if (settled) return;
      settled = true;
      reject(error);
    };

    try {
      const possiblePromise = chrome.runtime.sendMessage(message, (response) => {
        const runtimeError = chrome.runtime.lastError;
        if (runtimeError) {
          fail(new Error(runtimeError.message));
          return;
        }
        succeed(response);
      });

      if (possiblePromise && typeof possiblePromise.then === "function") {
        possiblePromise.then(succeed, fail);
      }
    } catch (error) {
      fail(error);
    }
  });
}

function queryActiveTab() {
  return new Promise((resolve, reject) => {
    if (!globalThis.chrome?.tabs?.query) {
      reject(new Error("Chrome ไม่อนุญาตให้เข้าถึงแท็บปัจจุบัน"));
      return;
    }

    let settled = false;
    const succeed = (value) => {
      if (settled) return;
      settled = true;
      resolve(value);
    };
    const fail = (error) => {
      if (settled) return;
      settled = true;
      reject(error);
    };

    try {
      const possiblePromise = chrome.tabs.query({ active: true, currentWindow: true }, (tabs) => {
        const runtimeError = chrome.runtime.lastError;
        if (runtimeError) {
          fail(new Error(runtimeError.message));
          return;
        }
        succeed(tabs);
      });

      if (possiblePromise && typeof possiblePromise.then === "function") {
        possiblePromise.then(succeed, fail);
      }
    } catch (error) {
      fail(error);
    }
  });
}

function sendTabMessage(tabId, message) {
  return new Promise((resolve, reject) => {
    let settled = false;
    const succeed = (value) => {
      if (settled) return;
      settled = true;
      resolve(value);
    };
    const fail = (error) => {
      if (settled) return;
      settled = true;
      reject(error);
    };

    try {
      const possiblePromise = chrome.tabs.sendMessage(tabId, message, (response) => {
        const runtimeError = chrome.runtime.lastError;
        if (runtimeError) {
          fail(new Error(runtimeError.message));
          return;
        }
        succeed(response);
      });

      if (possiblePromise && typeof possiblePromise.then === "function") {
        possiblePromise.then(succeed, fail);
      }
    } catch (error) {
      fail(error);
    }
  });
}

function assertOk(response, fallback) {
  if (!response || response.ok !== true) {
    throw new Error(errorMessage(response, fallback));
  }
  return response;
}

function setKeyState(configured) {
  elements.keyState.dataset.state = configured ? "configured" : "missing";
  elements.keyState.textContent = configured ? "ตั้งค่าแล้ว" : "ยังไม่ได้ตั้งค่า";
}

function setCompanionState(state) {
  const connected = state === "connected";
  elements.companionState.dataset.state = connected ? "configured" : state === "checking" ? "checking" : "missing";
  elements.companionState.textContent = connected ? "เชื่อมแล้ว" : state === "checking" ? "กำลังตรวจ" : "ยังไม่ติดตั้ง";
}

function setCompanionBusy(busy) {
  companionBusy = busy;
  elements.fetchCompanionPack.disabled = busy;
  elements.syncCompanionBrief.disabled = busy;
}

async function refreshCompanionState() {
  setCompanionState("checking");
  try {
    assertOk(await sendRuntimeMessage({ type: "companion.health" }), "เชื่อม Companion ไม่สำเร็จ");
    setCompanionState("connected");
  } catch {
    setCompanionState("missing");
  }
}

function openSettings() {
  elements.settingsPanel.hidden = false;
  elements.settingsToggle.setAttribute("aria-expanded", "true");
  elements.settingsToggle.setAttribute("aria-label", "ปิดการตั้งค่า API key");
  elements.settingsPanel.scrollIntoView({ block: "nearest" });
  elements.apiKey.focus();
}

function closeSettings({ restoreFocus = true } = {}) {
  elements.settingsPanel.hidden = true;
  elements.settingsToggle.setAttribute("aria-expanded", "false");
  elements.settingsToggle.setAttribute("aria-label", "เปิดการตั้งค่า API key");
  elements.apiKey.value = "";
  if (restoreFocus) elements.settingsToggle.focus();
}

async function refreshSettingsState() {
  try {
    const response = assertOk(await sendRuntimeMessage({ type: "settings.get" }), "โหลดการตั้งค่าไม่สำเร็จ");
    const configured = Boolean(
      response.apiKeyConfigured ??
        response.settings?.apiKeyConfigured ??
        response.settings?.hasApiKey ??
        response.hasApiKey,
    );
    setKeyState(configured);
  } catch (error) {
    setKeyState(false);
    setStatus(errorMessage(error, "โหลดการตั้งค่าไม่สำเร็จ"), "error");
  }
}

async function saveApiKey() {
  const apiKey = elements.apiKey.value.trim();
  if (!apiKey) {
    setStatus("กรุณาใส่ API key ก่อนบันทึก", "error");
    elements.apiKey.focus();
    return;
  }

  elements.saveApiKey.disabled = true;
  try {
    assertOk(await sendRuntimeMessage({ type: "settings.save", apiKey }), "บันทึก API key ไม่สำเร็จ");
    elements.apiKey.value = "";
    setKeyState(true);
    setStatus("บันทึก API key สำหรับเซสชันนี้แล้ว", "success");
  } catch (error) {
    setStatus(errorMessage(error, "บันทึก API key ไม่สำเร็จ"), "error");
  } finally {
    elements.saveApiKey.disabled = false;
  }
}

async function clearApiKey() {
  elements.clearApiKey.disabled = true;
  try {
    assertOk(await sendRuntimeMessage({ type: "settings.clear" }), "ล้าง API key ไม่สำเร็จ");
    elements.apiKey.value = "";
    setKeyState(false);
    setStatus("ล้าง API key ออกจากเซสชันแล้ว", "success");
  } catch (error) {
    setStatus(errorMessage(error, "ล้าง API key ไม่สำเร็จ"), "error");
  } finally {
    elements.clearApiKey.disabled = false;
  }
}

function isHttpUrl(value) {
  try {
    const url = new URL(value);
    return url.protocol === "http:" || url.protocol === "https:";
  } catch {
    return false;
  }
}

function isFacebookUrl(value) {
  try {
    const url = new URL(value);
    const hostname = url.hostname.toLowerCase();
    return url.protocol === "https:" && (hostname === "facebook.com" || hostname.endsWith(".facebook.com"));
  } catch {
    return false;
  }
}

function parseEvidence(rawValue) {
  return String(rawValue || "")
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line, index) => {
      const parts = line
        .split("|")
        .map((part) => part.trim())
        .filter(Boolean);
      const url = parts.find(isHttpUrl);
      const nonUrlParts = parts.filter((part) => part !== url);
      const hasStructuredTitle = Boolean(url && nonUrlParts.length > 1);
      const title = hasStructuredTitle ? nonUrlParts[0] : `ข้อมูลยืนยัน ${index + 1}`;
      const notesParts = hasStructuredTitle ? nonUrlParts.slice(1) : nonUrlParts;

      return {
        id: `S${index + 1}`,
        title,
        ...(url ? { url } : {}),
        notes: notesParts.join(" | ") || line,
      };
    });
}

function collectDraft() {
  return {
    campaignProduct: elements.campaignProduct.value,
    audience: elements.audience.value,
    objective: elements.objective.value,
    offer: elements.offer.value,
    brandVoice: elements.brandVoice.value,
    evidence: elements.evidence.value,
    constraints: elements.constraints.value,
    useGrounding: elements.useGrounding.checked,
  };
}

function applyDraft(draft) {
  if (!draft || typeof draft !== "object") return false;
  const textFields = ["campaignProduct", "audience", "objective", "offer", "brandVoice", "evidence", "constraints"];

  for (const field of textFields) {
    if (typeof draft[field] === "string") elements[field].value = draft[field];
  }
  if (typeof draft.useGrounding === "boolean") elements.useGrounding.checked = draft.useGrounding;
  return true;
}

function buildBrief() {
  const campaignProduct = elements.campaignProduct.value.trim();
  const constraints = elements.constraints.value.trim();
  const verifiedFacts = elements.evidence.value.trim();

  return {
    topic: campaignProduct,
    productDetails: campaignProduct,
    audience: elements.audience.value.trim(),
    objective: elements.objective.value.trim(),
    offer: elements.offer.value.trim(),
    brandVoice: elements.brandVoice.value.trim(),
    language: "th",
    evidence: parseEvidence(verifiedFacts),
    additionalInstructions: constraints,
  };
}

function currentBriefFingerprint() {
  return JSON.stringify(buildBrief());
}

function invalidateResultsAfterBriefChange(event) {
  const companionBriefChanged = !event || event.target !== elements.useGrounding;
  const hadReusableResult = companionRevision !== "" || !elements.resultsSection.hidden || isGenerating || companionBusy;
  if (companionBriefChanged) {
    companionRevision = "";
    companionBriefDirty = true;
  }
  elements.resultsSection.hidden = true;
  if (hadReusableResult) {
    setStatus("Brief เปลี่ยนแล้ว ผลเดิมถูกพักไว้ กรุณาสร้างหรือซิงก์เวอร์ชันล่าสุดก่อนใช้", "info");
  }
}

function validateCompanionBrief() {
  elements.evidence.setCustomValidity("");
  if (!elements.briefForm.reportValidity()) return false;
  if (parseEvidence(elements.evidence.value).length > 30) {
    elements.evidence.setCustomValidity("ใส่หลักฐานได้สูงสุด 30 รายการ");
    elements.evidence.reportValidity();
    elements.evidence.focus();
    return false;
  }
  return true;
}

async function syncBriefToCompanion() {
  if (companionBusy || !validateCompanionBrief()) return;
  const brief = buildBrief();
  const submittedFingerprint = JSON.stringify(brief);
  setCompanionBusy(true);
  setStatus("กำลังส่ง brief เข้า Content Blueprint Companion", "info");
  try {
    const response = assertOk(
      await sendRuntimeMessage({ type: "companion.syncBrief", brief }),
      "ส่ง brief เข้า Companion ไม่สำเร็จ",
    );
    if (currentBriefFingerprint() !== submittedFingerprint) {
      companionRevision = "";
      companionBriefDirty = true;
      elements.resultsSection.hidden = true;
      setStatus("Brief ถูกแก้ระหว่างซิงก์ กรุณาซิงก์อีกครั้งเพื่อส่งเวอร์ชันล่าสุด", "info");
      return;
    }
    companionRevision = typeof response.briefRevision === "string" ? response.briefRevision : "";
    companionBriefDirty = false;
    setCompanionState("connected");
    setStatus("ส่ง brief แล้ว เปิด Claude/Codex และสั่งให้สร้าง Facebook Content Pack จาก brief ล่าสุด จากนั้นกดรับผลล่าสุด", "success");
  } catch (error) {
    setCompanionState("missing");
    setStatus(errorMessage(error, "ส่ง brief เข้า Companion ไม่สำเร็จ"), "error");
  } finally {
    setCompanionBusy(false);
  }
}

async function fetchLatestCompanionPack() {
  if (companionBusy) return;
  if (companionBriefDirty) {
    elements.resultsSection.hidden = true;
    setStatus("Brief ถูกแก้แล้ว กรุณากดส่งโจทย์เข้า MCP ก่อนรับผล เพื่อไม่ใช้ Content Pack ผิดเวอร์ชัน", "error");
    return;
  }
  const requestedFingerprint = currentBriefFingerprint();
  setCompanionBusy(true);
  setStatus("กำลังรับ Content Pack ล่าสุดจากโปรแกรม", "info");
  try {
    const request = { type: "companion.getLatestPack" };
    if (companionRevision) request.briefRevision = companionRevision;
    const response = assertOk(await sendRuntimeMessage(request), "รับผลจาก Companion ไม่สำเร็จ");
    setCompanionState("connected");
    if (companionBriefDirty || currentBriefFingerprint() !== requestedFingerprint) {
      elements.resultsSection.hidden = true;
      setStatus("Brief ถูกแก้ระหว่างรับผล กรุณาซิงก์เวอร์ชันล่าสุดก่อนลองอีกครั้ง", "error");
      return;
    }
    if (response.found !== true) {
      setStatus("ยังไม่มี Content Pack ใน Companion ให้สร้างจาก Wails/Claude/Codex ก่อนแล้วลองใหม่", "info");
      return;
    }
    if (response.stale === true) {
      setStatus("ผลล่าสุดเป็นของ brief คนละชุด กรุณาสร้างจาก brief ปัจจุบันก่อนรับผล", "error");
      return;
    }
    if (!response.pack && !response.content) {
      throw new Error("ผลลัพธ์ไม่มี Facebook Content Pack");
    }
    renderResults(response);
    setStatus("รับ Content Pack แล้ว กรุณาตรวจทานก่อนคัดลอกหรือแทรกใน Facebook", "success");
  } catch (error) {
    setStatus(errorMessage(error, "รับผลจาก Companion ไม่สำเร็จ"), "error");
  } finally {
    setCompanionBusy(false);
  }
}

async function saveDraft() {
  elements.saveDraft.disabled = true;
  try {
    assertOk(
      await sendRuntimeMessage({ type: "draft.save", draft: collectDraft() }),
      "บันทึก Draft ไม่สำเร็จ",
    );
    setStatus("บันทึก Draft ที่ไม่รวม API key แล้ว", "success");
  } catch (error) {
    setStatus(errorMessage(error, "บันทึก Draft ไม่สำเร็จ"), "error");
  } finally {
    elements.saveDraft.disabled = false;
  }
}

async function loadDraft() {
  elements.loadDraft.disabled = true;
  try {
    const response = assertOk(await sendRuntimeMessage({ type: "draft.load" }), "โหลด Draft ไม่สำเร็จ");
    const draft = response.draft ?? response.state?.draft ?? response.result?.draft;
    if (!applyDraft(draft)) {
      setStatus("ยังไม่มี Draft ที่บันทึกไว้", "info");
      return;
    }
    invalidateResultsAfterBriefChange();
    setStatus("โหลด Draft แล้ว", "success");
    elements.campaignProduct.focus();
  } catch (error) {
    setStatus(errorMessage(error, "โหลด Draft ไม่สำเร็จ"), "error");
  } finally {
    elements.loadDraft.disabled = false;
  }
}

function setGenerating(busy) {
  isGenerating = busy;
  elements.generateButton.disabled = busy;
  elements.generateSpinner.hidden = !busy;
  elements.generateLabel.textContent = busy ? "กำลังสร้างชุดคอนเทนต์..." : "สร้าง Facebook Content Pack";
  elements.resultsSection.setAttribute("aria-busy", String(busy));
}

function pickValue(source, aliases) {
  if (!source || typeof source !== "object") return undefined;
  for (const alias of aliases) {
    if (source[alias] !== undefined && source[alias] !== null) return source[alias];
  }
  return undefined;
}

function humanizeKey(key) {
  const labels = {
    title: "หัวข้อ",
    heading: "หัวข้อ",
    headline: "พาดหัว",
    body: "เนื้อหา",
    text: "ข้อความ",
    caption: "แคปชัน",
    visual: "แนวภาพ",
    scene: "ฉาก",
    voiceover: "เสียงบรรยาย",
    reply: "คำตอบ",
    question: "คำถาม",
    objection: "ข้อกังวล",
    risk: "ความเสี่ยง",
    recommendation: "คำแนะนำ",
  };
  if (labels[key]) return labels[key];
  return String(key)
    .replace(/[_-]+/g, " ")
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .trim();
}

function stringifyPrimitive(value) {
  if (typeof value === "string") return value.trim();
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return "";
}

function objectToText(value) {
  if (!value || typeof value !== "object" || Array.isArray(value)) return stringifyPrimitive(value);
  const lines = [];
  for (const [key, nestedValue] of Object.entries(value)) {
    const primitive = stringifyPrimitive(nestedValue);
    if (primitive) {
      lines.push(`${humanizeKey(key)}: ${primitive}`);
    } else if (Array.isArray(nestedValue)) {
      const nestedLines = nestedValue.map(stringifyPrimitive).filter(Boolean);
      if (nestedLines.length) lines.push(`${humanizeKey(key)}:\n${nestedLines.join("\n")}`);
    }
  }
  return lines.join("\n").trim();
}

function normalizeEntries(value, baseLabel) {
  if (value === undefined || value === null) return [];

  const primitive = stringifyPrimitive(value);
  if (primitive) return [{ label: baseLabel, text: primitive }];

  if (Array.isArray(value)) {
    return value.flatMap((item, index) => {
      const label = `${baseLabel} ${index + 1}`;
      const itemPrimitive = stringifyPrimitive(item);
      if (itemPrimitive) return [{ label, text: itemPrimitive }];
      const itemText = objectToText(item);
      if (itemText) return [{ label, text: itemText }];
      return normalizeEntries(item, label);
    });
  }

  if (typeof value === "object") {
    const grouped = objectToText(value);
    if (grouped) return [{ label: baseLabel, text: grouped }];

    return Object.entries(value).flatMap(([key, nestedValue]) =>
      normalizeEntries(nestedValue, humanizeKey(key) || baseLabel),
    );
  }

  return [];
}

function createActionButton(label, className, handler) {
  const button = document.createElement("button");
  button.type = "button";
  button.className = className;
  button.textContent = label;
  button.addEventListener("click", handler);
  return button;
}

async function copyText(text, trigger) {
  trigger.disabled = true;
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
    } else {
      const textarea = document.createElement("textarea");
      textarea.value = text;
      textarea.setAttribute("readonly", "");
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      document.body.append(textarea);
      textarea.select();
      const copied = document.execCommand("copy");
      textarea.remove();
      if (!copied) throw new Error("คัดลอกไม่สำเร็จ");
    }
    setStatus("คัดลอกข้อความแล้ว", "success");
  } catch (error) {
    setStatus(errorMessage(error, "คัดลอกข้อความไม่สำเร็จ"), "error");
  } finally {
    trigger.disabled = false;
  }
}

async function insertIntoFacebook(text, trigger) {
  trigger.disabled = true;
  try {
    const tabs = await queryActiveTab();
    const activeTab = Array.isArray(tabs) ? tabs[0] : undefined;
    if (!activeTab?.id) {
      throw new Error("ไม่พบแท็บ Facebook ที่ใช้งานอยู่");
    }
    if (activeTab.url && !isFacebookUrl(activeTab.url)) {
      throw new Error("กรุณาเปิด facebook.com และคลิกช่องเขียนโพสต์หรือคอมเมนต์ก่อน");
    }

    const response = await sendTabMessage(activeTab.id, { type: "facebook.insert", text });
    assertOk(response, "ไม่พบช่อง Facebook ที่พร้อมรับข้อความ");
    setStatus("แทรกข้อความแล้ว กรุณาตรวจทานและกดโพสต์ด้วยตัวเอง", "success");
  } catch (error) {
    setStatus(errorMessage(error, "แทรกข้อความใน Facebook ไม่สำเร็จ"), "error");
  } finally {
    trigger.disabled = false;
  }
}

function createOutputCard(entry) {
  const article = document.createElement("article");
  article.className = "output-card";

  const heading = document.createElement("h3");
  heading.className = "output-card-heading";
  heading.textContent = entry.label;

  const text = document.createElement("pre");
  text.className = "output-text";
  text.tabIndex = 0;
  text.textContent = entry.text;

  const actions = document.createElement("div");
  actions.className = "output-actions";
  const copyButton = createActionButton("คัดลอก", "output-action", () => copyText(entry.text, copyButton));
  copyButton.setAttribute("aria-label", `คัดลอก ${entry.label}`);
  const insertButton = createActionButton(
    "แทรกใน Facebook",
    "output-action output-action--insert",
    () => insertIntoFacebook(entry.text, insertButton),
  );
  insertButton.setAttribute("aria-label", `แทรก ${entry.label} ในช่อง Facebook ที่โฟกัสล่าสุด`);
  actions.append(copyButton, insertButton);

  article.append(heading, text, actions);
  return article;
}

function createTab(section, index) {
  const button = document.createElement("button");
  button.type = "button";
  button.className = "tab-button";
  button.id = `output-tab-${section.key}`;
  button.setAttribute("role", "tab");
  button.setAttribute("aria-controls", `output-panel-${section.key}`);
  button.setAttribute("aria-selected", index === 0 ? "true" : "false");
  button.tabIndex = index === 0 ? 0 : -1;
  button.textContent = section.label;
  button.title = section.label;
  button.addEventListener("click", () => activateTab(index, false));
  button.addEventListener("keydown", handleTabKeydown);
  return button;
}

function createTabPanel(section, entries, index) {
  const panel = document.createElement("section");
  panel.className = "tab-panel";
  panel.id = `output-panel-${section.key}`;
  panel.setAttribute("role", "tabpanel");
  panel.setAttribute("aria-labelledby", `output-tab-${section.key}`);
  panel.tabIndex = 0;
  panel.hidden = index !== 0;

  if (!entries.length) {
    const empty = document.createElement("p");
    empty.className = "empty-output";
    empty.textContent = "โมเดลไม่ได้ส่งเนื้อหาในส่วนนี้กลับมา";
    panel.append(empty);
    return panel;
  }

  const list = document.createElement("div");
  list.className = "output-list";
  for (const entry of entries) list.append(createOutputCard(entry));
  panel.append(list);
  return panel;
}

function activateTab(index, shouldFocus = true) {
  const tabs = [...elements.outputTabList.querySelectorAll('[role="tab"]')];
  const panels = [...elements.outputPanels.querySelectorAll('[role="tabpanel"]')];
  if (!tabs[index] || !panels[index]) return;

  tabs.forEach((tab, tabIndex) => {
    const selected = tabIndex === index;
    tab.setAttribute("aria-selected", String(selected));
    tab.tabIndex = selected ? 0 : -1;
  });
  panels.forEach((panel, panelIndex) => {
    panel.hidden = panelIndex !== index;
  });
  if (shouldFocus) tabs[index].focus();
}

function handleTabKeydown(event) {
  const tabs = [...elements.outputTabList.querySelectorAll('[role="tab"]')];
  const currentIndex = tabs.indexOf(event.currentTarget);
  if (currentIndex < 0) return;

  let nextIndex;
  if (event.key === "ArrowRight" || event.key === "ArrowDown") nextIndex = (currentIndex + 1) % tabs.length;
  if (event.key === "ArrowLeft" || event.key === "ArrowUp") nextIndex = (currentIndex - 1 + tabs.length) % tabs.length;
  if (event.key === "Home") nextIndex = 0;
  if (event.key === "End") nextIndex = tabs.length - 1;
  if (nextIndex === undefined) return;

  event.preventDefault();
  activateTab(nextIndex, true);
}

function formatUsage(usage) {
  if (!usage || typeof usage !== "object") return "";
  const aliases = [
    ["inputTokens", "input_tokens", "promptTokenCount", "prompt_tokens"],
    ["outputTokens", "output_tokens", "candidatesTokenCount", "completion_tokens"],
    ["totalTokens", "total_tokens", "totalTokenCount"],
  ];
  const labels = ["input", "output", "รวม"];
  const parts = [];
  aliases.forEach((keys, index) => {
    const value = keys.map((key) => usage[key]).find((candidate) => Number.isFinite(Number(candidate)));
    if (value !== undefined) parts.push(`${labels[index]} ${Number(value).toLocaleString("th-TH")}`);
  });
  return parts.join(" · ");
}

function renderMeta(model, usage) {
  elements.resultMeta.replaceChildren();
  const values = [];
  if (typeof model === "string" && model.trim()) values.push(`โมเดล: ${model.trim()}`);
  const usageText = formatUsage(usage);
  if (usageText) values.push(`โทเคน: ${usageText}`);

  for (const value of values) {
    const chip = document.createElement("span");
    chip.className = "meta-chip";
    chip.textContent = value;
    chip.title = value;
    elements.resultMeta.append(chip);
  }
}

function normalizeSources(sources) {
  if (!Array.isArray(sources)) return [];
  const seen = new Set();
  const normalized = [];
  for (const source of sources) {
    const url = typeof source === "string" ? source : source?.url;
    if (!isHttpUrl(url) || seen.has(url)) continue;
    seen.add(url);
    const title = typeof source?.title === "string" && source.title.trim() ? source.title.trim() : url;
    normalized.push({ title, url });
  }
  return normalized;
}

function renderSources(sources) {
  const normalized = normalizeSources(sources);
  elements.sourceList.replaceChildren();
  elements.sourceCount.textContent = normalized.length ? `(${normalized.length})` : "";
  elements.sourcesDetails.hidden = normalized.length === 0;
  elements.sourcesDetails.open = false;

  for (const source of normalized) {
    const item = document.createElement("li");
    const link = document.createElement("a");
    link.href = source.url;
    link.target = "_blank";
    link.rel = "noopener noreferrer";
    link.textContent = source.title;
    link.title = source.url;
    item.append(link);
    elements.sourceList.append(item);
  }
}

function renderResults(result) {
  const pack = result?.pack ?? result?.content ?? result;
  elements.outputTabList.replaceChildren();
  elements.outputPanels.replaceChildren();

  OUTPUT_SECTIONS.forEach((section, index) => {
    const value = pickValue(pack, section.aliases);
    const entries = normalizeEntries(value, section.itemLabel);
    elements.outputTabList.append(createTab(section, index));
    elements.outputPanels.append(createTabPanel(section, entries, index));
  });

  renderMeta(result?.model, result?.usage);
  renderSources(result?.sources ?? result?.groundingSources);
  elements.resultsSection.hidden = false;
  activateTab(0, false);
  elements.resultsSection.scrollIntoView({ block: "start" });
  document.querySelector("#output-tab-hooks")?.focus({ preventScroll: true });
}

function growthInsertAllowed() {
  if (!growthSnapshot) return { allowed: false, reason: "ยังไม่ได้โหลด Growth Pack" };
  const guard = growthInsertGuard(growthSnapshot, growthStale, elements.growthPendingConfirm.checked);
  if (guard.allowed) return guard;
  if (growthStale) return { ...guard, reason: "Brief เปลี่ยนแล้ว กรุณาสร้าง Growth Pack ใหม่ก่อนแทรก" };
  if (growthSnapshot.reviewStatus === "rejected") {
    return { ...guard, reason: "Growth Pack นี้ถูกปฏิเสธ จึงแทรกข้อความไม่ได้" };
  }
  return { ...guard, reason: "ตรวจข้อความและติ๊กยืนยันก่อนแทรก" };
}

function updateGrowthInsertControls() {
  const guard = growthInsertAllowed();
  const fullPackLength = growthSnapshot ? growthPackToPlainText(growthSnapshot).length : 0;
  elements.growthInsertAll.disabled = !guard.allowed || fullPackLength > 50_000;
  for (const button of elements.growthBlocks.querySelectorAll("[data-growth-insert]")) {
    button.disabled = !guard.allowed || Number(button.dataset.textLength) > 50_000;
  }
}

async function insertGrowthText(text, trigger) {
  const guard = growthInsertAllowed();
  if (!guard.allowed) {
    setStatus(guard.reason, "error");
    return;
  }
  if (text.length > 50_000) {
    setStatus("ข้อความยาวเกินขนาดที่แทรกได้ กรุณาใช้ปุ่มคัดลอกแทน", "error");
    return;
  }
  await insertIntoFacebook(text, trigger);
  updateGrowthInsertControls();
}

function renderGrowthBlock(block) {
  const article = document.createElement("article");
  article.className = "growth-block";
  const heading = document.createElement("h3");
  heading.textContent = block.title;
  const purpose = document.createElement("p");
  purpose.className = "helper-text";
  purpose.textContent = block.purpose;
  const basis = document.createElement("span");
  basis.className = "growth-basis";
  basis.textContent = block.evidenceBasis.replaceAll("_", " ");
  article.append(heading, purpose, basis);

  if (block.kind === "prose") {
    const body = document.createElement("pre");
    body.className = "growth-text";
    body.textContent = block.body;
    article.append(body);
  } else if (block.kind === "code") {
    const code = document.createElement("pre");
    code.className = "growth-code";
    code.textContent = block.code;
    article.append(code);
  } else if (block.kind === "table") {
    const wrap = document.createElement("div");
    wrap.className = "growth-table-wrap";
    const table = document.createElement("table");
    table.className = "growth-table";
    const head = table.createTHead().insertRow();
    for (const column of block.columns) {
      const cell = document.createElement("th");
      cell.scope = "col";
      cell.textContent = column;
      head.append(cell);
    }
    const body = table.createTBody();
    for (const row of block.rows) {
      const tr = body.insertRow();
      for (const value of row) {
        const cell = tr.insertCell();
        cell.textContent = value;
      }
    }
    wrap.append(table);
    article.append(wrap);
  } else {
    const list = document.createElement(block.kind === "checklist" ? "ul" : "ol");
    list.className = "growth-list";
    for (const item of block.items) {
      const entry = document.createElement("li");
      const label = document.createElement("strong");
      label.textContent = `${item.label}: `;
      entry.append(label, document.createTextNode(item.value));
      if (item.note) entry.append(document.createTextNode(` (${item.note})`));
      list.append(entry);
    }
    article.append(list);
  }

  const plainText = growthBlockToPlainText(block);
  const actions = document.createElement("div");
  actions.className = "button-row growth-block-actions";
  const copyButton = createActionButton("คัดลอกส่วนนี้", "button button--quiet", () => copyText(plainText, copyButton));
  const insertButton = createActionButton("แทรกในช่องที่โฟกัส", "button button--quiet", () => insertGrowthText(plainText, insertButton));
  insertButton.dataset.growthInsert = "true";
  insertButton.dataset.textLength = String(plainText.length);
  actions.append(copyButton, insertButton);
  article.append(actions);
  return article;
}

function renderGrowthPack(response) {
  growthSnapshot = response.snapshot;
  growthStale = response.stale === true;
  elements.growthPendingConfirm.checked = false;
  elements.growthResults.hidden = false;
  elements.growthBlocks.replaceChildren(...growthSnapshot.pack.blocks.map(renderGrowthBlock));
  elements.growthMeta.textContent = `${growthSnapshot.pack.title} · ${growthSnapshot.playbookId} · ${growthSnapshot.reviewStatus}`;
  elements.growthConfirmWrap.hidden = growthSnapshot.reviewStatus !== "needs_review" || growthStale;
  elements.growthWarning.className = "growth-warning";
  if (growthStale) {
    elements.growthWarning.classList.add("growth-warning--blocked");
    elements.growthWarning.textContent = "Brief เปลี่ยนหลังสร้างร่าง กรุณาสร้างใหม่ก่อนแทรก";
  } else if (growthSnapshot.reviewStatus === "rejected") {
    elements.growthWarning.classList.add("growth-warning--blocked");
    elements.growthWarning.textContent = "ร่างนี้ถูกปฏิเสธ อ่านหรือคัดลอกได้ แต่แทรกไม่ได้";
  } else if (growthSnapshot.reviewStatus === "needs_review") {
    elements.growthWarning.classList.add("growth-warning--review");
    elements.growthWarning.textContent = "ต้องตรวจทาน: เช็กทุกข้อกล่าวอ้าง แล้วติ๊กยืนยันก่อนแทรก";
  } else {
    elements.growthWarning.textContent = "อนุมัติแล้ว ตรวจช่องปลายทางก่อนแทรก และกดโพสต์ด้วยตัวเอง";
  }
  elements.growthState.dataset.state = growthStale ? "missing" : "configured";
  elements.growthState.textContent = growthStale
    ? "ข้อมูลเก่า"
    : ({ needs_review: "รอตรวจ", approved: "อนุมัติแล้ว", rejected: "ปฏิเสธแล้ว" })[growthSnapshot.reviewStatus];
  updateGrowthInsertControls();
}

async function fetchLatestGrowthPack() {
  if (growthBusy) return;
  growthBusy = true;
  elements.growthFetch.disabled = true;
  elements.growthState.dataset.state = "checking";
  elements.growthState.textContent = "กำลังโหลด";
  try {
    const response = assertOk(await sendRuntimeMessage({ type: "companion.getLatestGrowthPack" }), "รับ Growth Pack ไม่สำเร็จ");
    if (!response.found || !response.snapshot) {
      growthSnapshot = null;
      elements.growthResults.hidden = true;
      elements.growthState.dataset.state = "missing";
      elements.growthState.textContent = "ยังไม่มีผลลัพธ์";
      setStatus("ยังไม่มี Growth Pack ในเครื่อง กรุณาสร้างใน Content Blueprint ก่อน", "info");
      return;
    }
    renderGrowthPack(response);
    setStatus("รับ Growth Pack จากเครื่องแล้ว ยังไม่มีการโพสต์หรือส่งข้อความ", "success");
  } catch (error) {
    growthSnapshot = null;
    elements.growthResults.hidden = true;
    elements.growthState.dataset.state = "missing";
    elements.growthState.textContent = "ใช้งานไม่ได้";
    setStatus(errorMessage(error, "รับ Growth Pack ไม่สำเร็จ"), "error");
  } finally {
    growthBusy = false;
    elements.growthFetch.disabled = false;
  }
}

async function generatePack(event) {
  event.preventDefault();
  if (isGenerating) return;
  elements.evidence.setCustomValidity("");
  if (!elements.briefForm.reportValidity()) return;
  if (parseEvidence(elements.evidence.value).length > 30) {
    elements.evidence.setCustomValidity("ใส่หลักฐานได้สูงสุด 30 รายการ");
    elements.evidence.reportValidity();
    elements.evidence.focus();
    return;
  }
  if (elements.keyState.dataset.state !== "configured") {
    setStatus("ตั้งค่า Gemini API key สำหรับเซสชันนี้ก่อนสร้างคอนเทนต์", "error");
    openSettings();
    return;
  }

  const brief = buildBrief();
  const submittedFingerprint = JSON.stringify({
    brief,
    useGrounding: elements.useGrounding.checked,
  });
  setGenerating(true);
  setStatus("กำลังสร้างและตรวจชุดคอนเทนต์ กรุณารอสักครู่", "info");
  try {
    const response = assertOk(
      await sendRuntimeMessage({
        type: "generate.facebookPack",
        brief,
        useGrounding: elements.useGrounding.checked,
      }),
      "สร้างชุดคอนเทนต์ไม่สำเร็จ",
    );
    const result = response.result ?? response;
    if (!result.pack && !result.content) throw new Error("ผลลัพธ์ไม่มี Facebook Content Pack");
    const currentFingerprint = JSON.stringify({
      brief: buildBrief(),
      useGrounding: elements.useGrounding.checked,
    });
    if (currentFingerprint !== submittedFingerprint) {
      elements.resultsSection.hidden = true;
      setStatus("Brief ถูกแก้ระหว่างสร้าง จึงพักผลเวอร์ชันเก่าไว้ กรุณาสร้างใหม่", "info");
      return;
    }
    renderResults(result);
    setStatus("สร้างชุดคอนเทนต์แล้ว เลือกตรวจทาน คัดลอก หรือแทรกใน Facebook ได้เลย", "success");
  } catch (error) {
    setStatus(errorMessage(error, "สร้างชุดคอนเทนต์ไม่สำเร็จ"), "error");
  } finally {
    setGenerating(false);
  }
}

elements.settingsToggle.addEventListener("click", () => {
  if (elements.settingsPanel.hidden) openSettings();
  else closeSettings();
});
elements.closeSettings.addEventListener("click", () => closeSettings());
elements.saveApiKey.addEventListener("click", saveApiKey);
elements.clearApiKey.addEventListener("click", clearApiKey);
elements.apiKey.addEventListener("keydown", (event) => {
  if (event.key === "Enter") {
    event.preventDefault();
    saveApiKey();
  }
});
elements.saveDraft.addEventListener("click", saveDraft);
elements.loadDraft.addEventListener("click", loadDraft);
elements.fetchCompanionPack.addEventListener("click", fetchLatestCompanionPack);
elements.syncCompanionBrief.addEventListener("click", syncBriefToCompanion);
elements.growthFetch.addEventListener("click", fetchLatestGrowthPack);
elements.growthPendingConfirm.addEventListener("change", updateGrowthInsertControls);
elements.growthCopyAll.addEventListener("click", () => {
  if (growthSnapshot) copyText(growthPackToPlainText(growthSnapshot), elements.growthCopyAll);
});
elements.growthInsertAll.addEventListener("click", () => {
  if (growthSnapshot) insertGrowthText(growthPackToPlainText(growthSnapshot), elements.growthInsertAll);
});
elements.briefForm.addEventListener("submit", generatePack);
elements.briefForm.addEventListener("input", invalidateResultsAfterBriefChange);
elements.briefForm.addEventListener("change", invalidateResultsAfterBriefChange);
elements.evidence.addEventListener("input", () => elements.evidence.setCustomValidity(""));

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape" && !elements.settingsPanel.hidden) closeSettings();
});

refreshSettingsState();
refreshCompanionState();
