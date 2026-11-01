// Walks the accounting integration modal (every setup step, the connected tabs and the
// connection's states) and the connect dialogs against the mock API, one PNG per state.
// Usage: node e2e/mock-api/accounting-screenshots.mjs [outDir] [name,name,...]
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/accounting");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 1500);
const PHONE = { width: 390, height: 844 };

const tab = (name) => async (page) => {
  await page.getByRole("tab", { name }).click();
  await page.waitForTimeout(500);
};
const radio = (name) => async (page) => {
  await page.getByRole("radio", { name }).click();
  await page.waitForTimeout(400);
};

const SHOTS = [
  { name: "catalog", path: "", scenario: { accounting: "connected" }, page: true },
  { name: "connect-quickbooks", type: "QuickBooksOnline" },
  { name: "connect-quickbooks-dark", type: "QuickBooksOnline", colorScheme: "dark" },
  { name: "connect-quickbooks-phone", type: "QuickBooksOnline", viewport: PHONE, page: true },
  { name: "connect-no-app", type: "QuickBooksOnline", scenario: { accountingApp: "none" } },
  { name: "connect-netsuite", type: "NetSuite" },
  {
    name: "connect-netsuite-certificate",
    type: "NetSuite",
    act: radio(/^Connect with a certificate/),
  },
  { name: "connect-xero-elsewhere", type: "Xero", scenario: { accounting: "connected" } },
  {
    name: "review",
    type: "QuickBooksOnline",
    setup: "connected",
    scenario: { accounting: "mode" },
  },
  { name: "mode", type: "QuickBooksOnline", scenario: { accounting: "mode" } },
  {
    name: "mode-journal-entries",
    type: "QuickBooksOnline",
    scenario: { accounting: "mode" },
    act: radio(/^Send journal entries/),
  },
  { name: "mode-xero", type: "Xero", scenario: { accounting: "mode", accountingSystem: "Xero" } },
  { name: "mappings", type: "QuickBooksOnline", scenario: { accounting: "mappings" } },
  {
    name: "mappings-dark",
    type: "QuickBooksOnline",
    scenario: { accounting: "mappings" },
    colorScheme: "dark",
  },
  { name: "start-date", type: "QuickBooksOnline", scenario: { accounting: "startDate" } },
  { name: "connected", type: "QuickBooksOnline", scenario: { accounting: "connected" } },
  {
    name: "connected-dark",
    type: "QuickBooksOnline",
    scenario: { accounting: "connected" },
    colorScheme: "dark",
  },
  {
    name: "connected-phone",
    type: "QuickBooksOnline",
    scenario: { accounting: "connected" },
    viewport: PHONE,
    page: true,
  },
  {
    name: "connected-settings",
    type: "QuickBooksOnline",
    scenario: { accounting: "connected" },
    act: tab("Settings"),
  },
  {
    name: "connected-app",
    type: "QuickBooksOnline",
    scenario: { accounting: "connected" },
    act: tab("Intuit app"),
  },
  { name: "failing", type: "QuickBooksOnline", scenario: { accounting: "failing" } },
  { name: "revoked", type: "QuickBooksOnline", scenario: { accounting: "revoked" } },
  {
    name: "expiring-netsuite",
    type: "NetSuite",
    scenario: { accounting: "expiring", accountingSystem: "NetSuite" },
  },
  {
    name: "business-central",
    type: "BusinessCentral",
    scenario: { accounting: "degraded", accountingSystem: "BusinessCentral" },
  },
  {
    name: "netsuite-certificate",
    type: "NetSuite",
    scenario: {
      accounting: "connected",
      accountingSystem: "NetSuite",
      accountingAuth: "certificate",
    },
  },
  { name: "resend", type: "Resend" },
  { name: "resend-dark", type: "Resend", colorScheme: "dark" },
  { name: "pcmiler", type: "PCMiler" },
  { name: "google-maps", type: "GoogleMaps" },
];

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium-1194/chrome-linux/chrome",
});
const failures = [];

for (const shot of SHOTS) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  const scenario = await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      board: "quiet",
      accounting: "connect",
      accountingSystem: "QuickBooksOnline",
      accountingApp: "instance",
      accountingAuth: "signIn",
      ...shot.scenario,
    }),
  }).then((response) => response.json());
  const page = await browser.newPage({
    viewport: shot.viewport ?? { width: 1440, height: 900 },
    colorScheme: shot.colorScheme ?? "light",
  });
  await page.clock.install({ time: new Date(scenario.anchor * 1000) });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().includes("Failed to load resource")) {
      errors.push(message.text());
    }
  });
  const query = new URLSearchParams();
  if (shot.type) query.set("type", shot.type);
  if (shot.setup) query.set("setup", shot.setup);
  await page.goto(`${APP}/admin/integrations${query.size ? `?${query}` : ""}`, {
    waitUntil: "domcontentloaded",
  });
  // The first load after Vite starts can take a while; wait for what the shot is of.
  const ready = shot.type
    ? page.getByRole("dialog").first()
    : page.getByRole("heading", { name: "Integrations" }).first();
  await ready.waitFor({ timeout: 30_000 }).catch(() => errors.push("never became ready"));
  await page.waitForTimeout(SETTLE_MS);
  try {
    if (shot.act) await shot.act(page);
  } catch (error) {
    errors.push(`interaction failed: ${error.message.split("\n")[0]}`);
  }
  const file = `${OUT}/${shot.name}.png`;
  const dialog = page.getByRole("dialog").first();
  if (!shot.page && (await dialog.count())) await dialog.screenshot({ path: file });
  else await page.screenshot({ path: file, fullPage: shot.name === "catalog" });
  console.log(`${errors.length ? "!" : "✓"} ${shot.name} → ${file}`);
  for (const error of errors) console.log(`    ${error.slice(0, 300)}`);
  if (errors.length) failures.push(shot.name);
  await page.close();
}

await browser.close();
if (failures.length) process.exitCode = 1;
