// Walks through the Settings page against the mock API and writes one PNG per state. Usage: node e2e/mock-api/settings-screenshots.mjs [outDir] [name,name,...]
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const PAGE = `${APP}/shipment-management/shipments`;
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/settings");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 6000);

async function openSettings(page) {
  await page.getByRole("heading", { name: "Shipments" }).first().click();
  await page.keyboard.press("Control+Shift+S");
  await page.waitForURL("**/settings");
  await settingsReady(page);
}

async function settingsReady(page) {
  await page.getByRole("navigation", { name: "Settings sections" }).waitFor();
  await page.getByText("All changes saved").waitFor();
  await page.waitForTimeout(800);
}

async function jumpTo(page, section) {
  await page
    .getByRole("navigation", { name: "Settings sections" })
    .getByRole("button", { name: section })
    .click();
  await page.waitForTimeout(900);
}

const SHOTS = [
  { name: "profile-light", colorScheme: "light", act: openSettings },
  { name: "profile-dark", act: openSettings },
  {
    name: "profile-away",
    colorScheme: "light",
    act: async (page) => {
      await openSettings(page);
      await page.getByRole("radio", { name: "Away" }).click();
      await page.waitForTimeout(400);
      await page.locator("#settings-row-availability").scrollIntoViewIfNeeded();
      await page.waitForTimeout(500);
    },
  },
  {
    name: "preferences",
    colorScheme: "light",
    act: async (page) => {
      await openSettings(page);
      await jumpTo(page, "Preferences");
    },
  },
  {
    name: "notifications",
    colorScheme: "light",
    act: async (page) => {
      await openSettings(page);
      await jumpTo(page, "Notifications");
    },
  },
  {
    name: "notifications-types-expanded",
    colorScheme: "light",
    act: async (page) => {
      await openSettings(page);
      await jumpTo(page, "Notifications");
      const card = page.locator("#settings-row-types");
      await card
        .getByRole("button", { name: /of 6|All 6/ })
        .first()
        .click();
      await page.waitForTimeout(300);
      await card.scrollIntoViewIfNeeded();
      await page.waitForTimeout(400);
    },
  },
  {
    name: "notifications-dark",
    act: async (page) => {
      await openSettings(page);
      await jumpTo(page, "Notifications");
    },
  },
  {
    name: "saved-undo-toast",
    colorScheme: "light",
    act: async (page) => {
      await openSettings(page);
      await jumpTo(page, "Notifications");
      await page.getByRole("radio", { name: "Everything" }).first().click();
      await page.waitForTimeout(400);
    },
  },
  {
    name: "paused",
    colorScheme: "light",
    act: async (page) => {
      await openSettings(page);
      await jumpTo(page, "Notifications");
      await page.locator("#settings-row-pause button").first().click();
      await page.getByText("For 1 hour", { exact: true }).click();
      await page.waitForTimeout(1200);
    },
  },
  {
    name: "security",
    colorScheme: "light",
    act: async (page) => {
      await openSettings(page);
      await jumpTo(page, "Security");
    },
  },
  {
    name: "security-password",
    colorScheme: "light",
    act: async (page) => {
      await openSettings(page);
      await jumpTo(page, "Security");
      await page.getByRole("button", { name: "Change password" }).click();
      await page.locator('input[name="currentPassword"]').fill("correct-horse");
      await page.locator('input[name="newPassword"]').fill("Reefer-temp");
      await page.waitForTimeout(400);
    },
  },
  {
    name: "search",
    colorScheme: "light",
    act: async (page) => {
      await openSettings(page);
      await page.keyboard.press("Control+K");
      await page.keyboard.type("tender");
      await page.waitForTimeout(500);
    },
  },
  {
    name: "search-jump",
    colorScheme: "light",
    act: async (page) => {
      await openSettings(page);
      await page.keyboard.press("Control+K");
      await page.keyboard.type("quiet");
      await page.keyboard.press("Enter");
      await page.waitForTimeout(500);
    },
  },
  {
    name: "narrow-800",
    colorScheme: "light",
    viewport: { width: 800, height: 900 },
    act: openSettings,
  },
  {
    name: "back-returns",
    colorScheme: "light",
    act: async (page) => {
      await openSettings(page);
      await page.goBack();
      await page.getByRole("heading", { name: "Shipments" }).first().waitFor();
      await page.waitForTimeout(600);
    },
  },
  {
    name: "email-link",
    colorScheme: "light",
    act: async (page) => {
      await page.goto(`${APP}/?settings=notifications`, { waitUntil: "domcontentloaded" });
      await page.waitForURL("**/settings");
      await settingsReady(page);
    },
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
    body: JSON.stringify({ board: "busy" }),
  }).then((response) => response.json());
  const page = await browser.newPage({
    viewport: shot.viewport ?? { width: 1440, height: 900 },
    colorScheme: shot.colorScheme ?? "dark",
  });
  await page.clock.install({ time: new Date(scenario.anchor * 1000) });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().includes("Failed to load resource")) {
      errors.push(message.text());
    }
  });
  await page.goto(PAGE, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(SETTLE_MS);
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
