// Screenshots of the Lading studio against the mock API: Acme Retail Group's project, its draft
// version 3 being written, tested, run and replayed, and version 3 in shadow.
//
//   node e2e/mock-api/server.mjs
//   pnpm --filter @trenova/web dev
//   node e2e/mock-api/lading-studio-screenshots.mjs [outDir] [name,name,...]
//
// Every shot starts from a fresh scenario, so a save, a certify or a shadow one makes never
// leaks into the next. Shots are 1600 by 1000 at 1.5x, the 2400 by 1500 the site's posts use.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";
import { cursorAt, pointAt } from "./editor-points.mjs";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/lading-studio");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 6000);
const STEP_MS = Number(process.env.STEP_MS ?? 1500);

const ACME = "/edi/lading/edip_lading_acme";
const ACTIVE = `${ACME}?version=ldv_acme_2`;
const SHADOW = `${ACME}?version=ldv_acme_3`;
const NORTHWIND = "/edi/lading/edip_lading_northwind";

const pause = (page, ms = STEP_MS) => page.waitForTimeout(ms);

// React's development build warns that useEdi calls use() only until the module has loaded; the
// studio renders the same either way, so the warning is not counted as a failure.
const KNOWN_DEV_WARNINGS = ["did not call use() when it finished"];

/** Opens a file or sample from the studio's file tree by the name the tree shows. */
const open = (name) => async (page) => {
  await page
    .getByRole("navigation", { name: "Project files" })
    .getByRole("button", { name: new RegExp(`^${name}(?![\\w.-])`) })
    .first()
    .click();
  await pause(page);
};

/** Opens one of the panels under the editor. */
const panel = (name) => async (page) => {
  await page.getByRole("tab", { name: new RegExp(`^${name}`) }).click();
  await pause(page, 900);
};

/** Runs the project's tests from the Tests panel. */
async function runTests(page) {
  await panel("Tests")(page);
  await page.getByRole("button", { name: "Run the tests" }).click();
  await pause(page);
}

/** Runs tenderIn on the tender with a PO number, its lookups answered from the organisation. */
async function runWithLiveData(page) {
  await panel("Run")(page);
  await page.getByRole("combobox", { name: "Map" }).click();
  await pause(page, 500);
  await page.getByRole("option", { name: /^tenderIn/ }).click();
  await pause(page, 500);
  await page.getByRole("combobox", { name: "Sample" }).click();
  await pause(page, 500);
  await page.getByRole("option", { name: /tender-with-po/ }).click();
  await pause(page, 500);
  await page.getByRole("button", { name: "Run with live data" }).click();
  await pause(page);
}

/** Makes the bottom panel taller, for a shot about what is in it. */
async function raisePanels(page, by = 260) {
  const handle = page.getByRole("separator").last();
  const box = await handle.boundingBox();
  if (!box) return;
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width / 2, box.y - by, { steps: 8 });
  await page.mouse.up();
  await pause(page, 600);
}

const steps =
  (...actions) =>
  async (page) => {
    for (const action of actions) await action(page);
  };

function shotsFor(colorScheme) {
  return [
    { name: "partner-file", path: ACME },
    {
      name: "map",
      path: ACME,
      act: steps(open("tender-in")),
    },
    {
      name: "map-hover",
      path: ACME,
      act: steps(open("tender-in"), async (page) => {
        const point = await pointAt(page, "B204", 2);
        await page.mouse.move(point.x, point.y);
        await pause(page, 1200);
      }),
    },
    {
      name: "map-checked",
      path: ACME,
      act: steps(open("tender-in"), async (page) => {
        await cursorAt(page, "custom.po_number", "custom.po_number".length);
        await page.keyboard.press("Backspace");
        await page.keyboard.press("Backspace");
        await page.keyboard.type("br");
        await pause(page);
        await panel("Problems")(page);
      }),
    },
    {
      name: "map-complete",
      path: ACME,
      act: steps(open("tender-in"), async (page) => {
        await cursorAt(page, "custom.po_number", "custom.".length);
        await page.keyboard.down("Shift");
        await page.keyboard.press("End");
        await page.keyboard.up("Shift");
        await page.keyboard.press("Backspace");
        await pause(page, 400);
        await page.keyboard.press("Control+Space");
        await pause(page, 1200);
      }),
    },
    { name: "rules", path: ACME, act: open("tender-checks") },
    { name: "flow", path: ACME, act: open("tender") },
    {
      name: "tests",
      path: ACME,
      act: steps(open("tender-in"), runTests, raisePanels),
    },
    {
      name: "run",
      path: ACME,
      act: steps(open("tender-in"), runWithLiveData, raisePanels),
    },
    {
      name: "run-lineage",
      path: ACME,
      act: steps(open("tender-in"), runWithLiveData, async (page) => {
        const point = await pointAt(page, "party.N102.titleCase", -11);
        await page.mouse.move(point.x, point.y);
        await pause(page, 1200);
      }),
    },
    {
      name: "release-replay",
      path: ACME,
      act: steps(open("tender-checks"), panel("Release"), raisePanels),
    },
    {
      name: "release-shadow",
      path: SHADOW,
      scenario: { lading: "shadow" },
      act: steps(open("tender-in"), panel("Release"), raisePanels),
    },
    { name: "active-settings", path: ACTIVE, act: panel("Settings") },
    { name: "versions", path: ACME, act: (page) => page.getByRole("combobox").first().click() },
    { name: "start", path: NORTHWIND },
  ].map((shot) => ({ ...shot, name: `${shot.name}-${colorScheme}`, colorScheme }));
}

async function setScenario(patch) {
  const response = await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ lading: "draft", ...patch }),
  });
  return response.json();
}

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch(
  process.env.CHROMIUM_PATH ? { executablePath: process.env.CHROMIUM_PATH } : {},
);
const failures = [];

for (const shot of [...shotsFor("light"), ...shotsFor("dark")]) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  await setScenario(shot.scenario ?? {});
  const context = await browser.newContext({
    viewport: shot.viewport ?? { width: 1600, height: 1000 },
    deviceScaleFactor: Number(process.env.SCALE ?? 1.5),
    colorScheme: shot.colorScheme,
  });
  await context.addInitScript((theme) => {
    window.localStorage.setItem("vite-ui-theme", theme);
  }, shot.colorScheme);
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    const text = message.text();
    if (
      message.type() === "error" &&
      !text.includes("Failed to load resource") &&
      !KNOWN_DEV_WARNINGS.some((warning) => text.includes(warning))
    ) {
      errors.push(text);
    }
  });
  await page.goto(`${APP}${shot.path}`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(shot.settle ?? SETTLE_MS);
  if (shot.act) {
    try {
      await shot.act(page);
      await page.waitForTimeout(600);
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

await setScenario({});
await browser.close();
if (failures.length) process.exitCode = 1;
