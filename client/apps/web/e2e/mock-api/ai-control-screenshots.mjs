// Screenshots of AI control (/admin/agent-control) against the mock API. See README.md.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const PAGE = `${APP}/admin/agent-control`;
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/ai-control");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 7000);

const DEFAULT_SCENARIO = { ai: true, aiProviders: "configured", aiPaused: false };

/** Scrolls the page's scrolling ancestor so the lower half of a tab is in view. */
async function scrollDown(page, by = 760) {
  await page.evaluate((amount) => {
    let element = document.querySelector(".aic");
    while (element && element !== document.body) {
      const style = getComputedStyle(element);
      if (/(auto|scroll)/.test(style.overflowY) && element.scrollHeight > element.clientHeight) {
        element.scrollTop += amount;
        return;
      }
      element = element.parentElement;
    }
    window.scrollBy(0, amount);
  }, by);
  await page.waitForTimeout(400);
}

const SHOTS = [
  { name: "overview-dark", scenario: {} },
  { name: "overview-lower", scenario: {}, act: (page) => scrollDown(page) },
  { name: "overview-light", scenario: {}, colorScheme: "light" },
  {
    name: "overview-tune-up-applied",
    scenario: {},
    act: async (page) => {
      await page.getByRole("button", { name: "Go live" }).click();
      await page.waitForTimeout(900);
    },
  },
  {
    name: "overview-tune-up-dismissed",
    scenario: {},
    act: async (page) => {
      await page.getByRole("button", { name: "Dismiss for 30 days" }).first().click();
      await page.waitForTimeout(900);
    },
  },
  { name: "overview-many-providers", scenario: { aiProviders: "many" } },
  { name: "overview-paused", scenario: { aiPaused: true } },
  { name: "overview-no-provider", scenario: { aiProviders: "none" } },
  {
    name: "overview-policy-editor",
    scenario: {},
    act: async (page) => {
      await page.getByRole("button", { name: "Edit", exact: true }).click();
      await page.waitForTimeout(700);
    },
  },
  { name: "overview-narrow-1000", scenario: {}, viewport: { width: 1000, height: 900 } },
  { name: "agents", scenario: {}, query: "?tab=agents" },
  { name: "agents-lower", scenario: {}, query: "?tab=agents", act: (page) => scrollDown(page, 900) },
  { name: "agents-light", scenario: {}, query: "?tab=agents", colorScheme: "light" },
  { name: "agents-shadow-filter", scenario: {}, query: "?tab=agents&agents=shadow" },
  { name: "agents-no-provider", scenario: { aiProviders: "none" }, query: "?tab=agents" },
  { name: "agents-paused", scenario: { aiPaused: true }, query: "?tab=agents" },
  {
    name: "agents-new-menu",
    scenario: {},
    query: "?tab=agents",
    act: async (page) => {
      await page.getByRole("button", { name: /New agent/ }).click();
    },
  },
  {
    name: "agents-sheet",
    scenario: {},
    query: "?tab=agents",
    act: async (page) => {
      await page.getByRole("button", { name: /Billing exceptions/ }).first().click();
      await page.waitForTimeout(500);
    },
  },
  {
    name: "agents-remove-confirm",
    scenario: {},
    query: "?tab=agents",
    act: async (page) => {
      await page.getByRole("button", { name: /Report analyst/ }).first().click();
      await page.waitForTimeout(400);
      await page.getByRole("button", { name: "Remove", exact: true }).click();
    },
  },
  { name: "builder", scenario: {}, query: "?tab=agents&panelType=edit&panelEntityId=agd_dispatch" },
  {
    name: "builder-tools",
    scenario: {},
    query: "?tab=agents&panelType=edit&panelEntityId=agd_dispatch",
    act: async (page) => {
      await page.getByRole("button", { name: /Tools and autonomy/ }).click();
      await page.waitForTimeout(900);
    },
  },
  {
    name: "builder-schedule",
    scenario: {},
    query: "?tab=agents&panelType=edit&panelEntityId=agd_digest",
    act: async (page) => {
      await page.getByRole("button", { name: /When it runs/ }).click();
      await page.waitForTimeout(900);
    },
  },
  {
    name: "builder-try",
    scenario: {},
    query: "?tab=agents&panelType=edit&panelEntityId=agd_dispatch",
    act: async (page) => {
      await page.getByRole("button", { name: "Try it", exact: true }).first().click();
      await page.waitForTimeout(400);
      await page.getByRole("button", { name: /What can't you do/ }).click();
      await page.waitForTimeout(5000);
    },
  },
  {
    name: "builder-unsaved",
    scenario: {},
    query: "?tab=agents&panelType=edit&panelEntityId=agd_dispatch",
    act: async (page) => {
      await page.getByRole("textbox", { name: "Name" }).fill("Dispatch desk (nights)");
      await page.getByRole("button", { name: /unsaved change/ }).click();
    },
  },
  {
    name: "builder-above-ceiling",
    scenario: {},
    query: "?tab=agents&panelType=edit&panelEntityId=agd_dispatch",
    act: async (page) => {
      await page.getByRole("button", { name: /Tools and autonomy/ }).click();
      await page.waitForTimeout(600);
      await page.getByRole("radio", { name: /Every change is a proposal/ }).click();
      await page.waitForTimeout(500);
      await page.locator(".cal.w").first().scrollIntoViewIfNeeded();
      await page.waitForTimeout(300);
    },
  },
  {
    name: "builder-tool-picker",
    scenario: {},
    query: "?tab=agents&panelType=edit&panelEntityId=agd_dispatch",
    act: async (page) => {
      await page.getByRole("button", { name: /Tools and autonomy/ }).click();
      await page.waitForTimeout(700);
      await page.getByRole("button", { name: "Add tools" }).first().click();
    },
  },
  { name: "builder-create", scenario: {}, query: "?tab=agents&panelType=create&panelEntityId=blank" },
  {
    name: "builder-create-started",
    scenario: {},
    query: "?tab=agents&panelType=create&panelEntityId=blank",
    act: async (page) => {
      await page.getByRole("button", { name: /Scheduled report/ }).click();
      await page.waitForTimeout(700);
    },
  },
  {
    name: "builder-create-no-provider",
    scenario: { aiProviders: "none" },
    query: "?tab=agents&panelType=create&panelEntityId=blank",
  },
  { name: "providers", scenario: {}, query: "?tab=providers" },
  { name: "providers-lower", scenario: {}, query: "?tab=providers", act: (page) => scrollDown(page, 700) },
  { name: "providers-light", scenario: {}, query: "?tab=providers", colorScheme: "light" },
  { name: "providers-many", scenario: { aiProviders: "many" }, query: "?tab=providers" },
  { name: "providers-empty", scenario: { aiProviders: "none" }, query: "?tab=providers" },
  { name: "providers-empty-light", scenario: { aiProviders: "none" }, query: "?tab=providers", colorScheme: "light" },
  { name: "providers-many-light", scenario: { aiProviders: "many" }, query: "?tab=providers", colorScheme: "light" },
  {
    name: "providers-show-off",
    scenario: { aiProviders: "many" },
    query: "?tab=providers",
    act: async (page) => {
      await page.getByRole("button", { name: /Show \d+ off/ }).click();
      await page.waitForTimeout(500);
    },
  },
  {
    name: "providers-new-menu",
    scenario: {},
    query: "?tab=providers",
    act: async (page) => {
      await page.getByRole("button", { name: /New provider/ }).click();
      await page.waitForTimeout(400);
    },
  },
  {
    name: "providers-sheet",
    scenario: {},
    query: "?tab=providers",
    act: async (page) => {
      await page.locator(".pl-t", { hasText: "Local Ollama" }).click();
      await page.waitForTimeout(800);
    },
  },
  {
    name: "providers-sheet-key",
    scenario: {},
    query: "?tab=providers",
    act: async (page) => {
      await page.getByRole("link", { name: "Anthropic", exact: true }).click();
      await page.waitForTimeout(800);
    },
  },
  {
    name: "provider-editor",
    scenario: {},
    query: "?tab=providers",
    act: async (page) => {
      await page.locator(".pl-t", { hasText: "Workstation vLLM" }).click();
      await page.waitForTimeout(600);
      await page.getByRole("button", { name: "Edit connection" }).first().click();
      await page.waitForTimeout(900);
    },
  },
  {
    name: "provider-editor-test-failed",
    scenario: {},
    query: "?tab=providers",
    act: async (page) => {
      await page.locator(".pl-t", { hasText: "Workstation vLLM" }).click();
      await page.waitForTimeout(600);
      await page.getByRole("button", { name: "Edit connection" }).first().click();
      await page.waitForTimeout(700);
      await page.getByRole("button", { name: "Test draft" }).click();
      await page.waitForTimeout(1200);
    },
  },
  {
    name: "provider-editor-impact",
    scenario: {},
    query: "?tab=providers",
    act: async (page) => {
      await page.locator(".pl-t", { hasText: "Workstation vLLM" }).click();
      await page.waitForTimeout(600);
      await page.getByRole("button", { name: "Edit connection" }).first().click();
      await page.waitForTimeout(700);
      const handles = page.getByRole("group", { name: "What it handles" });
      await handles.getByRole("button", { name: "Assistant chat" }).click();
      await handles.getByRole("button", { name: "Document extraction" }).click();
      await page.waitForTimeout(900);
      await page.locator("#es-tasks").scrollIntoViewIfNeeded();
      await page.waitForTimeout(400);
    },
  },
  {
    name: "provider-editor-key",
    scenario: { aiProviders: "many" },
    query: "?tab=providers",
    act: async (page) => {
      await page.locator(".pl-t", { hasText: /^OpenAI/ }).click();
      await page.waitForTimeout(600);
      await page.getByRole("button", { name: "Edit connection" }).first().click();
      await page.waitForTimeout(700);
      await page.getByRole("textbox", { name: "Replace key" }).fill("sk-proj-new-key-9c1d");
      await page.locator("#es-key").scrollIntoViewIfNeeded();
      await page.waitForTimeout(500);
    },
  },
  {
    name: "provider-editor-limits",
    scenario: { aiProviders: "many" },
    query: "?tab=providers",
    act: async (page) => {
      await page.locator(".pl-t", { hasText: /^OpenAI/ }).click();
      await page.waitForTimeout(600);
      await page.getByRole("button", { name: "Edit connection" }).first().click();
      await page.waitForTimeout(700);
      await page.locator("#es-limits").scrollIntoViewIfNeeded();
      await page.waitForTimeout(500);
    },
  },
  {
    name: "provider-create-local",
    scenario: {},
    query: "?tab=providers&panelType=create&panelEntityId=ollama",
    act: (page) => page.waitForTimeout(1500),
  },
  {
    name: "provider-create-hosted",
    scenario: {},
    query: "?tab=providers&panelType=create&panelEntityId=anthropic",
  },
  {
    name: "provider-create-private-url",
    scenario: {},
    query: "?tab=providers&panelType=create&panelEntityId=openrouter",
    act: async (page) => {
      await page.getByRole("textbox", { name: "Base URL" }).fill("http://192.168.1.40:8000/v1");
      await page.waitForTimeout(500);
    },
  },
  { name: "extensions", scenario: {}, query: "?tab=extensions" },
  { name: "extensions-light", scenario: {}, query: "?tab=extensions", colorScheme: "light" },
  {
    name: "extensions-sheet",
    scenario: {},
    query: "?tab=extensions",
    act: async (page) => {
      await page.getByRole("button", { name: /Set up Web search/ }).click();
      await page.waitForTimeout(900);
    },
  },
  {
    name: "extensions-sheet-agents",
    scenario: {},
    query: "?tab=extensions",
    act: async (page) => {
      await page.getByRole("button", { name: /Set up Web search/ }).click();
      await page.waitForTimeout(900);
      await page.locator(".ex-ag").scrollIntoViewIfNeeded();
      await page.waitForTimeout(300);
    },
  },
  {
    name: "extensions-on",
    scenario: {},
    query: "?tab=extensions",
    act: async (page) => {
      await page.getByRole("button", { name: /Set up Web search/ }).click();
      await page.waitForTimeout(700);
      await page.getByRole("textbox", { name: "API key" }).fill("exa-test-key-1234");
      await page.getByRole("button", { name: "Turn on", exact: true }).click();
      await page.waitForTimeout(1200);
      await page.keyboard.press("Escape");
      await page.waitForTimeout(700);
    },
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

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium-1194/chrome-linux/chrome",
});
const failures = [];

for (const shot of SHOTS) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  const { anchor } = await setScenario(shot.scenario);
  const page = await browser.newPage({
    viewport: shot.viewport ?? { width: 1440, height: 1000 },
    colorScheme: shot.colorScheme ?? "dark",
  });
  await page.clock.install({ time: new Date(anchor * 1000) });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().includes("Failed to load resource")) {
      errors.push(message.text());
    }
  });
  await page.goto(`${PAGE}${shot.query ?? ""}`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(SETTLE_MS);
  try {
    if (shot.act) {
      await shot.act(page);
      await page.waitForTimeout(600);
    }
  } catch (error) {
    errors.push(`interaction failed: ${error.message.split("\n")[0]}`);
  }
  const file = `${OUT}/${shot.name}.png`;
  await page.screenshot({ path: file });
  console.log(`${errors.length ? "!" : "✓"} ${shot.name} → ${file}`);
  for (const error of errors) console.log(`    ${error.slice(0, 300)}`);
  if (errors.length) failures.push(shot.name);
  await page.close();
}

await setScenario({});
await browser.close();
if (failures.length) process.exitCode = 1;
