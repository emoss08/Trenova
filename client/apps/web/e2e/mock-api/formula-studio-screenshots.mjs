// Screenshots of the Formula Studio (design_handoff_formula_studio) against the mock API.
//
//   node e2e/mock-api/server.mjs
//   pnpm --filter @trenova/web dev
//   node e2e/mock-api/formula-studio-screenshots.mjs [outDir] [name,name,...]
//
// Every shot starts from a fresh scenario, so a save, a scenario or a submit
// one makes never leaks into the next.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/formula-studio");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 5000);
const STEP_MS = Number(process.env.STEP_MS ?? 1800);

const BASE = "/billing/configuration-files/formula-templates";
const DRAFT = `${BASE}/fmt_studio_draft/edit`;
const REVIEW = `${BASE}/fmt_studio_review/edit`;
const ACTIVE = `${BASE}/fmt_studio_active/edit`;
const NEW = `${BASE}/new`;

// A pane's module is read the first time its tab opens, so the click waits for it.
const tab = (name) => async (page) => {
  await page.getByRole("tab", { name }).click();
  await page.waitForTimeout(1500);
};
const pause = (page, ms = STEP_MS) => page.waitForTimeout(ms);

async function typeInEditor(page, text) {
  const editor = page.locator(".cm-content").first();
  await editor.click();
  await page.keyboard.press("Control+End");
  await page.keyboard.type(text);
}

function shotsFor(colorScheme) {
  return [
    { name: "edit-draft", path: DRAFT },
    { name: "edit-review", path: REVIEW },
    { name: "edit-active", path: ACTIVE },
    {
      name: "active-material-edit",
      path: ACTIVE,
      act: async (page) => {
        await typeInEditor(page, " + 10");
        await pause(page);
      },
    },
    {
      name: "active-in-use",
      path: ACTIVE,
      act: async (page) => {
        await page.getByRole("button", { name: /in use$/ }).hover();
        await pause(page, 900);
      },
    },
    { name: "create-empty", path: NEW },
    {
      name: "create-started",
      path: NEW,
      act: async (page) => {
        await page
          .getByRole("button", { name: /^Per Mile/ })
          .first()
          .click();
        await pause(page);
      },
    },
    {
      name: "more-menu",
      path: DRAFT,
      act: (page) => page.getByRole("button", { name: "More actions" }).click(),
    },
    {
      name: "review-more-menu",
      path: REVIEW,
      act: (page) => page.getByRole("button", { name: "More actions" }).click(),
    },
    {
      name: "explain",
      path: DRAFT,
      act: async (page) => {
        await page.getByRole("button", { name: "Explain", exact: true }).click();
        await pause(page);
        await page
          .getByText(/^let fuel/)
          .first()
          .hover();
        await pause(page, 500);
      },
    },
    {
      name: "problems",
      path: DRAFT,
      act: async (page) => {
        await typeInEditor(page, " + weekendFee");
        await pause(page);
      },
    },
    {
      name: "problems-many",
      path: DRAFT,
      act: async (page) => {
        await typeInEditor(
          page,
          "\n+ weekendFee * hasHazmat + rond(weight) + customer.nmae + pow(2) + liftgate + (totalStops ? 1 : 0)",
        );
        await pause(page, 1800);
      },
    },
    {
      name: "editor-hover",
      path: DRAFT,
      act: async (page) => {
        await page
          .locator(".cm-content")
          .first()
          .getByText("baseRate", { exact: true })
          .first()
          .hover();
        await pause(page, 900);
      },
    },
    {
      name: "editor-signature",
      path: DRAFT,
      act: async (page) => {
        await typeInEditor(page, "\n+ max(");
        await pause(page, 800);
      },
    },
    {
      name: "inputs",
      path: DRAFT,
      act: async (page) => {
        await page.getByRole("heading", { name: "Inputs" }).scrollIntoViewIfNeeded();
        await pause(page, 900);
      },
    },
    {
      name: "policy-breakdown",
      path: DRAFT,
      act: async (page) => {
        await page.getByRole("heading", { name: "Charge policy" }).scrollIntoViewIfNeeded();
        await page.getByRole("button", { name: "Charge policy" }).first().click();
        await pause(page, 1200);
      },
    },
    {
      name: "preview-shipment",
      path: DRAFT,
      act: async (page) => {
        await page.getByRole("radio", { name: "Real shipment" }).click();
        await pause(page, 600);
      },
    },
    {
      name: "preview-edited",
      path: DRAFT,
      act: async (page) => {
        const value = page.getByRole("textbox", { name: "totalDistance" });
        await value.fill("120");
        await pause(page);
      },
    },
    { name: "scenarios", path: DRAFT, act: tab(/^Scenarios/) },
    {
      name: "scenario-why",
      path: DRAFT,
      act: async (page) => {
        await tab(/^Scenarios/)(page);
        await pause(page);
        await page
          .getByRole("button", { name: /^Why did/ })
          .first()
          .click();
        await pause(page);
      },
    },
    {
      name: "scenario-suggestions",
      path: DRAFT,
      act: async (page) => {
        await tab(/^Scenarios/)(page);
        await page.getByRole("button", { name: "Suggest edge cases" }).click();
        await pause(page);
      },
    },
    {
      name: "reference",
      path: DRAFT,
      act: async (page) => {
        await page.locator("body").click({ position: { x: 5, y: 300 } });
        await page.keyboard.press("/");
        await page.keyboard.type("stop");
        await pause(page, 600);
      },
    },
    {
      name: "editor-formatted",
      path: NEW,
      act: async (page) => {
        await typeInEditor(
          page,
          "let linehaul = mileRate*totalDistance; let weightCharge = max(0, totalWeight - weightFreeLbs) / 100 * weightRatePerCwt; let stopCharge = max(0, totalStops - stopsIncluded) * stopRate; let hazmatCharge = hasHazmat?hazmatFee:0; linehaul+weightCharge+stopCharge+hazmatCharge",
        );
        await pause(page, 600);
        await page.keyboard.press("Shift+Alt+F");
        await pause(page, 1200);
      },
    },
    {
      name: "reference-functions",
      path: DRAFT,
      act: async (page) => {
        await tab(/^Reference/)(page);
        await pause(page, 600);
        await page.getByRole("radio", { name: /^Functions/ }).click();
        await pause(page, 600);
      },
    },
    {
      name: "preview-all-values",
      path: DRAFT,
      act: async (page) => {
        await page.getByRole("button", { name: /^Show all/ }).click();
        await pause(page, 900);
      },
    },
    {
      name: "submit-after-signature",
      path: DRAFT,
      act: async (page) => {
        await typeInEditor(page, "\n+ max(");
        await pause(page, 800);
        await page.keyboard.press("Control+Z");
        await page.keyboard.press("Control+Z");
        await pause(page, 600);
        await page.getByRole("button", { name: "Submit for review" }).click();
        await pause(page, 2500);
      },
    },
    { name: "nova-empty", path: DRAFT, act: tab(/^Nova/) },
    {
      name: "nova-let",
      path: DRAFT,
      act: async (page) => {
        await tab(/^Nova/)(page);
        await pause(page, 900);
        await page.getByRole("textbox", { name: /^Message/ }).fill("Add a liftgate fee");
        await page.keyboard.press("Enter");
        await pause(page, 7000);
      },
    },
    {
      name: "nova-return",
      path: DRAFT,
      act: async (page) => {
        await tab(/^Nova/)(page);
        await pause(page, 900);
        await page
          .getByRole("textbox", { name: /^Message/ })
          .fill("Explain this formula, term by term");
        await page.keyboard.press("Enter");
        await pause(page, 6000);
        await tab(/^Preview/)(page);
        await pause(page, 800);
        await tab(/^Nova/)(page);
        await pause(page, 800);
      },
    },
    {
      name: "nova-proposal",
      path: DRAFT,
      act: async (page) => {
        await tab(/^Nova/)(page);
        await pause(page, 900);
        await page.getByRole("textbox", { name: /^Message/ }).fill("Add a weekend pickup fee");
        await page.keyboard.press("Enter");
        await pause(page, 7000);
      },
    },
    {
      name: "nova-policy",
      path: DRAFT,
      act: async (page) => {
        await tab(/^Nova/)(page);
        await pause(page, 900);
        await page.getByRole("textbox", { name: /^Message/ }).fill("Set a $300 minimum charge");
        await page.keyboard.press("Enter");
        await pause(page, 7000);
      },
    },
    {
      name: "trace-folded",
      path: DRAFT,
      act: (page) => page.getByRole("button", { name: /Hide sidebar/ }).click(),
    },
    {
      name: "submit-dialog",
      path: `${BASE}/fmt_studio_active/edit`,
      scenario: {},
      act: async (page) => {
        await page.goto(`${APP}${DRAFT}`);
        await pause(page, SETTLE_MS);
        await page.getByRole("button", { name: "Submit for review" }).click();
        await pause(page, 2500);
      },
    },
    {
      name: "backtest-dialog",
      path: DRAFT,
      act: async (page) => {
        await page.getByRole("button", { name: "More actions" }).click();
        await page.getByRole("menuitem", { name: /Backtest/ }).click();
        await pause(page, 2500);
      },
    },
    {
      name: "save-demote",
      path: ACTIVE,
      act: async (page) => {
        await typeInEditor(page, " + 10");
        await pause(page, 900);
        await page.getByRole("button", { name: /Save changes/ }).click();
        await pause(page, 900);
      },
    },
    { name: "narrow-1100", path: DRAFT, viewport: { width: 1100, height: 860 } },
    { name: "stacked-860", path: DRAFT, viewport: { width: 860, height: 900 } },
  ].map((shot) => ({ ...shot, name: `${shot.name}-${colorScheme}`, colorScheme }));
}

async function setScenario(patch) {
  const response = await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(patch ?? {}),
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
  await setScenario(shot.scenario ?? {});
  const context = await browser.newContext({
    viewport: shot.viewport ?? { width: 1600, height: 1000 },
    deviceScaleFactor: 2,
    colorScheme: shot.colorScheme,
  });
  await context.addInitScript((theme) => {
    window.localStorage.setItem("vite-ui-theme", theme);
  }, shot.colorScheme);
  const page = await context.newPage();
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
