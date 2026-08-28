import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { once } from "node:events";
import {
  cp,
  mkdir,
  mkdtemp,
  readFile,
  rm,
  writeFile,
} from "node:fs/promises";
import { tmpdir } from "node:os";
import { basename, dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import puppeteer from "puppeteer-core";

const EXTENSION_ID = "ppncejmpiekmkepaeccdnpnpgdcfafje";
const EXTENSION_ORIGIN = `chrome-extension://${EXTENSION_ID}`;
const FIXTURE_HOST = "e2e.facebook.com";
const HERE = dirname(fileURLToPath(import.meta.url));
const EXTENSION_ROOT = resolve(HERE, "../..");
const REPO_ROOT = resolve(EXTENSION_ROOT, "..");

function browserPaths(environmentOverride, vendorPath, executableName) {
  return Object.freeze(
    [
      process.env[environmentOverride],
      join(process.env.ProgramFiles ?? "C:\\Program Files", vendorPath, executableName),
      join(process.env["ProgramFiles(x86)"] ?? "C:\\Program Files (x86)", vendorPath, executableName),
      process.env.LOCALAPPDATA
        ? join(process.env.LOCALAPPDATA, vendorPath, executableName)
        : null,
    ].filter((path, index, paths) => path && paths.indexOf(path) === index),
  );
}

const BROWSERS = Object.freeze([
  {
    name: "chrome",
    executablePaths: browserPaths(
      "CONTENT_BLUEPRINT_CHROME_PATH",
      "Google\\Chrome\\Application",
      "chrome.exe",
    ),
  },
  {
    name: "brave",
    executablePaths: browserPaths(
      "CONTENT_BLUEPRINT_BRAVE_PATH",
      "BraveSoftware\\Brave-Browser\\Application",
      "brave.exe",
    ),
  },
]);

const BRIEF = Object.freeze({
  topic: "Local E2E content workflow",
  audience: "Page administrators testing the local companion",
  objective: "Verify a human-reviewed content pack handoff",
  offer: "",
  brandVoice: "Clear and factual",
  language: "English",
  productDetails: "Synthetic browser-test data only.",
  evidence: [],
  additionalInstructions: "Do not publish anything.",
});

const PACK = Object.freeze({
  hooks: [
    "A local workflow you can inspect",
    "Keep the page admin in control",
    "Move a reviewed draft without auto-posting",
  ],
  longPost:
    "This synthetic Content Pack proves the local browser handoff while keeping the page administrator responsible for review and publishing.",
  shortPost: "Synthetic local handoff verified. Review before publishing.",
  reelScript: "Hook: local control. Body: review the draft. CTA: verify it yourself.",
  carouselSlides: [
    { headline: "Brief", body: "Start with an explicit brief." },
    { headline: "Review", body: "Check every claim before use." },
    { headline: "Insert", body: "Insert plain text only when you choose." },
  ],
  cta: "Review the draft before deciding whether to publish.",
  firstComment: "This is synthetic local browser-test content.",
  replyBank: [
    { intent: "How does it work?", reply: "The companion moves a validated local draft." },
    { intent: "Does it post?", reply: "No. A human remains responsible for publishing." },
    { intent: "Is this real campaign data?", reply: "No. This is synthetic E2E test data." },
  ],
  complianceNotes: ["Synthetic test content; not for publication."],
});

const GROWTH_BRIEF = Object.freeze({
  playbookId: "offer-audience",
  language: "English",
  brandVoice: "Clear and factual",
  inputs: {
    audience: "Synthetic page administrators",
    offer: "A local-only test offer",
    problems: "Need a reviewed handoff without automatic publishing",
  },
  evidence: null,
});

const GROWTH_PACK = Object.freeze({
  title: "Synthetic Growth Pack",
  summary: "A browser-test pack that remains under human control.",
  blocks: [
    {
      id: "message",
      title: "Reviewed message",
      purpose: "Verify safe plain-text rendering and insertion.",
      kind: "prose",
      body: "Growth text <b>must stay plain text</b>.",
      items: [], columns: [], rows: [], code: "",
      evidenceBasis: "user_input", sourceIds: [],
    },
    {
      id: "matrix",
      title: "Angle matrix",
      purpose: "Verify finite table rendering.",
      kind: "table",
      body: "", items: [], columns: ["Angle", "Proof"],
      rows: [["Local control", "User-reviewed fixture"]], code: "",
      evidenceBasis: "user_input", sourceIds: [],
    },
  ],
  openQuestions: [], riskFlags: [],
  reviewChecks: [{ status: "ready", label: "Human control", reason: "Publishing remains manual." }],
});

const CODE_BRIEF = Object.freeze({
  playbookId: "seo-structured-data", language: "English", brandVoice: "Factual",
  inputs: { canonicalUrl: "https://example.com/test", pageType: "Article", visibleContent: "Synthetic test page" },
  evidence: null,
});

const CODE_PACK = Object.freeze({
  title: "Synthetic Code Pack", summary: "Safe code rendering fixture.",
  blocks: [{
    id: "snippet", title: "Tracking snippet", purpose: "Verify code is rendered as text.", kind: "code",
    body: "", items: [], columns: [], rows: [], code: "utm_source=local&safe=true",
    evidenceBasis: "user_input", sourceIds: [],
  }],
  openQuestions: [], riskFlags: [],
  reviewChecks: [{ status: "ready", label: "Synthetic", reason: "Test-only content." }],
});

function log(message) {
  process.stdout.write(`${message}\n`);
}

async function exists(path) {
  try {
    await readFile(path);
    return true;
  } catch (error) {
    if (error?.code === "EISDIR") return true;
    if (error?.code === "ENOENT") return false;
    throw error;
  }
}

async function createTemporaryExtension(root) {
  const destination = join(root, "extension");
  await mkdir(destination, { recursive: true });
  for (const path of [
    "content-script.js",
    "service-worker.js",
    "sidepanel.css",
    "sidepanel.html",
    "sidepanel.js",
  ]) {
    await cp(join(EXTENSION_ROOT, path), join(destination, path));
  }
  await cp(join(EXTENSION_ROOT, "src"), join(destination, "src"), { recursive: true });

  const productionManifestPath = join(EXTENSION_ROOT, "manifest.json");
  const productionText = await readFile(productionManifestPath, "utf8");
  assert.equal(
    /localhost|127\.0\.0\.1/u.test(productionText),
    false,
    "production manifest must not grant localhost permissions",
  );
  const manifest = JSON.parse(productionText);
  const iconPaths = new Set(Object.values(manifest.icons ?? {}));
  const actionIcons = manifest.action?.default_icon;
  if (typeof actionIcons === "string") {
    iconPaths.add(actionIcons);
  } else {
    for (const path of Object.values(actionIcons ?? {})) iconPaths.add(path);
  }
  for (const path of iconPaths) {
    await cp(join(EXTENSION_ROOT, path), join(destination, path));
  }
  const localMatches = [
    "http://127.0.0.1/*",
    "https://127.0.0.1/*",
    "http://localhost/*",
    "https://localhost/*",
  ];
  manifest.host_permissions = [...new Set([...manifest.host_permissions, ...localMatches])];
  manifest.content_scripts[0].matches = [
    ...new Set([...manifest.content_scripts[0].matches, ...localMatches]),
  ];
  await writeFile(
    join(destination, "manifest.json"),
    `${JSON.stringify(manifest, null, 2)}\n`,
    "utf8",
  );
  return destination;
}

async function startFixtureServer() {
  const child = spawn("go", ["run", "./tests/e2e/fixture-server.go"], {
    cwd: EXTENSION_ROOT,
    stdio: ["pipe", "pipe", "pipe"],
    windowsHide: true,
  });
  let stderr = "";
  child.stderr.setEncoding("utf8");
  child.stderr.on("data", (chunk) => {
    stderr += chunk;
  });
  child.stdout.setEncoding("utf8");

  const url = await new Promise((resolveURL, reject) => {
    let stdout = "";
    const timer = setTimeout(() => {
      reject(new Error(`fixture server did not start in time: ${stderr.trim()}`));
    }, 30_000);
    child.once("error", (error) => {
      clearTimeout(timer);
      reject(error);
    });
    child.once("exit", (code) => {
      clearTimeout(timer);
      reject(new Error(`fixture server exited with ${code}: ${stderr.trim()}`));
    });
    child.stdout.on("data", (chunk) => {
      stdout += chunk;
      const newline = stdout.indexOf("\n");
      if (newline === -1) return;
      clearTimeout(timer);
      resolveURL(stdout.slice(0, newline).trim());
    });
  });
  const parsed = new URL(url);
  assert.equal(parsed.hostname, "127.0.0.1");
  return {
    child,
    url: `https://${FIXTURE_HOST}:${parsed.port}/fixture.html`,
    async stop() {
      if (child.exitCode !== null) return;
      child.stdin.end();
      const exit = once(child, "exit");
      await Promise.race([
        exit,
        new Promise((resolveTimeout) => setTimeout(resolveTimeout, 5_000)),
      ]);
      if (child.exitCode === null) child.kill();
    },
  };
}

async function extensionMessage(page, message) {
  return page.evaluate(
    (payload) =>
      new Promise((resolveMessage, rejectMessage) => {
        chrome.runtime.sendMessage(payload, (response) => {
          const problem = chrome.runtime.lastError;
          if (problem) {
            rejectMessage(new Error(problem.message || "runtime message failed"));
            return;
          }
          resolveMessage(response);
        });
      }),
    message,
  );
}

async function extensionWorker(browser) {
  const expectedURLPrefix = `${EXTENSION_ORIGIN}/`;
  const target =
    browser
      .targets()
      .find(
        (candidate) =>
          candidate.type() === "service_worker" &&
          candidate.url().startsWith(expectedURLPrefix),
      ) ??
    (await browser.waitForTarget(
      (candidate) =>
        candidate.type() === "service_worker" &&
        candidate.url().startsWith(expectedURLPrefix),
      { timeout: 20_000 },
    ));
  return target.worker();
}

async function sendInsert(extensionPage, text) {
  return extensionPage.evaluate(async ({ fixtureHost, payload }) => {
    const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
    const fixture = tabs.find((tab) => new URL(tab.url).hostname === fixtureHost);
    if (!fixture?.id) {
      throw new Error("local fixture tab was not found");
    }
    return chrome.tabs.sendMessage(fixture.id, {
      type: "facebook.insert",
      text: payload,
    });
  }, { fixtureHost: FIXTURE_HOST, payload: text });
}

async function seedPack(dataDirectory, briefRevision, browserName) {
  const snapshot = {
    version: 1,
    briefRevision,
    pack: PACK,
    groundingSources: [
      { title: "Local E2E source", url: "https://example.com/browser-e2e" },
    ],
    generatedBy: `browser-e2e:${browserName}`,
    updatedAt: new Date().toISOString(),
  };
  await writeFile(
    join(dataDirectory, "facebook-pack.json"),
    `${JSON.stringify(snapshot, null, 2)}\n`,
    "utf8",
  );
}

function growthRevision(brief) {
  const canonical = { ...brief, evidence: [] };
  return createHash("sha256").update(JSON.stringify(canonical)).digest("hex");
}

async function seedGrowthState(dataDirectory, browserName, { brief = GROWTH_BRIEF, pack = GROWTH_PACK, packRevision, reviewStatus = "needs_review" } = {}) {
  const directory = join(dataDirectory, "GrowthWorkbench");
  await mkdir(directory, { recursive: true });
  const briefRevision = growthRevision(brief);
  const updatedAt = new Date().toISOString();
  await writeFile(join(directory, "growth-brief.json"), `${JSON.stringify({
    version: 1, briefRevision, brief, updatedAt,
  }, null, 2)}\n`, "utf8");
  await writeFile(join(directory, "growth-pack.json"), `${JSON.stringify({
    version: 1,
    briefRevision: packRevision ?? briefRevision,
    playbookId: brief.playbookId,
    evidenceSourceIds: [],
    pack,
    generatedBy: `browser-e2e:${browserName}`,
    updatedAt,
    reviewStatus,
  }, null, 2)}\n`, "utf8");
  return { briefRevision, packRevision: packRevision ?? briefRevision };
}

async function runBrowser(browser, fixtureURL) {
  let executablePath = null;
  for (const candidate of browser.executablePaths) {
    if (await exists(candidate)) {
      executablePath = candidate;
      break;
    }
  }
  assert.ok(
    executablePath,
    `${browser.name} executable is missing; checked: ${browser.executablePaths.join(", ")}`,
  );
  const temporaryRoot = await mkdtemp(join(tmpdir(), `content-blueprint-${browser.name}-`));
  const profileDirectory = join(temporaryRoot, "profile");
  const companionDirectory = join(temporaryRoot, "companion-data");
  await mkdir(profileDirectory, { recursive: true });
  await mkdir(companionDirectory, { recursive: true });
  const extensionDirectory = await createTemporaryExtension(temporaryRoot);
  let browserProcess;

  try {
    browserProcess = await puppeteer.launch({
      executablePath,
      browser: "chrome",
      headless: true,
      acceptInsecureCerts: true,
      userDataDir: profileDirectory,
      enableExtensions: true,
      env: {
        ...process.env,
        CONTENT_BLUEPRINT_DATA_DIR: companionDirectory,
      },
      args: [
        `--host-resolver-rules=MAP ${FIXTURE_HOST} 127.0.0.1`,
        "--disable-background-networking",
        "--disable-component-update",
        "--disable-default-apps",
        "--disable-sync",
        "--no-first-run",
        "--no-default-browser-check",
      ],
    });
    const loadedID = await browserProcess.installExtension(extensionDirectory);
    assert.equal(loadedID, EXTENSION_ID);

    const extensionPage = await browserProcess.newPage();
    extensionPage.setDefaultTimeout(20_000);
    await extensionPage.goto(`${EXTENSION_ORIGIN}/sidepanel.html`, {
      waitUntil: "domcontentloaded",
      timeout: 20_000,
    });
    assert.equal(await extensionPage.title(), "Content Blueprint for Facebook");
    const worker = await extensionWorker(browserProcess);
    assert.ok(worker, "extension service worker must be running");
    assert.equal(worker.url(), `${EXTENSION_ORIGIN}/service-worker.js`);

    const health = await extensionMessage(extensionPage, { type: "companion.health" });
    assert.deepEqual(health, { ok: true, connected: true, protocolVersion: "1.0" });

    const sync = await extensionMessage(extensionPage, {
      type: "companion.syncBrief",
      brief: BRIEF,
    });
    assert.equal(sync.ok, true, JSON.stringify(sync));
    assert.match(sync.briefRevision, /^[a-f0-9]{64}$/u);
    assert.match(sync.updatedAt, /^\d{4}-\d{2}-\d{2}T/u);
    assert.equal(await exists(join(companionDirectory, "facebook-brief.json")), true);

    await seedPack(companionDirectory, sync.briefRevision, browser.name);
    const latest = await extensionMessage(extensionPage, {
      type: "companion.getLatestPack",
      briefRevision: sync.briefRevision,
    });
    assert.equal(latest.ok, true);
    assert.equal(latest.found, true);
    assert.equal(latest.stale, false);
    assert.equal(latest.briefRevision, sync.briefRevision);
    assert.deepEqual(latest.pack, PACK);
    assert.equal(latest.sources[0].url, "https://example.com/browser-e2e");
    assert.equal(latest.model, `browser-e2e:${browser.name}`);

    await extensionPage.waitForSelector("#fetchCompanionPack:not([disabled])");
    await extensionPage.click("#fetchCompanionPack");
    await extensionPage.waitForFunction(() => !document.querySelector("#resultsSection")?.hidden);
    assert.match(
      await extensionPage.$eval("#outputPanels", (element) => element.textContent),
      /local workflow you can inspect/iu,
    );
    await extensionPage.type("#campaignProduct", "Edited after fetch");
    assert.equal(
      await extensionPage.$eval("#resultsSection", (element) => element.hidden),
      true,
      "editing a brief must hide the previously fetched pack",
    );
    await extensionPage.waitForSelector("#fetchCompanionPack:not([disabled])");
    await extensionPage.click("#fetchCompanionPack");
    assert.equal(
      await extensionPage.$eval("#resultsSection", (element) => element.hidden),
      true,
      "fetch must stay blocked until the edited brief is synced",
    );

    await extensionPage.reload({waitUntil: "domcontentloaded"});
    await extensionPage.waitForSelector("#fetchCompanionPack:not([disabled])");
    await extensionPage.click("#fetchCompanionPack");
    await extensionPage.type("#campaignProduct", "Changed while fetching");
    await new Promise((resolveDelay) => setTimeout(resolveDelay, 750));
    assert.equal(
      await extensionPage.$eval("#resultsSection", (element) => element.hidden),
      true,
      "an in-flight fetch must not reopen results after the brief changes",
    );

    const changedSync = await extensionMessage(extensionPage, {
      type: "companion.syncBrief",
      brief: {...BRIEF, topic: `${BRIEF.topic} changed`},
    });
    assert.equal(changedSync.ok, true);
    assert.notEqual(changedSync.briefRevision, sync.briefRevision);
    const staleWithoutExpectedRevision = await extensionMessage(extensionPage, {
      type: "companion.getLatestPack",
    });
    assert.equal(staleWithoutExpectedRevision.ok, true);
    assert.equal(staleWithoutExpectedRevision.found, true);
    assert.equal(
      staleWithoutExpectedRevision.stale,
      true,
      "native host must compare the pack with the stored brief even without a caller revision",
    );

    const fixturePage = await browserProcess.newPage();
    fixturePage.setDefaultTimeout(20_000);
    await fixturePage.goto(fixtureURL, { waitUntil: "domcontentloaded", timeout: 20_000 });
    assert.equal(await fixturePage.title(), "Content Blueprint local Facebook editor fixture");

    const initialGrowth = await seedGrowthState(companionDirectory, browser.name);
    const latestGrowth = await extensionMessage(extensionPage, { type: "companion.getLatestGrowthPack" });
    assert.equal(latestGrowth.ok, true, JSON.stringify(latestGrowth));
    assert.equal(latestGrowth.found, true);
    assert.equal(latestGrowth.stale, false);
    assert.equal(latestGrowth.briefRevision, initialGrowth.briefRevision);
    assert.equal(latestGrowth.snapshot.reviewStatus, "needs_review");
    assert.deepEqual(latestGrowth.snapshot.pack, GROWTH_PACK);

    await extensionPage.evaluate(() => document.querySelector("#growthFetch").click());
    await new Promise((resolveDelay) => setTimeout(resolveDelay, 1_000));
    assert.equal(await extensionPage.$eval("#growthResults", (element) => element.hidden), false, await extensionPage.$eval("#appStatus", (element) => element.textContent));
    assert.match(await extensionPage.$eval("#growthBlocks", (element) => element.textContent), /Growth text <b>must stay plain text<\/b>/u);
    assert.equal(await extensionPage.$$eval("#growthBlocks b", (elements) => elements.length), 0, "model HTML must never become DOM HTML");
    assert.equal(await extensionPage.$$eval(".growth-table", (elements) => elements.length), 1);
    assert.equal(await extensionPage.$eval("#growthInsertAll", (element) => element.disabled), true, "needs_review must block insertion before confirmation");
    assert.equal(await fixturePage.$eval("#plainEditor", (element) => element.value), "", "rendering must not insert anything");

    await seedGrowthState(companionDirectory, browser.name, { brief: CODE_BRIEF, pack: CODE_PACK, reviewStatus: "approved" });
    await extensionPage.evaluate(() => document.querySelector("#growthFetch").click());
    await new Promise((resolveDelay) => setTimeout(resolveDelay, 1_000));
    assert.equal(await extensionPage.$eval("#growthState", (element) => element.dataset.state), "configured", await extensionPage.$eval("#appStatus", (element) => element.textContent));
    assert.match(await extensionPage.$eval(".growth-code", (element) => element.textContent), /utm_source=local/u);

    const changedGrowthBrief = {
      ...GROWTH_BRIEF,
      inputs: { ...GROWTH_BRIEF.inputs, problems: "Changed after generation" },
    };
    await seedGrowthState(companionDirectory, browser.name, {
      brief: changedGrowthBrief,
      packRevision: initialGrowth.packRevision,
      reviewStatus: "approved",
    });
    const staleGrowth = await extensionMessage(extensionPage, { type: "companion.getLatestGrowthPack" });
    assert.equal(staleGrowth.ok, true);
    assert.equal(staleGrowth.stale, true, "native host must mark a Growth Pack stale against the stored brief");
    await extensionPage.evaluate(() => document.querySelector("#growthFetch").click());
    await new Promise((resolveDelay) => setTimeout(resolveDelay, 1_000));
    assert.equal(await extensionPage.$eval("#growthState", (element) => element.dataset.state), "missing");
    assert.equal(await extensionPage.$eval("#growthInsertAll", (element) => element.disabled), true);

    await seedGrowthState(companionDirectory, browser.name, { reviewStatus: "rejected" });
    await extensionPage.evaluate(() => document.querySelector("#growthFetch").click());
    await new Promise((resolveDelay) => setTimeout(resolveDelay, 1_000));
    assert.equal(await extensionPage.$eval("#growthWarning", (element) => element.classList.contains("growth-warning--blocked")), true);
    assert.equal(await extensionPage.$eval("#growthInsertAll", (element) => element.disabled), true, "rejected pack must not be insertable");

    await seedGrowthState(companionDirectory, browser.name, { reviewStatus: "needs_review" });
    await extensionPage.evaluate(() => document.querySelector("#growthFetch").click());
    await new Promise((resolveDelay) => setTimeout(resolveDelay, 1_000));
    assert.equal(await extensionPage.$eval("#growthConfirmWrap", (element) => element.hidden), false);
    assert.equal(await extensionPage.$eval("#growthInsertAll", (element) => element.disabled), true);
    await extensionPage.evaluate(() => document.querySelector("#growthPendingConfirm").click());
    assert.equal(await extensionPage.$eval("#growthInsertAll", (element) => element.disabled), false, "explicit review confirmation must unlock insertion");
    assert.equal(await fixturePage.$eval("#plainEditor", (element) => element.value), "", "confirmation alone must not insert");

    await fixturePage.click("#plainEditor");
    await extensionPage.evaluate(() => document.querySelector("#growthInsertAll").click());
    await fixturePage.waitForFunction(() => document.querySelector("#plainEditor").value.includes("Synthetic Growth Pack"));
    assert.match(await fixturePage.$eval("#plainEditor", (element) => element.value), /Growth text <b>must stay plain text<\/b>/u);
    assert.equal((await fixturePage.evaluate(() => window.fixtureState)).postClicks, 0, "Growth insertion must never click Post");

    await fixturePage.$eval("#plainEditor", (element) => { element.value = ""; });
    await fixturePage.click("#plainEditor");
    const textareaPayload = "Plain text from Content Blueprint";
    const textareaResult = await sendInsert(extensionPage, textareaPayload);
    assert.deepEqual(textareaResult, { ok: true, target: "textarea" });
    assert.equal(
      await fixturePage.$eval("#plainEditor", (element) => element.value),
      textareaPayload,
    );

    await fixturePage.click("#richEditor");
    const richPayload = "Literal <b>not HTML</b> & safe";
    const richResult = await sendInsert(extensionPage, richPayload);
    assert.deepEqual(richResult, { ok: true, target: "contenteditable" });
    assert.equal(
      await fixturePage.$eval("#richEditor", (element) => element.textContent),
      richPayload,
    );
    assert.equal(await fixturePage.$$eval("#richEditor b", (elements) => elements.length), 0);

    const fixtureState = await fixturePage.evaluate(() => window.fixtureState);
    assert.equal(fixtureState.postClicks, 0, "the extension must never click Post");
    assert.ok(fixtureState.beforeInputEvents >= 1, "insertion must emit an editable beforeinput event");
    assert.ok(fixtureState.inputEvents >= 2, "both editors must receive input events");

    log(`PASS ${browser.name}: extension + native health/sync/fetch + plain-text insertion`);
  } finally {
    await browserProcess?.close().catch(() => {});
    const resolvedTemporaryRoot = resolve(temporaryRoot);
    assert.equal(
      dirname(resolvedTemporaryRoot),
      resolve(tmpdir()),
      `refusing to remove unexpected directory: ${resolvedTemporaryRoot}`,
    );
    assert.ok(
      basename(resolvedTemporaryRoot).startsWith(`content-blueprint-${browser.name}-`),
      `refusing to remove unexpected directory: ${resolvedTemporaryRoot}`,
    );
    await rm(resolvedTemporaryRoot, { recursive: true, force: true });
  }
}

async function main() {
  const companionExecutable = join(REPO_ROOT, "build", "bin", "content-blueprint-companion.exe");
  assert.equal(
    await exists(companionExecutable),
    true,
    `build the native companion first: go build -o ${companionExecutable} ./cmd/content-blueprint-companion`,
  );
  const fixture = await startFixtureServer();
  try {
    for (const browser of BROWSERS) {
      await runBrowser(browser, fixture.url);
    }
  } finally {
    await fixture.stop();
  }
  log("PASS all installed browsers");
}

main().catch((error) => {
  process.stderr.write(`${error?.stack || error}\n`);
  process.exitCode = 1;
});
