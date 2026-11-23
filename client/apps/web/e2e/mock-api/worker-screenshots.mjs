// Walks the worker record page against the mock API and writes one PNG per state: every tab and
// view of Maria Alvarez in light and dark, the panes that matter, the roster with its stepper,
// each variant's Overview and the page at 1280 and 400 wide. The browser clock is frozen at the
// fixture's "now", Sunday 22 November 2026, 4:08 PM UTC (fixtures/worker/base.mjs).
//
// Usage: node e2e/mock-api/worker-screenshots.mjs [outDir] [name,name,...]
// Env: MOCK_API_URL, APP_URL, CHROMIUM_PATH, SETTLE_MS (default 2500), SCALE (default 1).
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";
import { WORKER_NOW, pid } from "./fixtures/worker/base.mjs";
import { MARIA_ID, NEW_HIRE_ID, NOT_CLEARED_ID } from "./fixtures/worker/index.mjs";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/worker");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 2500);
const SCALE = Number(process.env.SCALE ?? 1);
const WIDE = { width: 1600, height: 900 };

const NAMES = {
  [MARIA_ID]: "Maria Alvarez",
  [NOT_CLEARED_ID]: "Darnell Brooks",
  [NEW_HIRE_ID]: "Kenji Tanaka",
};

/** Opens a worker at a tab, view, form or record param, and waits for the header and the tab. */
async function openWorker(page, params = {}, workerId = MARIA_ID) {
  const search = new URLSearchParams(params).toString();
  await page.goto(`${APP}/hr/workers/${workerId}${search ? `?${search}` : ""}`, {
    waitUntil: "domcontentloaded",
  });
  await page.getByText(NAMES[workerId]).first().waitFor({ timeout: 20_000 });
  await page.waitForTimeout(SETTLE_MS);
}

const TAB_VIEWS = [
  ["overview"],
  ["profile"],
  ["employment", "history"],
  ["employment", "checklists"],
  ["qualification", "credentials"],
  ["qualification", "dq-file"],
  ["qualification", "drug-and-alcohol"],
  ["development", "training"],
  ["development", "reviews"],
  ["safety"],
  ["time", "hours-of-service"],
  ["time", "schedule"],
  ["time", "time-off"],
  ["time", "leave"],
  ["pay", "pay-profile"],
  ["pay", "benefits"],
  ["files", "documents"],
  ["files", "driver-portal"],
  ["files", "agent-memory"],
  ["activity"],
];

const areaParams = ([tab, view]) => (view ? { tab, view } : { tab });
const areaName = ([tab, view]) => (view ? `${tab}-${view}` : tab);

const SHOTS = [
  // Every tab and view of Maria, light and dark.
  ...TAB_VIEWS.flatMap((area) => [
    {
      name: `maria-${areaName(area)}-light`,
      colorScheme: "light",
      act: (page) => openWorker(page, areaParams(area)),
    },
    {
      name: `maria-${areaName(area)}-dark`,
      colorScheme: "dark",
      act: (page) => openWorker(page, areaParams(area)),
    },
  ]),

  // The panes and open states that matter.
  {
    name: "pane-add-credential",
    colorScheme: "light",
    act: (page) =>
      openWorker(page, { tab: "qualification", view: "credentials", form: "add-credential" }),
  },
  {
    name: "pane-credential-read",
    colorScheme: "light",
    act: (page) =>
      openWorker(page, { tab: "qualification", view: "credentials", credential: pid("wcr", 2) }),
  },
  {
    name: "pane-credential-read-unverified",
    colorScheme: "dark",
    act: (page) =>
      openWorker(page, { tab: "qualification", view: "credentials", credential: pid("wcr", 4) }),
  },
  {
    name: "pane-add-employer",
    colorScheme: "light",
    act: (page) =>
      openWorker(page, { tab: "qualification", view: "dq-file", form: "add-employer" }),
  },
  {
    name: "pane-record-event-terminated",
    colorScheme: "light",
    act: async (page) => {
      await openWorker(page, {
        tab: "employment",
        view: "history",
        form: "record-event",
        kind: "Terminated",
      });
      await page.getByText("What this will do").first().waitFor({ timeout: 10_000 });
      await page.waitForTimeout(800);
    },
  },
  {
    name: "pane-record-event-terminated-dark",
    colorScheme: "dark",
    act: async (page) => {
      await openWorker(page, {
        tab: "employment",
        view: "history",
        form: "record-event",
        kind: "Terminated",
      });
      await page.getByText("What this will do").first().waitFor({ timeout: 10_000 });
      await page.waitForTimeout(800);
    },
  },
  {
    name: "pane-record-event-transferred",
    colorScheme: "light",
    act: (page) =>
      openWorker(page, {
        tab: "employment",
        view: "history",
        form: "record-event",
        kind: "Transferred",
      }),
  },
  {
    name: "pane-record-training-result",
    colorScheme: "light",
    act: (page) =>
      openWorker(page, {
        tab: "development",
        view: "training",
        form: "record-training-result",
        training: pid("wtr", 1),
      }),
  },
  {
    name: "pane-record-safety-event",
    colorScheme: "light",
    act: (page) => openWorker(page, { tab: "safety", form: "record-safety-event" }),
  },
  {
    name: "pane-safety-event-read",
    colorScheme: "light",
    act: (page) => openWorker(page, { tab: "safety", safetyEvent: pid("wse", 1) }),
  },
  {
    name: "pane-issue-action",
    colorScheme: "light",
    act: (page) => openWorker(page, { tab: "safety", form: "issue-action" }),
  },
  {
    name: "pane-request-time-off",
    colorScheme: "light",
    act: (page) => openWorker(page, { tab: "time", view: "time-off", form: "request-time-off" }),
  },
  {
    name: "pane-profile-change-review",
    colorScheme: "light",
    act: (page) => openWorker(page, { tab: "profile", request: pid("wpc", 2) }),
  },
  {
    name: "pane-checklist",
    colorScheme: "light",
    act: (page) =>
      openWorker(page, { tab: "employment", view: "checklists", checklist: pid("wck", 1) }),
  },
  {
    name: "profile-save-dock",
    colorScheme: "light",
    act: async (page) => {
      await openWorker(page, { tab: "profile" });
      const phone = page.getByLabel("Phone", { exact: false }).first();
      await phone.click();
      await phone.press("End");
      await phone.press("Backspace");
      await phone.type("9");
      const email = page.getByLabel("Email").first();
      await email.fill("maria.alvarez@trenovafreight.example");
      await page.waitForTimeout(800);
    },
  },
  {
    name: "profile-save-dock-dark",
    colorScheme: "dark",
    act: async (page) => {
      await openWorker(page, { tab: "profile" });
      const email = page.getByLabel("Email").first();
      await email.fill("maria.alvarez@trenovafreight.example");
      await page.waitForTimeout(800);
    },
  },
  {
    name: "add-to-record-menu",
    colorScheme: "light",
    act: async (page) => {
      await openWorker(page);
      await page
        .getByRole("button", { name: /add to record/i })
        .first()
        .click();
      await page.waitForTimeout(600);
    },
  },
  {
    name: "roster",
    colorScheme: "light",
    act: async (page) => {
      await page.goto(`${APP}/hr/workers`, { waitUntil: "domcontentloaded" });
      await page.getByText("Maria Alvarez").first().waitFor({ timeout: 20_000 });
      await page.waitForTimeout(SETTLE_MS);
    },
  },
  {
    name: "roster-stepper",
    colorScheme: "light",
    act: async (page) => {
      await page.goto(`${APP}/hr/workers`, { waitUntil: "domcontentloaded" });
      await page.getByText("Maria Alvarez").first().waitFor({ timeout: 20_000 });
      await page.waitForTimeout(SETTLE_MS);
      await page.getByText("Maria Alvarez").first().click();
      await page.waitForURL(/\/hr\/workers\/wrk_/, { timeout: 15_000 });
      await page.waitForTimeout(SETTLE_MS);
    },
  },
  {
    name: "roster-stepper-next",
    colorScheme: "dark",
    act: async (page) => {
      await page.goto(`${APP}/hr/workers`, { waitUntil: "domcontentloaded" });
      await page.getByText("Maria Alvarez").first().waitFor({ timeout: 20_000 });
      await page.waitForTimeout(SETTLE_MS);
      await page.getByText("Maria Alvarez").first().click();
      await page.waitForURL(/\/hr\/workers\/wrk_/, { timeout: 15_000 });
      await page.waitForTimeout(SETTLE_MS);
      await page.getByRole("button", { name: "Next worker" }).first().click();
      await page.waitForTimeout(SETTLE_MS);
    },
  },
  {
    name: "new-worker",
    colorScheme: "light",
    act: async (page) => {
      await page.goto(`${APP}/hr/workers/new`, { waitUntil: "domcontentloaded" });
      await page.getByText("New worker").first().waitFor({ timeout: 20_000 });
      await page.waitForTimeout(SETTLE_MS);
    },
  },

  // The variants.
  {
    name: "not-cleared-overview",
    colorScheme: "light",
    act: (page) => openWorker(page, {}, NOT_CLEARED_ID),
  },
  {
    name: "not-cleared-overview-dark",
    colorScheme: "dark",
    act: (page) => openWorker(page, {}, NOT_CLEARED_ID),
  },
  {
    name: "not-cleared-credentials",
    colorScheme: "light",
    act: (page) => openWorker(page, { tab: "qualification", view: "credentials" }, NOT_CLEARED_ID),
  },
  {
    name: "new-hire-overview",
    colorScheme: "light",
    act: (page) => openWorker(page, {}, NEW_HIRE_ID),
  },
  ...[
    ["employment", "history"],
    ["employment", "checklists"],
    ["qualification", "credentials"],
    ["qualification", "dq-file"],
    ["qualification", "drug-and-alcohol"],
    ["development", "training"],
    ["development", "reviews"],
    ["safety"],
    ["time", "hours-of-service"],
    ["time", "schedule"],
    ["time", "time-off"],
    ["time", "leave"],
    ["pay", "pay-profile"],
    ["pay", "benefits"],
    ["files", "documents"],
    ["files", "driver-portal"],
    ["files", "agent-memory"],
    ["activity"],
  ].map((area) => ({
    name: `new-hire-${areaName(area)}`,
    colorScheme: "light",
    act: (page) => openWorker(page, areaParams(area), NEW_HIRE_ID),
  })),
  {
    name: "dispatcher-overview",
    colorScheme: "light",
    scenario: { workerAccess: "dispatcher" },
    act: (page) => openWorker(page),
  },
  {
    name: "dispatcher-profile",
    colorScheme: "light",
    scenario: { workerAccess: "dispatcher" },
    act: (page) => openWorker(page, { tab: "profile" }),
  },
  {
    name: "dispatcher-qualification",
    colorScheme: "light",
    scenario: { workerAccess: "dispatcher" },
    act: (page) => openWorker(page, { tab: "qualification" }),
  },
  {
    name: "core-plan-overview",
    colorScheme: "light",
    scenario: { workerPlan: "core" },
    act: (page) => openWorker(page),
  },
  {
    name: "core-plan-time",
    colorScheme: "light",
    scenario: { workerPlan: "core" },
    act: (page) => openWorker(page, { tab: "time" }),
  },
  {
    name: "hos-not-connected-overview",
    colorScheme: "light",
    scenario: { workerHos: "notConnected" },
    act: (page) => openWorker(page),
  },
  {
    name: "hos-not-connected-hours",
    colorScheme: "light",
    scenario: { workerHos: "notConnected" },
    act: (page) => openWorker(page, { tab: "time", view: "hours-of-service" }),
  },
  {
    name: "hos-not-linked-hours",
    colorScheme: "light",
    act: (page) => openWorker(page, { tab: "time", view: "hours-of-service" }, NEW_HIRE_ID),
  },

  // Narrower windows.
  {
    name: "overview-1280",
    colorScheme: "light",
    viewport: { width: 1280, height: 900 },
    act: (page) => openWorker(page),
  },
  {
    name: "credentials-1280",
    colorScheme: "light",
    viewport: { width: 1280, height: 900 },
    act: (page) => openWorker(page, { tab: "qualification", view: "credentials" }),
  },
  {
    name: "overview-400",
    colorScheme: "light",
    viewport: { width: 400, height: 860 },
    act: (page) => openWorker(page),
  },
  {
    name: "overview-400-dark",
    colorScheme: "dark",
    viewport: { width: 400, height: 860 },
    act: (page) => openWorker(page),
  },
  {
    name: "profile-400",
    colorScheme: "light",
    viewport: { width: 400, height: 860 },
    act: (page) => openWorker(page, { tab: "profile" }),
  },
  {
    name: "hours-400",
    colorScheme: "light",
    viewport: { width: 400, height: 860 },
    act: (page) => openWorker(page, { tab: "time", view: "hours-of-service" }),
  },
];

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium-1194/chrome-linux/chrome",
});
const failures = [];

for (const shot of SHOTS) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  // A fresh scenario per shot: writes from one shot never leak into the next.
  await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      board: "quiet",
      workerAccess: "full",
      workerPlan: "full",
      workerHos: "reporting",
      ...shot.scenario,
    }),
  });
  const page = await browser.newPage({
    viewport: shot.viewport ?? WIDE,
    colorScheme: shot.colorScheme ?? "light",
    deviceScaleFactor: SCALE,
    timezoneId: "UTC",
  });
  await page.clock.install({ time: new Date(WORKER_NOW * 1000) });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().includes("Failed to load resource")) {
      errors.push(message.text());
    }
  });
  try {
    await shot.act(page);
  } catch (error) {
    errors.push(`interaction failed: ${error.message.split("\n")[0]}`);
  }
  const file = `${OUT}/${shot.name}.png`;
  await page.screenshot({ path: file, fullPage: true });
  console.log(`${errors.length ? "!" : "✓"} ${shot.name} → ${file}`);
  for (const error of errors) console.log(`    ${error.slice(0, 300)}`);
  if (errors.length) failures.push(shot.name);
  await page.close();
}

await browser.close();
if (failures.length) {
  console.log(`\n${failures.length} shot(s) had errors: ${failures.join(", ")}`);
  process.exitCode = 1;
}
