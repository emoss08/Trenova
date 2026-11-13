// Screenshots of the notification panel against the mock API.
//
//   node e2e/mock-api/server.mjs
//   pnpm --filter @trenova/web dev
//   node e2e/mock-api/notifications-screenshots.mjs [outDir] [only,names]

import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const PAGE = `${APP}/shipment-management/shipments`;
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/notifications");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 6000);

const ROW = "[data-notification-id]";

/** The panel's sheet. On a phone the sidebar is a sheet too, opened first, so it is the last. */
const sheetOf = (page) => page.locator("[data-slot=sheet-content]").last();

async function open(page) {
  await page
    .getByRole("button", { name: /^Notifications/ })
    .first()
    .click();
  await sheetOf(page).waitFor();
  await sheetOf(page).locator(ROW).first().waitFor();
  await page.waitForTimeout(500);
}

async function tab(page, name) {
  await sheetOf(page).getByRole("tab", { name }).click();
  await page.waitForTimeout(700);
}

const row = (page, id) => sheetOf(page).locator(`[data-notification-id="${id}"]`);

const SHOTS = [
  { name: "for-you", act: open },
  { name: "for-you-light", colorScheme: "light", act: open },
  {
    name: "row-hover",
    act: async (p) => {
      await open(p);
      await row(p, "y2").hover();
      await p.waitForTimeout(400);
    },
  },
  {
    name: "later-menu",
    act: async (p) => {
      await open(p);
      await row(p, "y2").focus();
      await p.keyboard.press("h");
      await p.waitForTimeout(500);
    },
  },
  {
    name: "more-menu",
    act: async (p) => {
      await open(p);
      await row(p, "y2").hover();
      await row(p, "y2").getByRole("button", { name: "More" }).click();
      await p.waitForTimeout(500);
    },
  },
  {
    name: "team",
    act: async (p) => {
      await open(p);
      await tab(p, /^Team/);
    },
  },
  {
    name: "team-expanded",
    act: async (p) => {
      await open(p);
      await tab(p, /^Team/);
      await row(p, "t1")
        .getByRole("button", { name: /^Latest/ })
        .click();
      await p.waitForTimeout(500);
    },
  },
  {
    name: "later",
    act: async (p) => {
      await open(p);
      await tab(p, /^Later/);
    },
  },
  {
    name: "done",
    act: async (p) => {
      await open(p);
      await tab(p, /^Done/);
    },
  },
  {
    name: "done-light",
    colorScheme: "light",
    act: async (p) => {
      await open(p);
      await tab(p, /^Done/);
    },
  },
  {
    name: "acknowledged",
    act: async (p) => {
      await open(p);
      await row(p, "y1").getByRole("button", { name: "Acknowledge" }).click();
      await p.waitForTimeout(1200);
    },
  },
  {
    name: "empty",
    act: async (p) => {
      await open(p);
      await sheetOf(p).getByRole("button", { name: "Mark all done" }).click();
      await p.waitForTimeout(1200);
    },
  },
  {
    name: "narrow-390",
    viewport: { width: 390, height: 780 },
    // A phone has no bell in its header: it is in the sidebar, which opens over the page.
    act: async (p) => {
      await p
        .getByRole("button", { name: /sidebar/i })
        .first()
        .click();
      await p.waitForTimeout(600);
      await open(p);
    },
  },
];

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium",
});
const failures = [];

for (const shot of SHOTS) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  // Every shot starts from the same inbox, whatever the one before it finished.
  await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: "{}",
  });
  const colorScheme = shot.colorScheme ?? "dark";
  const page = await browser.newPage({
    viewport: shot.viewport ?? { width: 1440, height: 900 },
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
    await shot.act(page);
    await page.waitForTimeout(400);
  } catch (error) {
    errors.push(`interaction failed: ${error.message.split("\n")[0]}`);
  }
  const sheet = sheetOf(page);
  const file = `${OUT}/${shot.name}.png`;
  if (shot.viewport === undefined && (await sheet.count()) > 0) {
    const box = await sheet.boundingBox();
    await page.screenshot({
      path: file,
      clip: { x: Math.max(0, box.x - 280), y: 0, width: box.width + 296, height: 900 },
    });
  } else {
    await page.screenshot({ path: file });
  }
  if (errors.length > 0) failures.push({ name: shot.name, errors: [...new Set(errors)] });
  console.log(`${errors.length > 0 ? "✗" : "✓"} ${shot.name}`);
  await page.close();
}

await browser.close();
if (failures.length > 0) {
  console.error(JSON.stringify(failures, null, 2));
  process.exitCode = 1;
}
