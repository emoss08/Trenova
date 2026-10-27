// Screenshots of the Formula Studio and the formula libraries on Tariff, against the mock API.
//
//   node e2e/mock-api/server.mjs
//   pnpm --filter @trenova/web dev
//   node e2e/mock-api/formula-studio-screenshots.mjs [outDir] [name,name,...]
//
// Every shot starts from a fresh scenario, so a save, a test or a submit one makes never leaks
// into the next.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";
import { cursorAt, pointAt } from "./editor-points.mjs";

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
const LTL = `${BASE}/fmt_studio_ltl/edit`;
const SHIPMENT_WITH_CHOICES =
  "/shipment-management/shipments?panelType=edit&panelEntityId=shp_0412";
const NEW = `${BASE}/new`;
const LIBRARIES = "/billing/configuration-files/formula-libraries";
const ACCESSORIALS = `${LIBRARIES}/fl_studio_accessorials/edit`;
const FUEL = `${LIBRARIES}/fl_studio_fuel/edit`;
const ZONES = `${LIBRARIES}/fl_studio_zones/edit`;
const NEW_LIBRARY = `${LIBRARIES}/new`;

// A pane's module is read the first time its tab opens, so the click waits for it.
const tab = (name) => async (page) => {
  await page.getByRole("tab", { name }).click();
  await page.waitForTimeout(1500);
};
const pause = (page, ms = STEP_MS) => page.waitForTimeout(ms);

/** Scrolls the Studio's form to a section and lets it settle. */
async function scrollToSection(page, section) {
  await page.evaluate((id) => {
    document.getElementById(`studio-section-${id}`)?.scrollIntoView({ block: "start" });
  }, section);
  await pause(page, 900);
}

/** Hovers the coverage bar beside the first line no test reaches. */
async function hoverCoverageGap(page) {
  const gap = page.locator(".cm-tariff-coverage-gap").first();
  await gap.scrollIntoViewIfNeeded();
  await gap.hover();
  await pause(page, 600);
}

/** Picks the real shipment the preview rates. */
async function pickShipment(page, pro = "S-240741") {
  await page.getByRole("radio", { name: "Real shipment" }).click();
  await pause(page, 600);
  await page
    .getByText(/^Search by Pro/)
    .first()
    .click();
  await pause(page, 900);
  await page.getByText(pro, { exact: true }).first().click();
  await pause(page);
}

const ask = (question) => async (page) => {
  await tab(/^Nova/)(page);
  await pause(page, 900);
  await page.getByRole("textbox", { name: /^Message/ }).fill(question);
  await page.keyboard.press("Enter");
  await pause(page, 7000);
};

function shotsFor(colorScheme) {
  return [
    { name: "list", path: BASE },
    { name: "edit-draft", path: DRAFT },
    { name: "edit-review", path: REVIEW },
    { name: "edit-active", path: ACTIVE },
    {
      name: "active-material-edit",
      path: ACTIVE,
      act: async (page) => {
        await cursorAt(page, "$350\n");
        await page.keyboard.press("End");
        await page.keyboard.type("0");
        await pause(page);
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
          .getByText(/comes to/)
          .first()
          .scrollIntoViewIfNeeded();
        await pause(page, 600);
      },
    },
    {
      name: "problems",
      path: DRAFT,
      act: async (page) => {
        await cursorAt(page, 'charge "Fuel surcharge"');
        await page.keyboard.type("let weekendFee = waekendRate * 2\n");
        await pause(page);
        await page
          .getByText(/^Nothing is named/)
          .first()
          .scrollIntoViewIfNeeded();
        await pause(page, 600);
      },
    },
    {
      name: "editor-complete",
      path: DRAFT,
      act: async (page) => {
        await cursorAt(page, "let linehaul");
        await page.keyboard.press("Enter");
        await page.keyboard.press("ArrowUp");
        await page.keyboard.type("let w = shipment.");
        await pause(page, 900);
      },
    },
    {
      name: "editor-hover",
      path: DRAFT,
      act: async (page) => {
        const point = await pointAt(page, "= extraStops", 4);
        await page.mouse.move(point.x, point.y);
        await pause(page, 900);
      },
    },
    {
      name: "editor-signature",
      path: DRAFT,
      act: async (page) => {
        await cursorAt(page, "extraStops(", "extraStops(".length);
        await pause(page, 800);
      },
    },
    {
      name: "editor-rename",
      path: DRAFT,
      act: async (page) => {
        await cursorAt(page, "let linehaul", 6);
        await page.keyboard.press("F2");
        await pause(page, 600);
      },
    },
    {
      name: "editor-definition",
      path: DRAFT,
      act: async (page) => {
        await cursorAt(page, "= extraStops", 4);
        await page.keyboard.press("F12");
        await pause(page, 1200);
      },
    },
    {
      name: "inputs",
      path: DRAFT,
      act: async (page) => {
        await page.getByRole("heading", { name: "Inputs" }).last().scrollIntoViewIfNeeded();
        await pause(page, 900);
      },
    },
    {
      name: "preview-shipment",
      path: DRAFT,
      act: (page) => pickShipment(page),
    },
    {
      name: "save-as-test",
      path: DRAFT,
      act: async (page) => {
        await pickShipment(page);
        await page.getByRole("button", { name: "Save as test" }).click();
        await pause(page, 600);
      },
    },
    { name: "tests", path: DRAFT, act: tab(/^Tests/) },
    {
      name: "tests-fixed",
      path: DRAFT,
      act: async (page) => {
        await cursorAt(page, "expect total == $2_410.76");
        await page.keyboard.press("End");
        for (let i = 0; i < "2_410.76".length; i += 1) await page.keyboard.press("Backspace");
        await page.keyboard.type("2_495.35");
        await tab(/^Tests/)(page);
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
        await page.locator(".cm-content").first().click();
        await page.keyboard.type(
          'tariff "Messy"\n\n\n\ninput rate:Money/mi=$2/mi\ncharge "Linehaul"(quantity:shipment.distance,rate:rate)\n',
        );
        await pause(page, 600);
        await page.keyboard.press("Shift+Alt+F");
        await pause(page, 1200);
      },
    },
    {
      name: "preview-trace",
      path: DRAFT,
      act: async (page) => {
        await pickShipment(page);
        await page.getByRole("button", { name: /^Trace/ }).click();
        await pause(page, 900);
      },
    },
    { name: "nova-empty", path: DRAFT, act: tab(/^Nova/) },
    // The server offers a change only when it checks and every test passes, so the changes are
    // asked of the active template; the draft, whose fuel test fails, shows the refusal.
    { name: "nova-proposal", path: ACTIVE, act: ask("Add a weekend pickup fee of $125") },
    { name: "nova-minimum", path: ACTIVE, act: ask("Set the minimum to $400") },
    { name: "nova-refused", path: DRAFT, act: ask("Add a weekend pickup fee of $125") },
    {
      name: "suggested-tests",
      path: DRAFT,
      act: async (page) => {
        await page.getByRole("heading", { name: "Tests" }).last().scrollIntoViewIfNeeded();
        await page.getByRole("button", { name: "Suggest tests" }).click();
        await pause(page, 3000);
        await page.getByText("Suggested tests").scrollIntoViewIfNeeded();
      },
    },
    {
      name: "explain-test",
      path: DRAFT,
      act: async (page) => {
        await tab(/^Tests/)(page);
        await page.getByRole("button", { name: "Why does this fail?" }).first().click();
        await pause(page, 2500);
      },
    },
    {
      name: "nova-tests",
      path: ACTIVE,
      act: ask("Suggest tests for the cases this tariff could get wrong"),
    },
    { name: "nova-explain", path: DRAFT, act: ask("Explain this tariff, statement by statement") },
    {
      name: "submit-refused",
      path: DRAFT,
      act: async (page) => {
        await page.getByRole("button", { name: "Submit for review" }).click();
        await pause(page, 2500);
      },
    },
    {
      name: "approve-dialog",
      path: REVIEW,
      act: async (page) => {
        await page.getByRole("button", { name: "Approve" }).click();
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
        await cursorAt(page, "$350\n");
        await page.keyboard.press("End");
        await page.keyboard.type("0");
        await pause(page, 900);
        await page.getByRole("button", { name: /Save changes/ }).click();
        await pause(page, 900);
      },
    },
    { name: "narrow-1100", path: DRAFT, viewport: { width: 1100, height: 860 } },
    { name: "stacked-860", path: DRAFT, viewport: { width: 860, height: 900 } },

    // Properties, coverage, references, Promote to input, choices and input bounds, on the LTL
    // tariff: its weight break inverts at 1,000 lb, a hazmat fee no test reaches and a `best` of
    // three linehaul methods.
    { name: "ltl-edit", path: LTL },
    { name: "ltl-tests", path: LTL, act: tab(/^Tests/) },
    {
      name: "ltl-coverage-unreached",
      path: LTL,
      act: async (page) => {
        await tab(/^Tests/)(page);
        await page.getByRole("button", { name: /^Show \d+ unreached region/ }).click();
        await pause(page, 600);
      },
    },
    { name: "ltl-coverage-tooltip", path: LTL, act: hoverCoverageGap },
    {
      name: "ltl-save-as-test",
      path: LTL,
      act: async (page) => {
        await tab(/^Tests/)(page);
        await page.getByRole("button", { name: "Save as test" }).click();
        await pause(page, 2500);
      },
    },
    {
      name: "ltl-occurrences",
      path: LTL,
      act: async (page) => {
        await cursorAt(page, "if weight >= 1_000 lb", 4);
        await pause(page, 1200);
      },
    },
    {
      name: "ltl-find-references",
      path: LTL,
      act: async (page) => {
        await cursorAt(page, "if weight >= 1_000 lb", 4);
        await page.keyboard.press("Shift+F12");
        await pause(page, 1200);
      },
    },
    {
      name: "ltl-editor-menu",
      path: LTL,
      act: async (page) => {
        await cursorAt(page, "$425", 1);
        await page.getByRole("button", { name: "Editor actions" }).click();
        await pause(page, 600);
      },
    },
    {
      name: "ltl-promote-menu",
      path: LTL,
      act: async (page) => {
        await cursorAt(page, "$425", 1);
        await page.keyboard.press("Control+.");
        await pause(page, 600);
      },
    },
    {
      name: "ltl-promote-rename",
      path: LTL,
      act: async (page) => {
        await cursorAt(page, "$425", 1);
        await page.keyboard.press("Control+.");
        await pause(page, 600);
        await page.keyboard.press("Enter");
        await pause(page, 1200);
      },
    },
    {
      name: "ltl-promote-nothing",
      path: LTL,
      act: async (page) => {
        await cursorAt(page, 'best "Linehaul"', 1);
        await page.keyboard.press("Control+.");
        await pause(page, 500);
      },
    },
    {
      name: "ltl-hover-bounds",
      path: LTL,
      act: async (page) => {
        const point = await pointAt(page, "weight * rate", 10);
        await page.mouse.move(point.x, point.y);
        await pause(page, 1200);
      },
    },
    { name: "ltl-preview-choices", path: LTL },
    {
      name: "ltl-preview-choices-shipment",
      path: LTL,
      act: (page) => pickShipment(page, "S-240811"),
    },
    {
      name: "ltl-trace-choice",
      path: LTL,
      act: async (page) => {
        await page.getByRole("button", { name: /^Trace/ }).click();
        await pause(page, 600);
      },
    },
    { name: "ltl-inputs-bounds", path: LTL, act: (page) => scrollToSection(page, "inputs") },
    {
      name: "ltl-inputs-out-of-bounds",
      path: LTL,
      act: async (page) => {
        await scrollToSection(page, "inputs");
        await page.getByPlaceholder("Default: $0.31/lb").fill("3");
        await pause(page, 1200);
      },
    },
    {
      name: "ltl-inputs-service-select",
      path: LTL,
      act: async (page) => {
        await scrollToSection(page, "inputs");
        await page.getByRole("button", { name: /Default: Standard/ }).click();
        await pause(page, 600);
      },
    },
    {
      name: "ltl-submit-readiness",
      path: LTL,
      act: async (page) => {
        await page.getByRole("button", { name: "Submit for review" }).click();
        await pause(page, 2500);
      },
    },
    {
      name: "shipment-rating-choices",
      path: SHIPMENT_WITH_CHOICES,
      settle: 8000,
      act: async (page) => {
        await page.getByRole("region", { name: "Choices" }).scrollIntoViewIfNeeded();
        await pause(page, 900);
      },
    },

    { name: "libraries-list", path: LIBRARIES },
    { name: "library-active", path: ACCESSORIALS },
    { name: "library-review", path: FUEL },
    { name: "library-failing-test", path: ZONES },
    { name: "library-tests", path: ZONES, act: tab(/^Tests/) },
    { name: "library-usage", path: ACCESSORIALS, act: tab(/^Imported by/) },
    { name: "library-history", path: ACCESSORIALS, act: tab(/^History/) },
    {
      name: "library-compare",
      path: ACCESSORIALS,
      act: async (page) => {
        await tab(/^History/)(page);
        await page.getByRole("button", { name: "Compare v3" }).click();
        await pause(page, 900);
      },
    },
    {
      name: "library-submit",
      path: ZONES,
      act: async (page) => {
        await page.getByRole("button", { name: "Submit for review" }).click();
        await pause(page, 2500);
      },
    },
    {
      name: "library-approve",
      path: FUEL,
      act: async (page) => {
        await page.getByRole("button", { name: "Approve" }).click();
        await pause(page, 2500);
      },
    },
    { name: "library-new", path: NEW_LIBRARY },
    {
      name: "library-new-started",
      path: NEW_LIBRARY,
      act: async (page) => {
        await page.getByRole("textbox", { name: "Library name" }).fill("fees");
        await page.getByRole("button", { name: "Use the starter" }).click();
        await pause(page);
      },
    },
    { name: "library-nova", path: ACCESSORIALS, act: ask("Suggest tests for the edge cases") },
    { name: "library-nova-refused", path: ZONES, act: ask("Suggest tests for the edge cases") },
    { name: "library-reference", path: ACCESSORIALS, act: tab(/^Reference/) },
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
