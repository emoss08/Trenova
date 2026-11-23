// Screenshots of boards on the Desk: Dispatch Desk answered three questions as
// boards (six late loads, three carriers head to head, the October close), and
// in a second conversation placed five small boards in the conversation as the
// reply itself (eight late loads, the same loads as show_answer made them, Acme
// Foods' credit, a carrier comparison and its revision). Run with the mock API and the web app up (see README.md):
//
//   bun e2e/mock-api/desk-board-screenshots.mjs [outDir] [name,name,...]
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
const INLINE = "/desk/t/athr_desk_board_inline";
// What each conversation shot waits for before it is taken.
const DRAWN = "text=From the board for this answer";
const READING = '[aria-label="Reading the board…"]';
const UNREAD = "text=The exhibits could not be read just now";

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
  // Boards in the conversation, drawn as the reply. The room scrolls inside the
  // page, so these shots take a viewport tall enough for the whole conversation.
  { name: "inline-live-light", path: INLINE, stage: "live", colorScheme: "light", wait: DRAWN },
  { name: "inline-live-dark", path: INLINE, stage: "live", colorScheme: "dark", wait: DRAWN },
  { name: "inline-moved-light", path: INLINE, stage: "moved", colorScheme: "light", wait: DRAWN },
  {
    name: "inline-moved-popover-dark",
    path: INLINE,
    stage: "moved",
    colorScheme: "dark",
    wait: DRAWN,
    hover: 'button[aria-label^="Still owed on them"]',
  },
  {
    name: "inline-reading-popover-light",
    path: INLINE,
    stage: "live",
    colorScheme: "light",
    wait: DRAWN,
    hover: 'button[aria-label^="Late now"]',
  },
  // The board show_answer made: two groups side by side, each with a note
  // under its tab, one of which places a reading.
  {
    name: "inline-answer-caption-light",
    path: INLINE,
    stage: "live",
    colorScheme: "light",
    wait: DRAWN,
    hover: 'button[aria-label^="Waiting on his hours"]',
  },
  {
    name: "inline-answer-caption-locked-dark",
    path: INLINE,
    stage: "locked",
    colorScheme: "dark",
    wait: DRAWN,
    hover: 'button[aria-label^="Furthest behind"] >> nth=1',
  },
  {
    name: "inline-cite-light",
    path: INLINE,
    stage: "live",
    colorScheme: "light",
    wait: DRAWN,
    hover: ".dk-bd-hl >> nth=0",
  },
  { name: "inline-locked-dark", path: INLINE, stage: "locked", colorScheme: "dark", wait: DRAWN },
  {
    name: "inline-reading-light",
    path: INLINE,
    stage: "reading",
    colorScheme: "light",
    wait: READING,
  },
  {
    name: "inline-unreadable-dark",
    path: INLINE,
    stage: "unreadable",
    colorScheme: "dark",
    wait: UNREAD,
  },
  {
    name: "inline-narrow-light",
    path: INLINE,
    stage: "live",
    colorScheme: "light",
    width: 600,
    wait: DRAWN,
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
  const conversation = shot.wait !== undefined;
  const tab = await browser.newPage({
    viewport: { width: shot.width ?? 1440, height: conversation ? 3200 : 1000 },
    colorScheme: shot.colorScheme,
    deviceScaleFactor: 2,
  });
  await tab.addInitScript((theme) => {
    window.localStorage.setItem("vite-ui-theme", theme);
  }, shot.colorScheme);
  await tab.clock.install({ time: new Date(scenario.anchor * 1000) });
  await tab.goto(`${APP}${shot.path}`);
  if (conversation) {
    await tab.locator(shot.wait).first().waitFor({ timeout: 180000 });
  } else {
    await tab.waitForSelector(shot.stage === "reading" ? ".dk-bd[aria-busy]" : ".dk-bd-foot", {
      timeout: 180000,
    });
  }
  await tab.waitForTimeout(SETTLE_MS);
  if (shot.hover) {
    await tab.locator(shot.hover).first().hover();
    await tab.waitForTimeout(600);
  }
  const file = `${OUT}/${shot.name}.png`;
  if (conversation) {
    await tab.screenshot({ path: file });
  } else if (shot.full) {
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
