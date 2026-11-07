// Walks through the audit log and Data retention against the mock API and writes one PNG per
// state. Usage: node e2e/mock-api/audit-screenshots.mjs [outDir] [name,name,...]
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/audit");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 4000);

async function openAudit(page, view = "") {
  await page.goto(`${APP}/admin/audit-logs${view ? `?view=${view}` : ""}`, {
    waitUntil: "domcontentloaded",
  });
  await page.getByRole("heading", { name: "Audit log" }).first().waitFor();
  await page.waitForTimeout(SETTLE_MS);
}

async function openEntry(page, text) {
  await openAudit(page);
  await page.getByRole("row").filter({ hasText: text }).first().click();
  await page.waitForTimeout(1500);
}

const SHOTS = [
  { name: "entries-light", colorScheme: "light", act: (page) => openAudit(page) },
  { name: "entries-dark", act: (page) => openAudit(page) },
  {
    name: "entry-update",
    colorScheme: "light",
    act: (page) => openEntry(page, "S-104223"),
  },
  {
    name: "entry-sensitive",
    colorScheme: "light",
    act: (page) => openEntry(page, "Acme Retail Group"),
  },
  {
    name: "entry-field-history",
    colorScheme: "light",
    act: async (page) => {
      await openEntry(page, "S-104223");
      await page.getByRole("button", { name: "History" }).last().click();
      await page.waitForTimeout(1200);
      await page.getByRole("button", { name: "Hide history" }).scrollIntoViewIfNeeded();
      await page.mouse.wheel(0, 400);
      await page.waitForTimeout(600);
    },
  },
  { name: "exports", colorScheme: "light", act: (page) => openAudit(page, "exports") },
  {
    name: "export-dialog",
    colorScheme: "light",
    act: async (page) => {
      await openAudit(page, "exports");
      await page.getByRole("button", { name: "New export" }).click();
      await page.waitForTimeout(800);
    },
  },
  { name: "legal-holds", colorScheme: "light", act: (page) => openAudit(page, "holds") },
  {
    name: "legal-hold-dialog",
    colorScheme: "light",
    act: async (page) => {
      await openAudit(page, "holds");
      await page.getByRole("button", { name: "New legal hold" }).click();
      await page.waitForTimeout(800);
    },
  },
  { name: "alerts", colorScheme: "light", act: (page) => openAudit(page, "alerts") },
  { name: "alerts-dark", act: (page) => openAudit(page, "alerts") },
  {
    name: "data-retention",
    colorScheme: "light",
    act: async (page) => {
      await page.goto(`${APP}/organization/data-retention`, { waitUntil: "domcontentloaded" });
      await page.getByText("Reads and downloads").waitFor();
      await page.waitForTimeout(1500);
    },
  },
  {
    name: "data-retention-below-floor",
    colorScheme: "light",
    act: async (page) => {
      await page.goto(`${APP}/organization/data-retention`, { waitUntil: "domcontentloaded" });
      await page.getByText("Financial changes").waitFor();
      const row = page.getByRole("row").filter({ hasText: "Financial changes" });
      const input = row.getByRole("spinbutton").or(row.getByRole("textbox")).first();
      await input.fill("365");
      await input.blur();
      await page.waitForTimeout(800);
    },
  },
  {
    name: "narrow-800",
    colorScheme: "light",
    viewport: { width: 800, height: 900 },
    act: (page) => openAudit(page),
  },
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
    body: JSON.stringify({ board: "quiet" }),
  }).then((response) => response.json());
  const page = await browser.newPage({
    viewport: shot.viewport ?? { width: 1440, height: 900 },
    colorScheme: shot.colorScheme ?? "dark",
    deviceScaleFactor: 2,
  });
  await page.clock.install({ time: new Date(scenario.anchor * 1000) });
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
  await page.screenshot({ path: file });
  console.log(`${errors.length ? "!" : "✓"} ${shot.name} → ${file}`);
  for (const error of errors) console.log(`    ${error.slice(0, 300)}`);
  if (errors.length) failures.push(shot.name);
  await page.close();
}

await browser.close();
if (failures.length) process.exitCode = 1;
