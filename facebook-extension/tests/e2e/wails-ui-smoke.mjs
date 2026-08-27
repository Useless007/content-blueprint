import fs from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import puppeteer from "puppeteer-core";

const executablePath = process.env.CONTENT_BLUEPRINT_CHROME_PATH
  || "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe";
const targetURL = process.env.CONTENT_BLUEPRINT_WAILS_UI_URL || "http://127.0.0.1:4173";
const outputDirectory = path.resolve("..", "build", "qa");
const onboardingKey = "content-blueprint:onboarding:v3";

await fs.access(executablePath);
await fs.mkdir(outputDirectory, {recursive: true});

const browser = await puppeteer.launch({
  executablePath,
  headless: true,
  args: ["--no-first-run", "--no-default-browser-check"],
});

async function clickButtonByText(page, label, rootSelector = "body") {
  const clicked = await page.$eval(rootSelector, (root, expected) => {
    const button = [...root.querySelectorAll("button")]
      .find((candidate) => candidate.textContent?.trim().includes(expected));
    if (!(button instanceof HTMLButtonElement)) return false;
    button.click();
    return true;
  }, label);
  if (!clicked) throw new Error(`button not found: ${label}`);
}

async function waitForBodyText(page, text) {
  await page.waitForFunction(
    (expected) => document.body.innerText.includes(expected),
    {timeout: 6_000},
    text,
  );
}

try {
  const page = await browser.newPage();
  await page.emulateMediaFeatures([{name: "prefers-reduced-motion", value: "no-preference"}]);
  await page.setViewport({width: 1280, height: 900, deviceScaleFactor: 1});
  await page.goto(targetURL, {waitUntil: "networkidle0"});
  await page.evaluate((key) => localStorage.removeItem(key), onboardingKey);
  await page.reload({waitUntil: "networkidle0"});

  await page.waitForSelector(".onboarding-center[role=dialog]");
  const onboarding = await page.evaluate(() => ({
    title: document.querySelector("#onboarding-center-title")?.textContent?.trim(),
    missions: document.querySelectorAll(".onboarding-mission").length,
    hasMcpMission: document.body.innerText.includes("ส่งงานผ่าน MCP แล้วใช้ใน Chrome/Brave"),
  }));
  if (onboarding.title !== "เลือกงานที่อยากลองทำ" || onboarding.missions !== 8 || !onboarding.hasMcpMission) {
    throw new Error(`onboarding mission center is incomplete: ${JSON.stringify(onboarding)}`);
  }
  await page.screenshot({
    path: path.join(outputDirectory, "onboarding-mission-center.png"),
    fullPage: true,
  });

  const competitorStarted = await page.evaluate(() => {
    const mission = [...document.querySelectorAll(".onboarding-mission")]
      .find((item) => item.querySelector("h3")?.textContent?.includes("ศึกษาคู่แข่ง"));
    const button = mission?.querySelector("button");
    if (!(button instanceof HTMLButtonElement)) return false;
    button.click();
    return true;
  });
  if (!competitorStarted) throw new Error("competitor safety mission could not be started");
  await waitForBodyText(page, "เส้นแบ่งที่ต้องรู้ก่อน");
  await page.waitForSelector('[data-tour="growth-audience-safety"]');
  const safetyText = await page.$eval('[data-tour="growth-audience-safety"]', (element) => element.textContent ?? "");
  if (!safetyText.includes("ไม่รองรับการดูดรายชื่อ")) {
    throw new Error("competitor audience safety boundary is missing");
  }
  await page.screenshot({
    path: path.join(outputDirectory, "onboarding-competitor-safety.png"),
    fullPage: true,
  });
  await clickButtonByText(page, "จบภายหลัง");

  await page.evaluate((key) => {
    localStorage.setItem(key, JSON.stringify({
      version: 3,
      welcomed: true,
      activeMissionId: "",
      lastStepByMission: {},
      completedMissionIds: [],
    }));
    localStorage.setItem("content-blueprint:workspace", "growth");
  }, onboardingKey);

  for (const viewport of [
    {name: "desktop", width: 1280, height: 900},
    {name: "mobile", width: 375, height: 812},
    {name: "landscape", width: 844, height: 390},
  ]) {
    await page.setViewport({width: viewport.width, height: viewport.height, deviceScaleFactor: 1});
    await page.goto(targetURL, {waitUntil: "networkidle0"});
    await page.waitForSelector('[data-tour="growth-shell"]');
    await page.waitForSelector(".grw-shell .ai-studio__floor", {visible: true});
    const metrics = await page.evaluate(() => {
      const shell = document.querySelector(".grw-shell");
      const main = document.querySelector(".grw-main");
      const studio = shell?.querySelector(".ai-studio");
      const floor = shell?.querySelector(".ai-studio__floor");
      if (!(shell instanceof HTMLElement) || !(studio instanceof HTMLElement)
        || !(floor instanceof HTMLElement) || !(main instanceof HTMLElement)) {
        throw new Error("Growth Hub or AI Studio is missing");
      }
      return {
        documentClientWidth: document.documentElement.clientWidth,
        documentScrollWidth: document.documentElement.scrollWidth,
        shellClientWidth: shell.clientWidth,
        shellScrollWidth: shell.scrollWidth,
        studioClientWidth: studio.clientWidth,
        studioScrollWidth: studio.scrollWidth,
        floorClientWidth: floor.clientWidth,
        floorScrollWidth: floor.scrollWidth,
        mainLeft: Math.round(main.getBoundingClientRect().left),
        floatingControlsOverlap: (() => {
          const updater = document.querySelector('.update-launcher');
          const help = document.querySelector('.onboarding-help-button');
          if (!(updater instanceof HTMLElement) || !(help instanceof HTMLElement)) {
            throw new Error('floating update/help controls are missing');
          }
          const left = updater.getBoundingClientRect();
          const right = help.getBoundingClientRect();
          return left.left < right.right && left.right > right.left
            && left.top < right.bottom && left.bottom > right.top;
        })(),
      };
    });
    if (metrics.documentScrollWidth > metrics.documentClientWidth + 1) {
      throw new Error(`${viewport.name}: document has horizontal overflow ${JSON.stringify(metrics)}`);
    }
    if (metrics.shellScrollWidth > metrics.shellClientWidth + 1) {
      throw new Error(`${viewport.name}: Growth Hub has horizontal overflow ${JSON.stringify(metrics)}`);
    }
    if (metrics.studioScrollWidth > metrics.studioClientWidth + 1) {
      throw new Error(`${viewport.name}: AI Studio has horizontal overflow ${JSON.stringify(metrics)}`);
    }
    if (metrics.floorScrollWidth > metrics.floorClientWidth + 1) {
      throw new Error(`${viewport.name}: AI Studio floor has horizontal overflow ${JSON.stringify(metrics)}`);
    }
    if (metrics.floatingControlsOverlap) {
      throw new Error(`${viewport.name}: update and help controls overlap ${JSON.stringify(metrics)}`);
    }
    await page.screenshot({
      path: path.join(outputDirectory, `growth-hub-${viewport.name}.png`),
      fullPage: true,
    });
    process.stdout.write(`PASS ${viewport.name}: ${JSON.stringify(metrics)}\n`);
  }

  await page.setViewport({width: 1280, height: 900, deviceScaleFactor: 1});
  await page.goto(targetURL, {waitUntil: "networkidle0"});
  await clickButtonByText(page, "Leads & Commission", ".grw-tabs");
  await page.waitForSelector('[data-tour="growth-leads"]');
  await clickButtonByText(page, "UTM Builder", ".grw-tabs");
  await page.waitForSelector('[data-tour="growth-utm"]');
  await clickButtonByText(page, "Experiment Log", ".grw-tabs");
  await page.waitForSelector('[data-tour="growth-experiment"]');
  process.stdout.write("PASS Growth Hub tab navigation\n");

  await clickButtonByText(page, "AI Playbooks", ".grw-tabs");
  await page.waitForSelector(".grw-shell .ai-studio__floor", {visible: true});
  const handoffStyles = await page.evaluate(async () => {
    const floor = document.querySelector(".grw-shell .ai-studio__floor");
    const strategist = floor?.querySelector('.ai-agent[data-agent="strategist"]');
    const copywriter = floor?.querySelector('.ai-agent[data-agent="copywriter"]');
    if (!(floor instanceof HTMLElement) || !(strategist instanceof HTMLElement)
      || !(copywriter instanceof HTMLElement)) throw new Error("handoff fixture targets are missing");
    floor.scrollIntoView({block: "center"});
    strategist.dataset.status = "done";
    copywriter.dataset.status = "working";
    const carriedFile = document.createElement("span");
    carriedFile.className = "ai-agent__handoff-file";
    carriedFile.append(document.createElement("i"));
    strategist.append(carriedFile);
    const handoff = document.createElement("div");
    handoff.className = "ai-handoff";
    handoff.dataset.from = "strategist";
    handoff.dataset.to = "copywriter";
    const file = document.createElement("span");
    file.className = "ai-handoff__file";
    file.append(document.createElement("i"));
    const label = document.createElement("small");
    label.textContent = "กลยุทธ์ → นักเขียน";
    handoff.append(file, label);
    floor.append(handoff);
    await new Promise((resolve) => setTimeout(resolve, 850));
    return {
      animationName: getComputedStyle(handoff).animationName,
      animationDuration: getComputedStyle(handoff).animationDuration,
      reducedMotion: matchMedia("(prefers-reduced-motion: reduce)").matches,
      writerPose: getComputedStyle(copywriter.querySelector(".ai-agent__sprite")).transform,
      carriedFileDisplay: getComputedStyle(carriedFile).display,
    };
  });
  if (handoffStyles.animationName !== "ai-handoff-route" || handoffStyles.writerPose === "none"
    || handoffStyles.carriedFileDisplay === "none") {
    throw new Error(`AI handoff animation is not active: ${JSON.stringify(handoffStyles)}`);
  }
  await page.screenshot({
    path: path.join(outputDirectory, "ai-studio-handoff.png"),
    fullPage: true,
  });
  process.stdout.write(`PASS AI handoff animation: ${JSON.stringify(handoffStyles)}\n`);

  await page.emulateMediaFeatures([{name: "prefers-reduced-motion", value: "reduce"}]);
  await page.setViewport({width: 375, height: 812, deviceScaleFactor: 1});
  await page.goto(targetURL, {waitUntil: "networkidle0"});
  await page.waitForSelector(".grw-shell .ai-agent");
  const reducedMotion = await page.$eval(".grw-shell .ai-agent", (agent) => {
    const style = getComputedStyle(agent);
    return {animationDuration: style.animationDuration, transitionDuration: style.transitionDuration};
  });
  const reducedDurations = new Set(["0s", "1e-05s"]);
  if (!reducedDurations.has(reducedMotion.animationDuration) || !reducedDurations.has(reducedMotion.transitionDuration)) {
    throw new Error(`reduced motion is not enforced: ${JSON.stringify(reducedMotion)}`);
  }
  process.stdout.write(`PASS reduced-motion: ${JSON.stringify(reducedMotion)}\n`);
} finally {
  await browser.close();
}
