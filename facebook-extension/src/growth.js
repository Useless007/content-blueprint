const ID_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$/u;
const BLOCK_KINDS = new Set(["prose", "cards", "table", "sequence", "checklist", "tasks", "code"]);
const EVIDENCE_BASES = new Set(["user_input", "supplied_evidence", "ai_inference", "imported_metric", "mixed"]);
const REVIEW_STATUSES = new Set(["needs_review", "approved", "rejected"]);
const CHECK_STATUSES = new Set(["ready", "review", "blocked"]);
const PLAYBOOK_IDS = new Set([
  "offer-audience",
  "facebook-campaign",
  "sales-reply",
  "seo-topic-map",
  "seo-content-brief",
  "seo-onpage-review",
  "seo-internal-links",
  "seo-structured-data",
  "seo-search-console-opportunities",
  "cross-channel-repurpose",
]);

export class GrowthValidationError extends Error {
  constructor(message) {
    super(message);
    this.name = "GrowthValidationError";
  }
}

function plainObject(value) {
  if (value === null || typeof value !== "object" || Array.isArray(value)) return false;
  const prototype = Object.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}

function exactKeys(value, required, optional = [], path = "value") {
  if (!plainObject(value)) throw new GrowthValidationError(`${path} must be an object`);
  const allowed = new Set([...required, ...optional]);
  for (const key of Object.keys(value)) {
    if (!allowed.has(key)) throw new GrowthValidationError(`${path} contains unsupported field ${key}`);
  }
  for (const key of required) {
    if (!Object.prototype.hasOwnProperty.call(value, key)) throw new GrowthValidationError(`${path}.${key} is required`);
  }
}

function text(value, path, max, { required = true } = {}) {
  if (typeof value !== "string") throw new GrowthValidationError(`${path} must be text`);
  const result = value.trim();
  if (required && result === "") throw new GrowthValidationError(`${path} is required`);
  if (result.length > max || result.includes("\u0000")) throw new GrowthValidationError(`${path} is invalid or too long`);
  return result;
}

function textArray(value, path, maxItems, maxChars) {
  if (!Array.isArray(value) || value.length > maxItems) throw new GrowthValidationError(`${path} is not a bounded array`);
  return value.map((item, index) => text(item, `${path}[${index}]`, maxChars));
}

function validateItem(value, path) {
  exactKeys(value, ["label", "value"], ["note"], path);
  return {
    label: text(value.label, `${path}.label`, 1_000),
    value: text(value.value, `${path}.value`, 8_000),
    note: text(value.note ?? "", `${path}.note`, 2_000, { required: false }),
  };
}

function validateBlock(value, index, sourceIDs, playbookId) {
  const path = `blocks[${index}]`;
  exactKeys(value, ["id", "title", "purpose", "kind", "body", "items", "columns", "rows", "code", "evidenceBasis", "sourceIds"], [], path);
  const id = text(value.id, `${path}.id`, 128);
  if (!ID_PATTERN.test(id)) throw new GrowthValidationError(`${path}.id is invalid`);
  const kind = text(value.kind, `${path}.kind`, 20);
  if (!BLOCK_KINDS.has(kind)) throw new GrowthValidationError(`${path}.kind is invalid`);
  const evidenceBasis = text(value.evidenceBasis, `${path}.evidenceBasis`, 30);
  if (!EVIDENCE_BASES.has(evidenceBasis)) throw new GrowthValidationError(`${path}.evidenceBasis is invalid`);
  if (evidenceBasis === "imported_metric" && playbookId !== "seo-search-console-opportunities") {
    throw new GrowthValidationError(`${path} uses imported_metric outside the Search Console playbook`);
  }
  const blockSources = textArray(value.sourceIds, `${path}.sourceIds`, 30, 128);
  if ((evidenceBasis === "supplied_evidence" || evidenceBasis === "mixed") && blockSources.length === 0) {
    throw new GrowthValidationError(`${path}.sourceIds is required for its evidence basis`);
  }
  const seenSources = new Set();
  for (const sourceId of blockSources) {
    if (!sourceIDs.has(sourceId) || seenSources.has(sourceId)) throw new GrowthValidationError(`${path}.sourceIds is invalid`);
    seenSources.add(sourceId);
  }
  const body = text(value.body, `${path}.body`, 50_000, { required: false });
  const code = text(value.code, `${path}.code`, 100_000, { required: false });
  if (!Array.isArray(value.items) || value.items.length > 50) throw new GrowthValidationError(`${path}.items is invalid`);
  const items = value.items.map((item, itemIndex) => validateItem(item, `${path}.items[${itemIndex}]`));
  const columns = textArray(value.columns, `${path}.columns`, 20, 500);
  if (!Array.isArray(value.rows) || value.rows.length > 100) throw new GrowthValidationError(`${path}.rows is invalid`);
  const rows = value.rows.map((row, rowIndex) => {
    if (!Array.isArray(row) || row.length !== columns.length) throw new GrowthValidationError(`${path}.rows[${rowIndex}] does not match columns`);
    return row.map((cell, cellIndex) => text(cell, `${path}.rows[${rowIndex}][${cellIndex}]`, 4_000, { required: false }));
  });
  const hasBody = body !== "";
  const hasItems = items.length > 0;
  const hasTable = columns.length > 0 && rows.length > 0;
  const hasCode = code !== "";
  if (kind === "prose" && (!hasBody || hasItems || hasTable || hasCode)) throw new GrowthValidationError(`${path} has an invalid prose shape`);
  if (["cards", "sequence", "checklist", "tasks"].includes(kind) && (!hasItems || hasBody || hasTable || hasCode)) throw new GrowthValidationError(`${path} has an invalid item shape`);
  if (kind === "table" && (!hasTable || hasBody || hasItems || hasCode)) throw new GrowthValidationError(`${path} has an invalid table shape`);
  if (kind === "code" && (!hasCode || hasBody || hasItems || hasTable)) throw new GrowthValidationError(`${path} has an invalid code shape`);
  return {
    id,
    title: text(value.title, `${path}.title`, 1_000),
    purpose: text(value.purpose, `${path}.purpose`, 2_000),
    kind,
    body,
    items,
    columns,
    rows,
    code,
    evidenceBasis,
    sourceIds: blockSources,
  };
}

export function validateGrowthSnapshot(value) {
  exactKeys(value, ["version", "briefRevision", "playbookId", "evidenceSourceIds", "pack", "generatedBy", "updatedAt", "reviewStatus"], ["reviewerNote", "reviewUpdatedAt"], "growthSnapshot");
  if (value.version !== 1) throw new GrowthValidationError("growthSnapshot.version is unsupported");
  const briefRevision = text(value.briefRevision, "growthSnapshot.briefRevision", 64);
  if (!/^[a-f0-9]{64}$/u.test(briefRevision)) throw new GrowthValidationError("growthSnapshot.briefRevision is invalid");
  const playbookId = text(value.playbookId, "growthSnapshot.playbookId", 128);
  if (!PLAYBOOK_IDS.has(playbookId)) throw new GrowthValidationError("growthSnapshot.playbookId is unsupported");
  const evidenceSourceIds = textArray(value.evidenceSourceIds, "growthSnapshot.evidenceSourceIds", 30, 128);
  const sourceIDs = new Set();
  for (const sourceId of evidenceSourceIds) {
    if (!ID_PATTERN.test(sourceId) || sourceIDs.has(sourceId)) throw new GrowthValidationError("growthSnapshot.evidenceSourceIds is invalid");
    sourceIDs.add(sourceId);
  }
  const reviewStatus = text(value.reviewStatus, "growthSnapshot.reviewStatus", 30);
  if (!REVIEW_STATUSES.has(reviewStatus)) throw new GrowthValidationError("growthSnapshot.reviewStatus is invalid");
  exactKeys(value.pack, ["title", "summary", "blocks", "openQuestions", "riskFlags", "reviewChecks"], [], "growthSnapshot.pack");
  if (!Array.isArray(value.pack.blocks) || value.pack.blocks.length < 1 || value.pack.blocks.length > 30) throw new GrowthValidationError("growthSnapshot.pack.blocks is invalid");
  const seenBlocks = new Set();
  const blocks = value.pack.blocks.map((block, index) => {
    const normalized = validateBlock(block, index, sourceIDs, playbookId);
    if (seenBlocks.has(normalized.id)) throw new GrowthValidationError(`duplicate Growth block ${normalized.id}`);
    seenBlocks.add(normalized.id);
    return normalized;
  });
  if (!Array.isArray(value.pack.reviewChecks) || value.pack.reviewChecks.length < 1 || value.pack.reviewChecks.length > 30) throw new GrowthValidationError("growthSnapshot.pack.reviewChecks is invalid");
  const reviewChecks = value.pack.reviewChecks.map((check, index) => {
    exactKeys(check, ["status", "label", "reason"], [], `reviewChecks[${index}]`);
    const status = text(check.status, `reviewChecks[${index}].status`, 20);
    if (!CHECK_STATUSES.has(status)) throw new GrowthValidationError(`reviewChecks[${index}].status is invalid`);
    return { status, label: text(check.label, `reviewChecks[${index}].label`, 1_000), reason: text(check.reason, `reviewChecks[${index}].reason`, 2_000) };
  });
  return {
    version: 1,
    briefRevision,
    playbookId,
    evidenceSourceIds,
    pack: {
      title: text(value.pack.title, "growthSnapshot.pack.title", 1_000),
      summary: text(value.pack.summary, "growthSnapshot.pack.summary", 4_000),
      blocks,
      openQuestions: textArray(value.pack.openQuestions, "growthSnapshot.pack.openQuestions", 20, 2_000),
      riskFlags: textArray(value.pack.riskFlags, "growthSnapshot.pack.riskFlags", 20, 2_000),
      reviewChecks,
    },
    generatedBy: text(value.generatedBy, "growthSnapshot.generatedBy", 120),
    updatedAt: timestamp(value.updatedAt, "growthSnapshot.updatedAt", true),
    reviewStatus,
    reviewerNote: text(value.reviewerNote ?? "", "growthSnapshot.reviewerNote", 4_000, { required: false }),
    reviewUpdatedAt: timestamp(value.reviewUpdatedAt ?? "", "growthSnapshot.reviewUpdatedAt", false),
  };
}

function timestamp(value, path, required) {
  const normalized = text(value, path, 80, { required });
  if (!normalized) return "";
  const parsed = new Date(normalized);
  if (Number.isNaN(parsed.getTime()) || !/^\d{4}-\d{2}-\d{2}T/u.test(normalized)) {
    throw new GrowthValidationError(`${path} is not a valid timestamp`);
  }
  return normalized;
}

export function growthBlockToPlainText(block) {
  const heading = block.title;
  let content = "";
  if (block.kind === "prose") content = block.body;
  else if (["cards", "sequence", "checklist", "tasks"].includes(block.kind)) {
    content = block.items.map((item, index) => `${index + 1}. ${item.label}: ${item.value}${item.note ? ` (${item.note})` : ""}`).join("\n");
  } else if (block.kind === "table") {
    content = [block.columns.join("\t"), ...block.rows.map((row) => row.join("\t"))].join("\n");
  } else if (block.kind === "code") content = block.code;
  return `${heading}\n${content}`.trim();
}

export function growthPackToPlainText(snapshot) {
  const sections = [snapshot.pack.title, snapshot.pack.summary, ...snapshot.pack.blocks.map(growthBlockToPlainText)];
  if (snapshot.pack.openQuestions.length) sections.push(`Open questions\n${snapshot.pack.openQuestions.map((item) => `- ${item}`).join("\n")}`);
  if (snapshot.pack.riskFlags.length) sections.push(`Risk flags\n${snapshot.pack.riskFlags.map((item) => `- ${item}`).join("\n")}`);
  return sections.filter(Boolean).join("\n\n");
}

export function growthInsertGuard(snapshot, stale, pendingConfirmed = false) {
  if (stale) return { allowed: false, reason: "Growth Pack is stale. Generate it again from the current brief." };
  if (snapshot.reviewStatus === "rejected") return { allowed: false, reason: "Growth Pack was rejected and cannot be inserted." };
  if (snapshot.reviewStatus === "needs_review" && !pendingConfirmed) return { allowed: false, needsConfirmation: true, reason: "Growth Pack still needs review. Confirm that you reviewed this text before inserting it." };
  return { allowed: true, needsConfirmation: false, reason: "" };
}
