import assert from "node:assert/strict";
import test from "node:test";

import {
  GrowthValidationError,
  growthBlockToPlainText,
  growthInsertGuard,
  growthPackToPlainText,
  validateGrowthSnapshot,
} from "../src/growth.js";

function validSnapshot(overrides = {}) {
  return {
    version: 1,
    briefRevision: "a".repeat(64),
    playbookId: "offer-audience",
    evidenceSourceIds: ["outline"],
    pack: {
      title: "Offer plan",
      summary: "A bounded plan based on supplied facts.",
      blocks: [
        {
          id: "message",
          title: "Message",
          purpose: "Clarify the offer",
          kind: "prose",
          body: "Use only the supplied eight-lesson outline.",
          items: [],
          columns: [],
          rows: [],
          code: "",
          evidenceBasis: "supplied_evidence",
          sourceIds: ["outline"],
        },
        {
          id: "matrix",
          title: "Angle matrix",
          purpose: "Compare grounded angles",
          kind: "table",
          body: "",
          items: [],
          columns: ["Angle", "Use"],
          rows: [["Education", "Explain the verified outline"]],
          code: "",
          evidenceBasis: "mixed",
          sourceIds: ["outline"],
        },
      ],
      openQuestions: ["Confirm the schedule."],
      riskFlags: ["Do not promise sales results."],
      reviewChecks: [{ status: "review", label: "Owner review", reason: "Verify dates." }],
    },
    generatedBy: "Codex CLI",
    updatedAt: "2026-08-28T12:00:00Z",
    reviewStatus: "needs_review",
    ...overrides,
  };
}

test("validates finite Growth snapshot and renders plain text without HTML", () => {
  const snapshot = validateGrowthSnapshot(validSnapshot());
  assert.equal(snapshot.pack.blocks.length, 2);
  assert.equal(growthBlockToPlainText(snapshot.pack.blocks[1]), "Angle matrix\nAngle\tUse\nEducation\tExplain the verified outline");
  const all = growthPackToPlainText(snapshot);
  assert.match(all, /Offer plan/u);
  assert.match(all, /Open questions/u);
  assert.equal(all.includes("<table>"), false);
});

test("rejects unknown fields, invalid shapes, sources, enums, and imported metrics", () => {
  const cases = [];
  cases.push({ ...validSnapshot(), secret: "must not pass" });
  const wrongShape = validSnapshot();
  wrongShape.pack.blocks[0].items = [{ label: "extra", value: "not allowed", note: "" }];
  cases.push(wrongShape);
  const unknownSource = validSnapshot();
  unknownSource.pack.blocks[0].sourceIds = ["missing"];
  cases.push(unknownSource);
  const rejectedEnum = validSnapshot({ reviewStatus: "published" });
  cases.push(rejectedEnum);
  const metric = validSnapshot();
  metric.pack.blocks[0].evidenceBasis = "imported_metric";
  metric.pack.blocks[0].sourceIds = [];
  cases.push(metric);
  for (const value of cases) {
    assert.throws(() => validateGrowthSnapshot(value), GrowthValidationError);
  }
});

test("insertion guard blocks stale/rejected and requires explicit pending confirmation", () => {
  const pending = validateGrowthSnapshot(validSnapshot());
  assert.deepEqual(growthInsertGuard(pending, true), {
    allowed: false,
    reason: "Growth Pack is stale. Generate it again from the current brief.",
  });
  assert.equal(growthInsertGuard(pending, false).needsConfirmation, true);
  assert.equal(growthInsertGuard(pending, false, true).allowed, true);
  const rejected = validateGrowthSnapshot(validSnapshot({ reviewStatus: "rejected", reviewerNote: "Claims were not approved." }));
  assert.equal(growthInsertGuard(rejected, false).allowed, false);
  const approved = validateGrowthSnapshot(validSnapshot({ reviewStatus: "approved", reviewerNote: "Owner approved." }));
  assert.equal(growthInsertGuard(approved, false).allowed, true);
});
