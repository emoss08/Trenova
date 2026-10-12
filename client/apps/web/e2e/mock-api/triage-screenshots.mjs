// Screenshots of Triage (/triage) against the mock API: every state the prototype's Tweaks panel
// offers, on desktop and phone, dark and light, and the main interactions. See README.md.
//
// The walkthrough finds things the way a person does — by role and by the prototype's words
// ("File all", "Turn on", "Hand off to Marcus") — so it keeps working as the page's markup moves.
// Each shot is small and stands alone: a step that cannot be done is reported under its shot and
// the rest carry on.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const PAGE = `${APP}/triage`;
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/triage");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 6000);
const STEP_MS = Number(process.env.STEP_MS ?? 4000);

const DEFAULT_SCENARIO = { triage: "live", triageTime: "afternoon", triageReplyAfterMs: 6500 };
const DESKTOP = { width: 1440, height: 900 };
const PHONE = { width: 390, height: 844 };

// --- steps

/** Clicks the first of several ways to name the same control that is on the page. */
async function clickAny(page, locators, options = {}) {
  for (const make of locators) {
    const locator = make(page).first();
    if (await locator.isVisible({ timeout: 800 }).catch(() => false)) {
      await locator.click({ timeout: STEP_MS, ...options });
      await page.waitForTimeout(500);
      return;
    }
  }
  throw new Error(`none of ${locators.length} candidates is on the page`);
}

const button = (name, exact = false) => (page) => page.getByRole("button", { name, exact });
const titled = (title) => (page) => page.getByTitle(title);

/** The first message row in the open folder, found by its sender. */
async function rightClickRow(page, sender) {
  await page.getByText(sender, { exact: true }).first().click({ button: "right", timeout: STEP_MS });
  await page.waitForTimeout(500);
}

/** The reader's next-step button, whichever words it has for this step. */
async function doNextStep(page) {
  await clickAny(page, [button(/^Have Nova do it/), button(/^Do it/), button(/^Review (reply|draft)/)]);
}

async function graphql(operationName, variables) {
  const response = await fetch(`${MOCK}/graphql`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ operationName, variables }),
  });
  const body = await response.json();
  if (body.errors?.length) throw new Error(`${operationName}: ${body.errors[0].message}`);
  return body.data;
}

// --- shots

const views = (name, base) => [
  { ...base, name: `${name}-desktop-dark` },
  { ...base, name: `${name}-desktop-light`, colorScheme: "light" },
  { ...base, name: `${name}-phone-dark`, viewport: PHONE },
  { ...base, name: `${name}-phone-light`, viewport: PHONE, colorScheme: "light" },
];

const SHOTS = [
  // The Tweaks panel's states: Inbox live / first day / no mailbox / desk off, afternoon / end of day.
  ...views("briefing-live", { params: { brief: "" } }),
  ...views("briefing-end-of-day", { scenario: { triageTime: "endOfDay" }, params: { brief: "" } }),
  ...views("briefing-first-day", { scenario: { triage: "firstDay" }, params: { brief: "" } }),
  ...views("briefing-no-mailbox", { scenario: { triage: "noMailbox" }, params: { brief: "" } }),
  ...views("briefing-desk-off", { scenario: { triage: "deskOff" }, params: { brief: "" } }),
  ...views("waiting-desk-off", { scenario: { triage: "deskOff" }, params: { lane: "waiting" } }),
  ...views("waiting", { params: { lane: "waiting" } }),
  ...views("waiting-end-of-day", { scenario: { triageTime: "endOfDay" }, params: { lane: "waiting" } }),

  // The main interactions.
  { name: "folder-first-message", params: { lane: "waiting" } },
  { name: "folder-newest", params: { lane: "all", sort: "newest" } },
  { name: "row-context-menu", params: { lane: "waiting" }, act: (page) => rightClickRow(page, "Partner Carrier EDI") },
  {
    name: "row-context-snooze",
    params: { lane: "waiting" },
    act: async (page) => {
      await rightClickRow(page, "Crestline Retail");
      await page.getByRole("menuitem", { name: /^Snooze/ }).first().hover({ timeout: STEP_MS });
      await page.waitForTimeout(500);
    },
  },
  {
    name: "sender-card",
    params: { lane: "waiting", message: "imsg_m5" },
    act: (page) => clickAny(page, [button("Blue Ridge Brokerage", true), (p) => p.getByRole("main").getByText("Blue Ridge Brokerage", { exact: true })]),
  },
  {
    name: "kind-correction-menu",
    params: { lane: "waiting", message: "imsg_m3" },
    act: (page) => clickAny(page, [titled("Read wrong? Correct it"), button(/^Status request/), button(/Correct the kind/)]),
  },
  {
    name: "reply-composer-templates",
    params: { lane: "waiting", message: "imsg_m7" },
    act: async (page) => {
      await doNextStep(page);
      await page.waitForTimeout(2200);
      await clickAny(page, [button("Templates", true), button(/Templates/)]);
    },
  },
  {
    name: "reply-refused-550",
    params: { lane: "waiting", message: "imsg_m3" },
    act: async (page) => {
      await page.keyboard.press("r");
      await page.waitForTimeout(800);
      await page.getByRole("textbox").last().fill("Hi — which load is status file 0441 about? Your 214 was cut off.\n\nThanks,\nTrenova");
      await clickAny(page, [button(/^Send$/), button(/^Send/)]);
      await page.waitForTimeout(1500);
      await page.getByText(/Didn't send|550/).first().waitFor({ timeout: STEP_MS });
    },
  },
  {
    name: "reply-resent-to-alternate",
    params: { lane: "waiting", message: "imsg_m3" },
    act: async (page) => {
      await page.keyboard.press("r");
      await page.waitForTimeout(800);
      await page.getByRole("textbox").last().fill("Hi — which load is status file 0441 about?\n\nThanks,\nTrenova");
      await clickAny(page, [button(/^Send$/), button(/^Send/)]);
      await page.waitForTimeout(1500);
      await clickAny(page, [button(/^Send to ops@/), button(/instead/)]);
      await page.waitForTimeout(1200);
    },
  },
  {
    name: "load-form",
    params: { lane: "waiting", message: "imsg_m1" },
    act: async (page) => {
      await doNextStep(page);
      await page.waitForTimeout(2200);
    },
  },
  {
    name: "file-row-split-pages",
    params: { lane: "waiting", message: "imsg_m5" },
    act: async (page) => {
      await doNextStep(page);
      await clickAny(page, [button(/^Split into 2 documents/), button(/^Split/)]);
    },
  },
  {
    name: "stops-check",
    params: { lane: "waiting", message: "imsg_m2" },
    act: (page) => doNextStep(page),
  },
  {
    name: "stops-waived-reply",
    params: { lane: "waiting", message: "imsg_m2" },
    act: async (page) => {
      await doNextStep(page);
      await clickAny(page, [button(/^Waive \$185/), button(/^Waive/)]);
      await page.waitForTimeout(1200);
    },
  },
  { name: "conflict-banner", params: { lane: "waiting", message: "imsg_m5" } },
  { name: "conflict-overlapping-load", params: { lane: "waiting", message: "imsg_m8" } },
  { name: "rules", params: { rules: "" } },
  {
    name: "rules-add-row-preview",
    params: { rules: "" },
    act: async (page) => {
      await clickAny(page, [button(/New rule/), button(/arrives, Nova should/)]);
      const blanks = page.getByRole("combobox");
      await blanks.nth(0).selectOption({ label: "rate confirmation" }).catch(() => blanks.nth(0).click());
      await blanks.nth(1).selectOption({ label: "blueridgebrokerage.example" }).catch(() => {});
      await blanks.nth(3).selectOption({ index: 0 }).catch(() => {});
      await page.waitForTimeout(1200);
      await page.getByText(/would have matched/).first().waitFor({ timeout: STEP_MS });
    },
  },
  {
    name: "rule-matches-dialog",
    params: { rules: "" },
    act: async (page) => {
      await clickAny(page, [titled("See what it matched"), button(/See what it matched/), button(/See them/)]);
      await page.getByRole("dialog").first().waitFor({ timeout: STEP_MS });
    },
  },
  {
    name: "doc-viewer",
    params: { lane: "waiting", message: "imsg_m5" },
    act: async (page) => {
      await doNextStep(page);
      await clickAny(page, [button(/^Preview$/), button(/Preview/)]);
      await page.getByText("What Nova read").first().waitFor({ timeout: STEP_MS });
    },
  },
  {
    name: "shipment-peek",
    params: { lane: "waiting", message: "imsg_m7" },
    act: async (page) => {
      await clickAny(page, [button(/^S-104198/)]);
      await clickAny(page, [button(/^Open shipment/), button(/Peek/)]).catch(() => {});
      await page.getByText("On file").first().waitFor({ timeout: STEP_MS });
    },
  },
  {
    name: "customer-peek",
    params: { lane: "waiting", message: "imsg_m7" },
    act: async (page) => {
      await clickAny(page, [button(/^Crestline Retail\b.*Customer/), button(/^Crestline Retail/)]);
      await page.getByText(/shipments this week/).first().waitFor({ timeout: STEP_MS });
    },
  },
  {
    name: "desk-hand-off",
    params: { lane: "waiting", message: "imsg_m2" },
    act: async (page) => {
      await clickAny(page, [button(/Work on it in Desk/), button(/Ask an agent in Desk/)]);
      await page.waitForTimeout(2200);
    },
  },
  {
    name: "connect-a-source",
    scenario: { triage: "noMailbox" },
    params: { brief: "" },
    act: async (page) => {
      await clickAny(page, [button(/^Add a source/), button(/^Connect/)]);
      await page.getByRole("dialog").first().waitFor({ timeout: STEP_MS });
      await clickAny(page, [button(/^Continue/)]);
    },
  },
  {
    name: "first-day-reading",
    scenario: { triage: "firstDay" },
    params: { brief: "" },
    act: async (page) => {
      await graphql("CreateInboundMailbox", { input: { name: "Dispatch", address: "dispatch@trenovalogistics.com", provider: "Postmark", purpose: "", reviewPolicy: "ReviewBelowConfidence", minConfidence: 0.75, status: "Active" } });
      await page.waitForTimeout(1500);
    },
  },
  {
    name: "shortcuts-sheet",
    params: { lane: "waiting" },
    act: async (page) => {
      await page.keyboard.press("?");
      await page.getByText("Keyboard shortcuts").first().waitFor({ timeout: STEP_MS });
    },
  },
  {
    name: "bulk-selection-bar",
    params: { lane: "waiting" },
    act: async (page) => {
      await page.keyboard.press("x");
      await page.keyboard.press("j");
      await page.keyboard.press("x");
      await page.keyboard.press("j");
      await page.keyboard.press("x");
      await page.getByText(/3 selected/).first().waitFor({ timeout: STEP_MS });
    },
  },
  {
    name: "snoozed-lane",
    params: { lane: "snoozed" },
    prepare: async ({ triageAnchor }) => {
      await graphql("SnoozeInboundMessages", { selection: { ids: ["imsg_m8", "imsg_m6"] }, input: { until: triageAnchor + 18 * 3600 } });
      await graphql("SnoozeInboundMessages", { selection: { ids: ["imsg_m7"] }, input: { event: "ShipmentDelivered" } });
    },
  },
  {
    name: "waiting-on-them-lane",
    scenario: { triageReplyAfterMs: 600000 },
    params: { lane: "them" },
    prepare: async ({ triageAnchor }) => {
      await graphql("SendInboundReply", { input: { messageId: "imsg_m7", subject: "Re: Where is PO 88213?", body: "Hi — PO 88213 is on S-104198. Current ETA is 16:10.\n\nThanks,\nTrenova" } });
      await graphql("WaitForInboundReply", { id: "imsg_m7", until: triageAnchor + 18 * 3600 });
    },
  },
  {
    name: "end-of-day-carry-over",
    scenario: { triageTime: "endOfDay" },
    params: { brief: "" },
  },
  {
    name: "end-of-day-handed-off",
    scenario: { triageTime: "endOfDay" },
    params: { brief: "" },
    act: async (page) => {
      await clickAny(page, [button(/^Hand off to Marcus/), button(/Hand off/)]);
      await page.waitForTimeout(1500);
    },
  },
  {
    name: "briefing-file-all",
    params: { brief: "" },
    act: async (page) => {
      await clickAny(page, [button(/^File all/)]);
      await page.waitForTimeout(1500);
    },
  },
  {
    name: "briefing-rule-suggestion-preview",
    params: { brief: "" },
    act: async (page) => {
      await clickAny(page, [button(/^Preview$/)]);
      await page.getByRole("dialog").first().waitFor({ timeout: STEP_MS });
    },
  },
  {
    name: "desk-off-turn-on",
    scenario: { triage: "deskOff" },
    params: { lane: "waiting" },
    act: async (page) => {
      await clickAny(page, [button(/^Turn on Nova/), button(/^Turn on/)]);
      await page.waitForTimeout(2600);
    },
  },
  {
    name: "claims-quiet-test-email",
    params: { mailbox: "imb_claims" },
    act: async (page) => {
      await clickAny(page, [button(/^Send a test email/)]);
      await page.waitForTimeout(2200);
    },
  },
  {
    name: "message-phone-actions-sheet",
    viewport: PHONE,
    params: { lane: "waiting", message: "imsg_m6" },
    act: (page) => clickAny(page, [button(/Actions$/), button(/Actions/)]),
  },
];

async function setScenario(overrides) {
  const response = await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ ...DEFAULT_SCENARIO, ...overrides }),
  });
  return response.json();
}

function urlOf(params = {}) {
  const query = Object.entries(params)
    .map(([key, value]) => (value === "" ? encodeURIComponent(key) : `${encodeURIComponent(key)}=${encodeURIComponent(value)}`))
    .join("&");
  return query ? `${PAGE}?${query}` : PAGE;
}

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium-1194/chrome-linux/chrome",
});
const failures = [];

for (const shot of SHOTS) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  const errors = [];
  let page = null;
  try {
    const scenario = await setScenario(shot.scenario ?? {});
    if (shot.prepare) await shot.prepare(scenario);
    const viewport = shot.viewport ?? DESKTOP;
    page = await browser.newPage({
      viewport,
      colorScheme: shot.colorScheme ?? "dark",
      timezoneId: "UTC",
      isMobile: viewport === PHONE,
      hasTouch: viewport === PHONE,
    });
    await page.clock.install({ time: new Date(scenario.triageAnchor * 1000) });
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("console", (message) => {
      if (message.type() === "error" && !message.text().includes("Failed to load resource")) errors.push(message.text());
    });
    await page.goto(urlOf(shot.params), { waitUntil: "domcontentloaded" });
    await page.waitForTimeout(SETTLE_MS);
    if (shot.act) {
      try {
        await shot.act(page);
        await page.waitForTimeout(900);
      } catch (error) {
        errors.push(`interaction failed: ${error.message.split("\n")[0]}`);
      }
    }
  } catch (error) {
    errors.push(`setup failed: ${error.message.split("\n")[0]}`);
  }
  const file = `${OUT}/${shot.name}.png`;
  if (page) {
    await page.screenshot({ path: file }).catch((error) => errors.push(`screenshot failed: ${error.message.split("\n")[0]}`));
    await page.close();
  }
  console.log(`${errors.length ? "!" : "✓"} ${shot.name} → ${file}`);
  for (const error of errors) console.log(`    ${error.slice(0, 300)}`);
  if (errors.length) failures.push(shot.name);
}

await setScenario({});
await browser.close();
console.log(`\n${failures.length ? `${failures.length} shot(s) reported a problem: ${failures.join(", ")}` : "every shot went through"}`);
if (failures.length) process.exitCode = 1;
