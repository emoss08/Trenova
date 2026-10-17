// Screenshots of an EDI connection drafted on the Desk from a trading
// partner's implementation guide. Run with the mock API and the web app up
// (see README.md):
//
//   node e2e/mock-api/desk-edi-draft-screenshots.mjs [outDir] [name,name,...]
//
// Each shot sets the `ediDraft` stage it needs, so the conversation is drawn
// at that point: one value held, ready, proposed, or a revised guide.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/desk-edi-draft");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 6000);
const THREAD = "/desk/t/athr_desk_edi_guide";
const ARTIFACT = (slug) => `/desk/c/athr_desk_edi_guide/a/${slug}`;

const scrollPane = (text) => async (page) => {
  await page.getByText(text, { exact: true }).first().scrollIntoViewIfNeeded();
};

function shotsFor(colorScheme) {
  return [
    { name: "held", path: ARTIFACT("acme-retail-connection"), stage: "held" },
    { name: "ready", path: ARTIFACT("acme-retail-connection-2"), stage: "ready" },
    {
      name: "ready-profiles",
      path: ARTIFACT("acme-retail-connection-2"),
      stage: "ready",
      act: scrollPane("Document types"),
    },
    {
      name: "ready-source",
      path: ARTIFACT("acme-retail-connection-2"),
      stage: "ready",
      act: async (page) => {
        await scrollPane("Interchange")(page);
        await page
          .getByRole("button", { name: /^Page 3: what the guide says/ })
          .first()
          .hover();
      },
    },
    {
      name: "ready-page",
      path: `${ARTIFACT("acme-retail-connection-2")}/page`,
      stage: "ready",
      height: 2600,
    },
    { name: "proposed", path: THREAD, stage: "proposed", fold: true },
    {
      name: "proposed-review",
      path: THREAD,
      stage: "proposed",
      fold: true,
      act: (page) => page.getByRole("button", { name: "Review", exact: true }).first().click(),
    },
    { name: "revised", path: ARTIFACT("acme-retail-connection-v4"), stage: "revised" },
    {
      name: "revised-profiles",
      path: ARTIFACT("acme-retail-connection-v4"),
      stage: "revised",
      act: scrollPane("Acme Retail Group 210 outbound"),
    },
  ].map((shot) => ({ ...shot, name: `${shot.name}-${colorScheme}`, colorScheme }));
}

async function setScenario(patch) {
  const response = await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(patch),
  });
  return response.json();
}

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium-1194/chrome-linux/chrome",
});
const failures = [];

for (const shot of [...shotsFor("light"), ...shotsFor("dark")]) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  const { anchor } = await setScenario({ ediDraft: shot.stage });
  const context = await browser.newContext({
    viewport: { width: 1440, height: shot.height ?? 900 },
    deviceScaleFactor: 2,
    colorScheme: shot.colorScheme,
  });
  await context.addInitScript((theme) => {
    window.localStorage.setItem("vite-ui-theme", theme);
  }, shot.colorScheme);
  const page = await context.newPage();
  await page.clock.install({ time: new Date(anchor * 1000) });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().includes("Failed to load resource")) {
      errors.push(message.text());
    }
  });
  await page.goto(`${APP}${shot.path}`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(shot.settle ?? SETTLE_MS);
  if (shot.fold) {
    await page
      .getByRole("button", { name: /^Workspace/ })
      .first()
      .click();
    await page.waitForTimeout(800);
  }
  if (shot.act) {
    try {
      await shot.act(page);
      await page.waitForTimeout(1500);
    } catch (error) {
      errors.push(`step: ${error.message.split("\n")[0]}`);
    }
  }
  const file = `${OUT}/${shot.name}.png`;
  await page.screenshot({ path: file });
  console.log(`${errors.length ? "!" : "✓"} ${shot.name} → ${file}`);
  for (const error of errors) console.log(`    ${error.slice(0, 300)}`);
  if (errors.length) failures.push(shot.name);
  await context.close();
}

await setScenario({ ediDraft: "ready" });
await browser.close();
if (failures.length) process.exitCode = 1;
