// Screenshots of a Desk case and of a reply under way. Run with the mock API
// and the web app up (see README.md):
//
//   node e2e/mock-api/desk-case-screenshots.mjs [outDir] [name,name,...]
//
// Every shot starts from a fresh scenario, so the writes one makes (a snooze,
// a wait, a tick, a steer) never leak into the next.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/desk-case");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 6000);
const CASE = "/desk/t/athr_desk_case";
const LIVE = "/desk/t/athr_desk_live";

const caseMenu = (page) => page.getByRole("button", { name: "Case", exact: true }).first().click();

function shotsFor(colorScheme) {
  return [
    { name: "case", path: CASE },
    {
      name: "case-checklist",
      path: CASE,
      act: (page) =>
        page
          .getByRole("button", { name: /Ready to bill/ })
          .first()
          .click(),
    },
    { name: "case-menu", path: CASE, act: caseMenu },
    {
      name: "case-snoozed",
      path: CASE,
      act: async (page) => {
        await caseMenu(page);
        await page.getByRole("menuitem", { name: /Tomorrow morning/ }).click();
      },
    },
    {
      name: "case-waiting-customer",
      path: CASE,
      act: async (page) => {
        await caseMenu(page);
        await page.getByRole("menuitem", { name: /The customer, Acme Foods/ }).click();
      },
    },
    {
      name: "case-ticked",
      path: CASE,
      act: async (page) => {
        await page
          .getByRole("button", { name: /Ready to bill/ })
          .first()
          .click();
        await page.getByRole("checkbox", { name: /Lumper receipt on file/ }).click();
      },
    },
    { name: "checklist-settings", path: `${CASE}?settings=checklists`, keepWorkspace: true },
    { name: "live", path: LIVE, scenario: { deskReply: "working" }, settle: 9000 },
    {
      name: "live-steered",
      path: LIVE,
      scenario: { deskReply: "working" },
      settle: 9000,
      act: async (page) => {
        await page.getByRole("textbox").last().fill("Leave out anything delivered on Sunday");
        await page.keyboard.press("Enter");
      },
    },
    { name: "live-idle", path: LIVE },
  ].map((shot) => ({ ...shot, name: `${shot.name}-${colorScheme}`, colorScheme }));
}

async function setScenario(patch) {
  const response = await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ deskReply: "idle", ...patch }),
  });
  return response.json();
}

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium-1194/chrome-linux/chrome",
});
const failures = [];

for (const shot of [...shotsFor("light"), ...shotsFor("dark")]) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  const { anchor } = await setScenario(shot.scenario ?? {});
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 2,
    colorScheme: shot.colorScheme,
  });
  await context.addInitScript((theme) => {
    window.localStorage.setItem("vite-ui-theme", theme);
  }, shot.colorScheme);
  const page = await context.newPage();
  await page.clock.install({ time: new Date(anchor * 1000) });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().includes("Failed to load resource")) {
      errors.push(message.text());
    }
  });
  await page.goto(`${APP}${shot.path}`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(shot.settle ?? SETTLE_MS);
  if (!shot.keepWorkspace) {
    // These conversations have made nothing, so the empty workspace is folded away.
    await page
      .getByRole("button", { name: /^Workspace/ })
      .first()
      .click();
    await page.waitForTimeout(800);
  }
  if (shot.act) {
    try {
      await shot.act(page);
      await page.waitForTimeout(1500);
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
