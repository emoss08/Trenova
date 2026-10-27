// Screenshots of a table that pages on the Desk: Acme Foods' 128 shipments in
// transit, of which the conversation's agent read two pages, paged on with the
// person's own access. Run with the mock API and the web app up (see README.md):
//
//   node e2e/mock-api/desk-table-screenshots.mjs [outDir] [name,name,...]
//
// Each shot sets the `deskTable` stage it needs and, where the state is one a
// person reaches by working the table, takes the steps that reach it.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/desk-table");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 4000);
const PANE = "/desk/c/athr_desk_table/a/acme-shipments-in-transit";
const PAGE = `${PANE}/page`;

const next = (times) => async (page) => {
  for (let at = 0; at < times; at += 1) {
    await page.getByRole("button", { name: "Next page" }).click();
    await page.waitForTimeout(400);
  }
};

const readAgain = async (page) => {
  await page.getByRole("button", { name: "Read again now" }).click();
  await page.waitForTimeout(1200);
};

const shots = [
  { name: "agent-light", path: PAGE, stage: "paged", colorScheme: "light", full: true },
  { name: "agent-dark", path: PAGE, stage: "paged", colorScheme: "dark", full: true },
  {
    name: "live-light",
    path: PAGE,
    stage: "paged",
    colorScheme: "light",
    full: true,
    act: next(3),
  },
  { name: "live-dark", path: PAGE, stage: "paged", colorScheme: "dark", full: true, act: next(3) },
  {
    name: "turning-light",
    path: PAGE,
    stage: "slow",
    colorScheme: "light",
    full: true,
    act: next(3),
    settle: 800,
  },
  {
    name: "search-light",
    path: PAGE,
    stage: "paged",
    colorScheme: "light",
    full: true,
    act: async (page) => {
      await page.getByLabel("Search the list").fill("memphis");
      await page.waitForTimeout(1200);
    },
  },
  {
    name: "sorted-light",
    path: PAGE,
    stage: "paged",
    colorScheme: "light",
    full: true,
    act: async (page) => {
      await page.getByRole("columnheader", { name: "Revenue" }).click();
      await page.waitForTimeout(400);
      await page.getByRole("columnheader", { name: "Revenue" }).click();
      await page.waitForTimeout(1000);
    },
  },
  {
    name: "failed-light",
    path: PAGE,
    stage: "failing",
    colorScheme: "light",
    full: true,
    act: next(3),
  },
  { name: "locked-light", path: PAGE, stage: "locked", colorScheme: "light", full: true },
  { name: "unnamed-light", path: PAGE, stage: "unnamed", colorScheme: "light", full: true },
  {
    name: "again-light",
    path: PAGE,
    stage: "paged",
    colorScheme: "light",
    full: true,
    act: readAgain,
  },
  {
    name: "again-dark",
    path: PAGE,
    stage: "paged",
    colorScheme: "dark",
    full: true,
    act: readAgain,
  },
  {
    name: "again-whole-light",
    path: PAGE,
    stage: "whole",
    colorScheme: "light",
    full: true,
    act: readAgain,
  },
  {
    name: "withheld-dark",
    path: PAGE,
    stage: "withheld",
    colorScheme: "dark",
    full: true,
    act: next(3),
  },
  { name: "cut-light", path: PAGE, stage: "cut", colorScheme: "light", full: true, act: next(2) },
  { name: "whole-light", path: PAGE, stage: "whole", colorScheme: "light", full: true },
  { name: "short-light", path: PAGE, stage: "short", colorScheme: "light", full: true },
  { name: "agent-pane-light", path: PANE, stage: "paged", colorScheme: "light" },
  { name: "live-pane-dark", path: PANE, stage: "paged", colorScheme: "dark", act: next(2) },
  {
    name: "agent-narrow-light",
    path: PAGE,
    stage: "paged",
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
    body: JSON.stringify({ deskTable: shot.stage }),
  }).then((response) => response.json());
  const page = await browser.newPage({
    viewport: { width: shot.width ?? 1440, height: 1000 },
    colorScheme: shot.colorScheme,
    deviceScaleFactor: 2,
  });
  await page.addInitScript((theme) => {
    window.localStorage.setItem("vite-ui-theme", theme);
    window.localStorage.removeItem("desk.table.pageSize");
  }, shot.colorScheme);
  await page.clock.install({ time: new Date(scenario.anchor * 1000) });
  await page.goto(`${APP}${shot.path}`);
  await page.waitForSelector(".dk-tv", { timeout: 180000 });
  await page.waitForTimeout(SETTLE_MS);
  if (shot.act) await shot.act(page);
  await page.waitForTimeout(shot.settle ?? 600);
  const file = `${OUT}/${shot.name}.png`;
  if (shot.full) {
    const box = await page.locator(".dk-tv").first().boundingBox();
    await page.screenshot({
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
    const pane = await page.locator(".dk-apx").first().boundingBox();
    await page.screenshot({ path: file, clip: pane ?? undefined });
  }
  console.log(`✓ ${file}`);
  await page.close();
}

await browser.close();
