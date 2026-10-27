// Screenshots of a report run on the Desk: Billing Desk ran revenue by customer
// for the last 30 days, after previewing it. Run with the mock API and the web
// app up (see README.md):
//
//   node e2e/mock-api/desk-report-screenshots.mjs [outDir] [name,name,...]
//
// Each shot sets the `deskReport` stage it needs.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/desk-report");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 4000);
const RUN = "/desk/c/athr_desk_report/a/revenue-by-customer";
const PREVIEW = "/desk/c/athr_desk_report/a/revenue-by-customer-preview";

const page = (path) => `${path}/page`;

const shots = [
  { name: "finished-light", path: page(RUN), stage: "finished", colorScheme: "light", full: true },
  { name: "finished-dark", path: page(RUN), stage: "finished", colorScheme: "dark", full: true },
  { name: "trend-light", path: page(RUN), stage: "trend", colorScheme: "light", full: true },
  { name: "trend-dark", path: page(RUN), stage: "trend", colorScheme: "dark", full: true },
  { name: "running-light", path: page(RUN), stage: "running", colorScheme: "light", full: true },
  { name: "failed-light", path: page(RUN), stage: "failed", colorScheme: "light", full: true },
  { name: "empty-light", path: page(RUN), stage: "empty", colorScheme: "light", full: true },
  { name: "capped-light", path: page(RUN), stage: "capped", colorScheme: "light", full: true },
  { name: "expired-dark", path: page(RUN), stage: "expired", colorScheme: "dark", full: true },
  {
    name: "scheduled-light",
    path: page(RUN),
    stage: "scheduled",
    colorScheme: "light",
    full: true,
  },
  { name: "withheld-light", path: page(RUN), stage: "withheld", colorScheme: "light", full: true },
  { name: "older-light", path: page(RUN), stage: "older", colorScheme: "light", full: true },
  {
    name: "preview-light",
    path: page(PREVIEW),
    stage: "finished",
    colorScheme: "light",
    full: true,
  },
  { name: "preview-dark", path: page(PREVIEW), stage: "finished", colorScheme: "dark", full: true },
  { name: "finished-pane-light", path: RUN, stage: "finished", colorScheme: "light" },
  { name: "running-pane-dark", path: RUN, stage: "running", colorScheme: "dark" },
  {
    name: "finished-narrow-light",
    path: page(RUN),
    stage: "finished",
    colorScheme: "light",
    width: 600,
    full: true,
  },
];

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium-1194/chrome-linux/chrome",
});

for (const shot of shots) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  const scenario = await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ deskReport: shot.stage }),
  }).then((response) => response.json());
  const tab = await browser.newPage({
    viewport: { width: shot.width ?? 1440, height: 1000 },
    colorScheme: shot.colorScheme,
    deviceScaleFactor: 2,
  });
  await tab.addInitScript((theme) => {
    window.localStorage.setItem("vite-ui-theme", theme);
  }, shot.colorScheme);
  await tab.clock.install({ time: new Date(scenario.anchor * 1000) });
  await tab.goto(`${APP}${shot.path}`);
  await tab.waitForSelector(".dk-rr", { timeout: 180000 });
  await tab.waitForTimeout(SETTLE_MS);
  const file = `${OUT}/${shot.name}.png`;
  if (shot.full) {
    const box = await tab.locator(".dk-rr").first().boundingBox();
    await tab.screenshot({
      path: file,
      fullPage: true,
      clip: {
        x: 0,
        y: 0,
        width: shot.width ?? 1440,
        height: Math.ceil((box?.y ?? 0) + (box?.height ?? 1000) + 24),
      },
    });
  } else {
    const pane = await tab.locator(".dk-apx").first().boundingBox();
    await tab.screenshot({ path: file, clip: pane ?? undefined });
  }
  console.log(`✓ ${file}`);
  await tab.close();
}

await browser.close();
