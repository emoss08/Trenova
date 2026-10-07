// Screenshots of AI control (/admin/agent-control) against the mock API. See README.md.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const PAGE = `${APP}/admin/agent-control`;
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/ai-control");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 7000);

const DEFAULT_SCENARIO = { ai: true, aiProviders: "configured", aiPaused: false };

/** Scrolls the page's scrolling ancestor so the lower half of a tab is in view. */
async function scrollDown(page, by = 760) {
  await page.evaluate((amount) => {
    let element = document.querySelector(".aic");
    while (element && element !== document.body) {
      const style = getComputedStyle(element);
      if (/(auto|scroll)/.test(style.overflowY) && element.scrollHeight > element.clientHeight) {
        element.scrollTop += amount;
        return;
      }
      element = element.parentElement;
    }
    window.scrollBy(0, amount);
  }, by);
  await page.waitForTimeout(400);
}

const SHOTS = [
  { name: "overview-dark", scenario: {} },
  { name: "overview-lower", scenario: {}, act: (page) => scrollDown(page) },
  { name: "overview-light", scenario: {}, colorScheme: "light" },
  {
    name: "overview-tune-up-applied",
    scenario: {},
    act: async (page) => {
      await page.getByRole("button", { name: "Go live" }).click();
      await page.waitForTimeout(900);
    },
  },
  {
    name: "overview-tune-up-dismissed",
    scenario: {},
    act: async (page) => {
      await page.getByRole("button", { name: "Dismiss for 30 days" }).first().click();
      await page.waitForTimeout(900);
    },
  },
  { name: "overview-many-providers", scenario: { aiProviders: "many" } },
  { name: "overview-paused", scenario: { aiPaused: true } },
  { name: "overview-no-provider", scenario: { aiProviders: "none" } },
  {
    name: "overview-policy-editor",
    scenario: {},
    act: async (page) => {
      await page.getByRole("button", { name: "Edit", exact: true }).click();
      await page.waitForTimeout(700);
    },
  },
  { name: "overview-narrow-1000", scenario: {}, viewport: { width: 1000, height: 900 } },
];

async function setScenario(overrides) {
  const response = await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ ...DEFAULT_SCENARIO, ...overrides }),
  });
  return response.json();
}

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium-1194/chrome-linux/chrome",
});
const failures = [];

for (const shot of SHOTS) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  const { anchor } = await setScenario(shot.scenario);
  const page = await browser.newPage({
    viewport: shot.viewport ?? { width: 1440, height: 1000 },
    colorScheme: shot.colorScheme ?? "dark",
  });
  await page.clock.install({ time: new Date(anchor * 1000) });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().includes("Failed to load resource")) {
      errors.push(message.text());
    }
  });
  await page.goto(`${PAGE}${shot.query ?? ""}`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(SETTLE_MS);
  try {
    if (shot.act) {
      await shot.act(page);
      await page.waitForTimeout(600);
    }
  } catch (error) {
    errors.push(`interaction failed: ${error.message.split("\n")[0]}`);
  }
  const file = `${OUT}/${shot.name}.png`;
  await page.screenshot({ path: file });
  console.log(`${errors.length ? "!" : "✓"} ${shot.name} → ${file}`);
  for (const error of errors) console.log(`    ${error.slice(0, 300)}`);
  if (errors.length) failures.push(shot.name);
  await page.close();
}

await setScenario({});
await browser.close();
if (failures.length) process.exitCode = 1;
