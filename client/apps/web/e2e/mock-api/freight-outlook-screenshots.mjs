// Screenshots of the freight outlook against the mock API.
//
//   node e2e/mock-api/server.mjs
//   pnpm --filter @trenova/web dev
//   node e2e/mock-api/freight-outlook-screenshots.mjs [outDir] [only,names]

import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const PAGE = `${APP}/dispatch/outlook`;
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/freight-outlook");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 6000);
// 2 gives captures sharp enough to publish; 1 is enough to check a layout.
const SCALE = Number(process.env.SCALE ?? 1);
// A capture for a page that shows it small reads better from a smaller window with the
// sidebar folded: VIEWPORT=1200x750 SIDEBAR=collapsed.
const [WIDTH, HEIGHT] = (process.env.VIEWPORT ?? "1440x900").split("x").map(Number);
const COLLAPSED = process.env.SIDEBAR === "collapsed";

async function openArea(page, name) {
  await page
    .getByRole("button", { name: new RegExp(name) })
    .first()
    .click();
  await page.waitForTimeout(1500);
}

/** The tour, walked forward to the step with this title. */
async function tourTo(page, title) {
  await page
    .getByRole("button", { name: /^About/ })
    .first()
    .click();
  await page.getByRole("button", { name: "Take the tour" }).click();
  for (let step = 0; step < 12; step += 1) {
    if (await page.getByText(title, { exact: true }).isVisible()) return;
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(450);
  }
  throw new Error(`the tour has no step titled ${title}`);
}

const SHOTS = [
  { name: "outlook" },
  { name: "outlook-light", colorScheme: "light" },
  {
    name: "horizon-48",
    act: (page) => page.getByRole("radio", { name: "48 h" }).click(),
  },
  { name: "area", act: (page) => openArea(page, "Cleveland, OH") },
  { name: "area-light", colorScheme: "light", act: (page) => openArea(page, "Cleveland, OH") },
  {
    name: "move-hover",
    act: async (page) => {
      await page.getByRole("button", { name: /^Send/ }).first().hover();
      await page.waitForTimeout(600);
    },
  },
  {
    name: "move-hover-light",
    colorScheme: "light",
    act: async (page) => {
      await page.getByRole("button", { name: /^Send/ }).first().hover();
      await page.waitForTimeout(600);
    },
  },
  { name: "tour-record", act: (page) => tourTo(page, "How far to trust it") },
  {
    name: "tour-record-light",
    colorScheme: "light",
    act: (page) => tourTo(page, "How far to trust it"),
  },
  { name: "narrow-1000", viewport: { width: 1000, height: 820 } },
];

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium",
});
const failures = [];

for (const shot of SHOTS) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  // Every shot starts from the same outlook, whatever the one before it sent.
  await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: "{}",
  });
  const colorScheme = shot.colorScheme ?? "dark";
  const page = await browser.newPage({
    viewport: shot.viewport ?? { width: WIDTH, height: HEIGHT },
    deviceScaleFactor: SCALE,
    colorScheme,
  });
  await page.addInitScript((theme) => localStorage.setItem("trenova-ui-theme", theme), colorScheme);
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
    if (COLLAPSED && !shot.viewport) {
      await page.getByRole("button", { name: "Hide sidebar" }).first().click();
      await page.waitForTimeout(600);
    }
    await shot.act?.(page);
    await page.waitForTimeout(400);
  } catch (error) {
    errors.push(`interaction failed: ${error.message.split("\n")[0]}`);
  }
  await page.screenshot({ path: `${OUT}/${shot.name}.png` });
  if (errors.length > 0) failures.push({ name: shot.name, errors: [...new Set(errors)] });
  console.log(`${errors.length > 0 ? "✗" : "✓"} ${shot.name}`);
  await page.close();
}

await browser.close();
if (failures.length > 0) {
  console.error(JSON.stringify(failures, null, 2));
  process.exitCode = 1;
}
