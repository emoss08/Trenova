// Screenshots of the Desk's extraction artifact: what document intelligence
// read off a file, drawn over the file's own pages. Run with the mock API and
// the web app up (see README.md):
//
//   node e2e/mock-api/desk-extraction-screenshots.mjs [outDir] [name,name,...]
//
// Each shot sets the `extraction` variant and `extractionStage` it needs. Every
// shot also measures where each drawn box sits against the drawn page and
// prints the worst miss in CSS pixels, so a box that drifts off its text fails
// the walkthrough instead of waiting for someone to notice it in a picture.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/desk-extraction");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 5000);
const PANE = "/desk/c/athr_desk_extract/a/northline-rate-confirmation";
const PAGE = `${PANE}/page`;
const DESKTOP = { width: 1440, height: 900 };
const PHONE = { width: 390, height: 844 };

const field = (name) => (page) =>
  page.getByRole("button", { name: new RegExp(`^${name}`) }).first();

const selectField = (name) => async (page) => {
  const row = field(name)(page);
  await row.evaluate((node) => node.scrollIntoView({ block: "end" }));
  await row.click();
};

const zoomIn = (times) => async (page) => {
  for (let i = 0; i < times; i += 1) {
    await page.getByRole("button", { name: "Zoom in" }).first().click();
  }
};

/**
 * Every drawn box against the page it sits on: the box's place as a fraction of the page
 * image's own box, compared with the fraction the payload gave it. Returns the worst miss in
 * CSS pixels, and how the page frame's shape compares with the image's.
 */
async function measureBoxes(page) {
  return page.evaluate(() => {
    const image = document.querySelector("[data-extraction-page] img.dk-ex-img, .dk-ex-img");
    if (!image || !image.complete || image.naturalWidth === 0) return null;
    const frame = image.getBoundingClientRect();
    const boxes = [...document.querySelectorAll("[data-box-x], .dk-ex-box")];
    let worst = 0;
    for (const box of boxes) {
      const outer = box.getBoundingClientRect();
      const style = getComputedStyle(box);
      const left = parseFloat(style.paddingLeft) || 0;
      const top = parseFloat(style.paddingTop) || 0;
      const rect = {
        left: outer.left + left,
        top: outer.top + top,
        width: outer.width - left - (parseFloat(style.paddingRight) || 0),
        height: outer.height - top - (parseFloat(style.paddingBottom) || 0),
      };
      const expect = box.dataset.boxX
        ? {
            x: Number(box.dataset.boxX),
            y: Number(box.dataset.boxY),
            w: Number(box.dataset.boxW),
            h: Number(box.dataset.boxH),
          }
        : {
            x: parseFloat(box.style.left) / 100,
            y: parseFloat(box.style.top) / 100,
            w: parseFloat(box.style.width) / 100,
            h: parseFloat(box.style.height) / 100,
          };
      const misses = [
        Math.abs(rect.left - (frame.left + expect.x * frame.width)),
        Math.abs(rect.top - (frame.top + expect.y * frame.height)),
        Math.abs(rect.width - expect.w * frame.width),
        Math.abs(rect.height - expect.h * frame.height),
      ];
      worst = Math.max(worst, ...misses);
    }
    const shape = Math.abs(frame.width / frame.height - image.naturalWidth / image.naturalHeight);
    return { boxes: boxes.length, worstPx: Math.round(worst * 10) / 10, shapeMiss: shape };
  });
}

function shotsFor(colorScheme) {
  return [
    { name: "pane", path: PANE },
    { name: "pane-select-page2", path: PANE, act: selectField("Name: Anyco Clothes #612") },
    { name: "pane-hover-box", path: PANE, act: (page) => field("Commodity")(page).hover() },
    {
      name: "pane-zoom",
      path: PANE,
      act: async (page) => {
        await zoomIn(3)(page);
        await selectField("Rate")(page);
      },
    },
    {
      name: "pane-resized",
      path: PANE,
      act: async (page) => {
        await selectField("Load #")(page);
        await page.setViewportSize({ width: 1100, height: 800 });
      },
    },
    { name: "page", path: PAGE },
    { name: "page-select", path: PAGE, act: selectField("Name: Anyco Clothes #612") },
    { name: "phone-pane", path: PANE, viewport: PHONE },
    { name: "phone-page", path: PAGE, viewport: PHONE, full: true },
    { name: "phone-select", path: PAGE, viewport: PHONE, act: selectField("Consignee") },
  ].map((shot) => ({ ...shot, name: `${shot.name}-${colorScheme}`, colorScheme }));
}

const CASES = [
  { name: "image", path: PANE, scenario: { extraction: "image" } },
  { name: "legacy", path: PANE, scenario: { extraction: "legacy" } },
  { name: "nobox", path: PANE, scenario: { extraction: "nobox" } },
  { name: "broken", path: PANE, scenario: { extraction: "broken" } },
  { name: "long", path: PANE, scenario: { extraction: "long" } },
  { name: "long-page", path: PAGE, scenario: { extraction: "long" }, full: true },
  {
    name: "loading",
    path: PANE,
    settle: 6000,
    hold: /\/(pages\/\d+\/image|preview)\/?$/,
  },
  {
    name: "preview",
    path: PANE,
    settle: 6000,
    hold: /\/pages\/\d+\/image\/?$/,
  },
  {
    name: "drafting",
    path: PANE,
    act: (page) =>
      page.getByRole("button", { name: "Create shipment", exact: true }).first().click(),
    after: 200,
  },
  {
    name: "checking",
    path: PANE,
    act: (page) =>
      page
        .getByRole("button", { name: /^Ask the agent to check \d+/ })
        .first()
        .click(),
    after: 200,
  },
  { name: "replying", path: PANE, scenario: { extractionStage: "replying" } },
  {
    name: "file-dialog",
    path: PANE,
    act: (page) => page.getByRole("button", { name: "Open file" }).first().click(),
  },
  { name: "proposed", path: PANE, scenario: { extractionStage: "proposed" } },
  { name: "created", path: PANE, scenario: { extractionStage: "created" } },
  { name: "failed", path: PANE, scenario: { extractionStage: "failed" } },
  { name: "rejected", path: PANE, scenario: { extractionStage: "rejected" } },
  { name: "attached", path: PANE, scenario: { extractionStage: "attached" } },
  {
    name: "keyboard",
    path: PANE,
    act: async (page) => {
      await field("Shipper")(page).focus();
      for (let i = 0; i < 3; i += 1) await page.keyboard.press("Tab");
      await page.keyboard.press("Enter");
    },
  },
].map((shot) => ({ ...shot, colorScheme: "light" }));

async function setScenario(patch) {
  const response = await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ extraction: "ratecon", extractionStage: "idle", ...patch }),
  });
  return response.json();
}

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium-1194/chrome-linux/chrome",
});
const failures = [];

for (const shot of [...shotsFor("light"), ...shotsFor("dark"), ...CASES]) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  const { anchor } = await setScenario(shot.scenario ?? {});
  const viewport = shot.viewport ?? DESKTOP;
  const context = await browser.newContext({
    viewport,
    deviceScaleFactor: 2,
    colorScheme: shot.colorScheme,
  });
  await context.addInitScript((theme) => {
    window.localStorage.setItem("vite-ui-theme", theme);
  }, shot.colorScheme);
  const page = await context.newPage();
  if (shot.hold) await page.route(shot.hold, () => {});
  await page.clock.install({ time: new Date(anchor * 1000) });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().includes("Failed to load resource")) {
      errors.push(message.text());
    }
  });
  const started = Date.now();
  await page.goto(`${APP}${shot.path}`, { waitUntil: "domcontentloaded" });
  let firstPageMs = null;
  if (!shot.hold) {
    try {
      await page.waitForFunction(
        () => {
          const image = document.querySelector("[data-extraction-page] img.dk-ex-img, .dk-ex-img");
          return image && image.complete && image.naturalWidth > 0;
        },
        null,
        { timeout: 15000 },
      );
      firstPageMs = Date.now() - started;
    } catch {
      firstPageMs = null;
    }
  }
  await page.waitForTimeout(shot.settle ?? SETTLE_MS);
  if (shot.act) {
    try {
      await shot.act(page);
      await page.waitForTimeout(shot.after ?? 1500);
    } catch (error) {
      errors.push(`step: ${error.message.split("\n")[0]}`);
    }
  }
  const measured = shot.hold ? null : await measureBoxes(page);
  const file = `${OUT}/${shot.name}.png`;
  await page.screenshot({ path: file, fullPage: Boolean(shot.full) });
  const misplaced = measured && measured.worstPx > 1.5;
  console.log(
    `${errors.length || misplaced ? "!" : "✓"} ${shot.name} → ${file}` +
      (firstPageMs === null ? "" : ` · first page ${firstPageMs}ms`) +
      (measured ? ` · ${measured.boxes} boxes, worst ${measured.worstPx}px` : ""),
  );
  for (const error of errors) console.log(`    ${error.slice(0, 300)}`);
  if (errors.length || misplaced) failures.push(shot.name);
  await context.close();
}

await setScenario({});
await browser.close();
if (failures.length) process.exitCode = 1;
