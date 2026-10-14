// Screenshots of the customer's tracking page and the shipment panel's
// Customer tracking tab. Run with the mock API and the web app up (see README.md):
//
//   node e2e/mock-api/customer-tracking-screenshots.mjs [outDir] [name,name,...]
//
// E2E_CHROMIUM_PATH points at a Chromium binary when Playwright's own is not installed.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/customer-tracking");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 3500);

const DESKTOP = { width: 1440, height: 1000 };
const MOBILE = { width: 390, height: 844 };
const PANEL = "/shipment-management/shipments?panelType=edit&panelEntityId=shp_0398";

const openTab = async (page) => {
  await page.getByRole("tab", { name: /Customer tracking/i }).first().click();
  await page.waitForTimeout(1500);
};

const SHOTS = [
  { name: "public-in-transit", path: "/shipment-tracking/in-transit", fullPage: true },
  { name: "public-delayed", path: "/shipment-tracking/delayed", fullPage: true },
  { name: "public-delivered", path: "/shipment-tracking/pod-ready", fullPage: true },
  { name: "public-planned", path: "/shipment-tracking/planned", fullPage: true },
  { name: "public-minimal", path: "/shipment-tracking/minimal", fullPage: true },
  { name: "public-invalid", path: "/shipment-tracking/expired" },
  { name: "public-in-transit-mobile", path: "/shipment-tracking/in-transit", viewport: MOBILE, fullPage: true },
  { name: "public-delayed-mobile", path: "/shipment-tracking/delayed", viewport: MOBILE, fullPage: true },
  { name: "public-in-transit-dark", path: "/shipment-tracking/in-transit", colorScheme: "dark", fullPage: true },
  { name: "tab", path: PANEL, act: openTab, settle: 9000 },
  { name: "tab-dark", path: PANEL, act: openTab, settle: 9000, colorScheme: "dark" },
  {
    name: "tab-shared",
    path: PANEL,
    settle: 9000,
    act: async (page) => {
      await openTab(page);
      await page.getByRole("button", { name: /Copy a link/i }).first().click();
      await page.waitForTimeout(1200);
    },
  },
];

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch(
  process.env.E2E_CHROMIUM_PATH ? { executablePath: process.env.E2E_CHROMIUM_PATH } : {},
);
for (const shot of SHOTS) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  const context = await browser.newContext({
    viewport: shot.viewport ?? DESKTOP,
    deviceScaleFactor: 2,
    colorScheme: shot.colorScheme ?? "light",
  });
  if (shot.colorScheme === "dark") {
    await context.addInitScript(() => localStorage.setItem("trenova-ui-theme", "dark"));
  }
  const page = await context.newPage();
  page.on("pageerror", (error) => console.log(`[${shot.name}] page error:`, error.message));
  await page.goto(`${APP}${shot.path}`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(shot.settle ?? SETTLE_MS);
  if (shot.act) await shot.act(page);
  if (shot.fullPage) {
    // The public pages scroll inside a fixed container, so grow the window to
    // the tallest scroller and let the page lay out again before the shot.
    const height = await page.evaluate(() =>
      Math.max(...[...document.querySelectorAll("*")].map((el) => el.scrollHeight)),
    );
    await page.setViewportSize({ width: (shot.viewport ?? DESKTOP).width, height });
    await page.waitForTimeout(600);
  }
  await page.screenshot({ path: `${OUT}/${shot.name}.png` });
  console.log(`saved ${shot.name}`);
  await context.close();
}
await browser.close();
