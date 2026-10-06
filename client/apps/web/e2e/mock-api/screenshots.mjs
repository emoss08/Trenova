import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const PAGE = `${APP}/shipment-management/shipments`;
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 9000);

const DEFAULT_SCENARIO = { board: "high", ai: true, operationType: "both", hos: true, maps: false };

const SHOTS = [
  { name: "board-dark", scenario: {} },
  { name: "board-light", scenario: {}, colorScheme: "light" },
  { name: "expanded-row", scenario: {}, act: expandFirstRow },
  { name: "activity-tab", scenario: {}, act: (page) => page.getByRole("tab", { name: "Activity" }).click() },
  { name: "quick-filter-menu", scenario: {}, act: openQuickFilters },
  { name: "capacity-popover", scenario: {}, act: openFirstCapacityUnit },
  { name: "carriers-tab", scenario: {}, act: (page) => page.getByRole("tab", { name: /Carriers/ }).click() },
  { name: "timeline-view", scenario: {}, act: (page) => page.getByRole("radio", { name: "Timeline" }).click() },
  { name: "map-not-connected", scenario: {}, act: openMapView },
  { name: "panel-closed", scenario: {}, act: closePanel },
  { name: "asset-no-ai", scenario: { operationType: "asset", ai: false } },
  { name: "brokerage", scenario: { operationType: "brokerage" } },
  { name: "asset-no-hos", scenario: { operationType: "asset", hos: false } },
  { name: "quiet-board", scenario: { board: "quiet" } },
  { name: "narrow-1000", scenario: {}, viewport: { width: 1000, height: 820 } },
  { name: "narrow-800", scenario: {}, viewport: { width: 800, height: 820 } },
];

async function expandFirstRow(page) {
  await page.locator("tbody tr").filter({ hasText: /S2610-/ }).first().click();
}

async function openQuickFilters(page) {
  await page.getByRole("textbox", { name: "Search shipments" }).focus();
}

async function openFirstCapacityUnit(page) {
  await page.getByRole("list", { name: "Available drivers" }).getByRole("button").first().click();
}

async function openMapView(page) {
  await closePanel(page);
  await page.getByRole("radio", { name: "Map" }).click();
}

async function closePanel(page) {
  await page.getByRole("button", { name: "Close panel" }).click();
}

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
    viewport: shot.viewport ?? { width: 1440, height: 900 },
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
  await page.goto(PAGE, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(SETTLE_MS);
  try {
    if (shot.act) {
      await shot.act(page);
      await page.waitForTimeout(1200);
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
