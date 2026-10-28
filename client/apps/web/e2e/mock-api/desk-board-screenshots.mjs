// Screenshots of boards on the Desk: Dispatch Desk answered three questions as
// boards (six late loads, three carriers head to head, the October close). Run
// with the mock API and the web app up (see README.md):
//
//   node e2e/mock-api/desk-board-screenshots.mjs [outDir] [name,name,...]
//
// Each shot sets the `deskBoard` stage it needs.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/desk-board");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 4000);
const THREAD = "/desk/c/athr_desk_board/a";
const LATE = `${THREAD}/six-late-shipments`;
const CARRIERS = `${THREAD}/carriers-head-to-head`;
const CLOSE = `${THREAD}/october-close`;

const page = (path) => `${path}/page`;

const shots = [
  { name: "late-light", path: page(LATE), stage: "live", colorScheme: "light", full: true },
  { name: "late-dark", path: page(LATE), stage: "live", colorScheme: "dark", full: true },
  { name: "carriers-light", path: page(CARRIERS), stage: "live", colorScheme: "light", full: true },
  { name: "carriers-dark", path: page(CARRIERS), stage: "live", colorScheme: "dark", full: true },
  { name: "close-light", path: page(CLOSE), stage: "live", colorScheme: "light", full: true },
  { name: "close-dark", path: page(CLOSE), stage: "live", colorScheme: "dark", full: true },
  { name: "late-moved-light", path: page(LATE), stage: "moved", colorScheme: "light", full: true },
  {
    name: "carriers-moved-dark",
    path: page(CARRIERS),
    stage: "moved",
    colorScheme: "dark",
    full: true,
  },
  {
    name: "late-reading-light",
    path: page(LATE),
    stage: "reading",
    colorScheme: "light",
    full: true,
  },
  {
    name: "close-reading-dark",
    path: page(CLOSE),
    stage: "reading",
    colorScheme: "dark",
    full: true,
  },
  { name: "late-pane-light", path: LATE, stage: "live", colorScheme: "light" },
  { name: "close-pane-dark", path: CLOSE, stage: "live", colorScheme: "dark" },
  {
    name: "late-narrow-light",
    path: page(LATE),
    stage: "live",
    colorScheme: "light",
    width: 600,
    full: true,
  },
  {
    name: "close-narrow-dark",
    path: page(CLOSE),
    stage: "live",
    colorScheme: "dark",
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
    body: JSON.stringify({ deskBoard: shot.stage }),
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
  await tab.waitForSelector(shot.stage === "reading" ? ".dk-bd[aria-busy]" : ".dk-bd-foot", {
    timeout: 180000,
  });
  await tab.waitForTimeout(SETTLE_MS);
  const file = `${OUT}/${shot.name}.png`;
  if (shot.full) {
    const box = await tab.locator(".dk-bd").first().boundingBox();
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
