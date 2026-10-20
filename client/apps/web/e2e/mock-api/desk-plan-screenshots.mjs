// Screenshots of a plan on the Desk: the Dispatch agent's four steps to
// recover a load, drawn at every point the plan can stand at. Run with the
// mock API and the web app up (see README.md):
//
//   node e2e/mock-api/desk-plan-screenshots.mjs [outDir] [name,name,...]
//
// Each shot sets the `deskPlan` stage it needs. Every stage is shot on the
// artifact's own page in light and dark, and the ones a person works from
// beside the conversation and in a narrow window.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/desk-plan");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 6000);
const PANE = "/desk/c/athr_desk_plan/a/recover-s-240201";
const PAGE = `${PANE}/page`;
const STAGES = [
  "awaiting",
  "held",
  "approving",
  "running",
  "failed",
  "done",
  "declined",
  "expired",
  "superseded",
];

const openFold = (index) => async (page) => {
  await page.locator(".dk-pln-fold").nth(index).click();
};

function shots() {
  const out = [];
  for (const stage of STAGES) {
    for (const colorScheme of ["light", "dark"]) {
      out.push({ name: `${stage}-${colorScheme}`, path: PAGE, stage, colorScheme, full: true });
    }
  }
  out.push(
    { name: "awaiting-pane-light", path: PANE, stage: "awaiting", colorScheme: "light" },
    { name: "running-pane-dark", path: PANE, stage: "running", colorScheme: "dark" },
    {
      name: "failed-narrow-light",
      path: PAGE,
      stage: "failed",
      colorScheme: "light",
      width: 600,
      full: true,
    },
    {
      name: "awaiting-narrow-dark",
      path: PAGE,
      stage: "awaiting",
      colorScheme: "dark",
      width: 600,
      full: true,
    },
    {
      name: "done-unfolded-light",
      path: PAGE,
      stage: "done",
      colorScheme: "light",
      full: true,
      act: openFold(2),
    },
    {
      name: "awaiting-decline-light",
      path: PAGE,
      stage: "awaiting",
      colorScheme: "light",
      act: (page) => page.getByRole("button", { name: "Decline", exact: true }).click(),
    },
    {
      name: "skeleton-light",
      path: PAGE,
      stage: "awaiting",
      colorScheme: "light",
      hold: true,
    },
  );
  return out;
}

async function setScenario(patch) {
  const response = await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(patch),
  });
  return response.json();
}

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium-1194/chrome-linux/chrome",
});
const failures = [];

for (const shot of shots()) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  const { anchor } = await setScenario({ deskPlan: shot.stage });
  const context = await browser.newContext({
    viewport: { width: shot.width ?? 1440, height: 900 },
    deviceScaleFactor: 2,
    colorScheme: shot.colorScheme,
  });
  await context.addInitScript((theme) => {
    window.localStorage.setItem("vite-ui-theme", theme);
  }, shot.colorScheme);
  const page = await context.newPage();
  await page.clock.install({ time: new Date(anchor * 1000) });
  if (shot.hold) {
    await page.route(/\/assistant\/threads\/athr_desk_plan\/(plans|proposals)\/?$/, () => {});
  }
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().includes("Failed to load resource")) {
      errors.push(message.text());
    }
  });
  await page.goto(`${APP}${shot.path}`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(shot.settle ?? SETTLE_MS);
  if (shot.act) {
    try {
      await shot.act(page);
      await page.waitForTimeout(1200);
    } catch (error) {
      errors.push(`step: ${error.message.split("\n")[0]}`);
    }
  }
  const file = `${OUT}/${shot.name}.png`;
  await page.screenshot({ path: file, fullPage: shot.full === true });
  console.log(`${errors.length ? "!" : "✓"} ${shot.name} → ${file}`);
  for (const error of errors) console.log(`    ${error.slice(0, 300)}`);
  if (errors.length) failures.push(shot.name);
  await context.close();
}

await setScenario({ deskPlan: "awaiting" });
await browser.close();
if (failures.length) process.exitCode = 1;
