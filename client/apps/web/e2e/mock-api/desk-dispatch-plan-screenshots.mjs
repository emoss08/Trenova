// Screenshots of a dispatch plan on the Desk: six new loads the Dispatch desk
// planned, at every point the plan can stand at. Run with the mock API and the
// web app up (see README.md):
//
//   bun e2e/mock-api/desk-dispatch-plan-screenshots.mjs [outDir] [name,name,...]
//
// Each shot sets the `deskDispatchPlan` stage it needs: on the artifact's own
// page in light and dark, beside the conversation, and in a narrow window.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/desk-dispatch-plan");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 6000);
const PANE = "/desk/c/athr_desk_dispatch_plan/a/drivers-for-6-loads";
const PAGE = "/desk/v/athr_desk_dispatch_plan/drivers-for-6-loads";
const STAGES = ["plan", "proposed", "assigned", "none"];

function shots() {
  const out = [];
  for (const stage of STAGES) {
    for (const colorScheme of ["light", "dark"]) {
      out.push({ name: `${stage}-${colorScheme}`, path: PAGE, stage, colorScheme, full: true });
    }
  }
  out.push(
    { name: "plan-pane-light", path: PANE, stage: "plan", colorScheme: "light" },
    {
      name: "plan-narrow-dark",
      path: PAGE,
      stage: "plan",
      colorScheme: "dark",
      width: 600,
      full: true,
    },
    {
      name: "plan-assign-light",
      path: PAGE,
      stage: "plan",
      colorScheme: "light",
      full: true,
      act: (page) => page.getByRole("button", { name: "Assign 4 loads" }).click(),
    },
    { name: "skeleton-light", path: PAGE, stage: "plan", colorScheme: "light", hold: true },
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
  const { anchor } = await setScenario({ deskDispatchPlan: shot.stage });
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
    await page.route(/\/graphql\/?(\?.*)?$/, async (route) => {
      const body = route.request().postData() ?? "";
      if (body.includes("DispatchMovesByIds")) return;
      await route.continue();
    });
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

await setScenario({ deskDispatchPlan: "plan" });
await browser.close();
if (failures.length) process.exitCode = 1;
