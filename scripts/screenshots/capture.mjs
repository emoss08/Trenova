/**
 * Feature screenshot capture for docs/screenshots.
 *
 * Usage (from repo root, with the API on :8080 and the web client on :5173):
 *
 *   node scripts/screenshots/capture.mjs \
 *     --pages scripts/screenshots/pages.json \
 *     --out docs/screenshots/light
 *
 *   node scripts/screenshots/capture.mjs \
 *     --pages scripts/screenshots/pages-dark.json \
 *     --out docs/screenshots/dark --theme dark
 *
 * Credentials come from TRENOVA_USER / TRENOVA_PASSWORD.
 *
 * Playwright resolves from client/apps/web, so run this with that package's
 * node_modules on the resolution path, or run it from client/apps/web.
 */
import { createRequire } from "node:module";
import fs from "node:fs";
import path from "node:path";

const repoRoot = path.resolve(import.meta.dirname, "..", "..");
const require = createRequire(
  path.join(repoRoot, "client", "apps", "web", "package.json"),
);
const { chromium } = require("playwright");

function arg(name, fallback) {
  const i = process.argv.indexOf(`--${name}`);
  return i === -1 ? fallback : process.argv[i + 1];
}

const BASE = arg("base", "http://localhost:5173");
const OUT = path.resolve(arg("out", "docs/screenshots/light"));
const THEME = arg("theme", "light");
const PAGES = JSON.parse(fs.readFileSync(arg("pages", "scripts/screenshots/pages.json"), "utf8"));
const USER = process.env.TRENOVA_USER;
const PASSWORD = process.env.TRENOVA_PASSWORD;

if (!USER || !PASSWORD) {
  console.error("Set TRENOVA_USER and TRENOVA_PASSWORD.");
  process.exit(1);
}

fs.mkdirSync(OUT, { recursive: true });

const browser = await chromium.launch({ headless: true });
const ctx = await browser.newContext({
  viewport: { width: 1600, height: 1000 },
  deviceScaleFactor: 2,
  colorScheme: THEME === "dark" ? "dark" : "light",
});
const page = await ctx.newPage();

const serverErrors = [];
page.on("response", (r) => {
  if (r.status() >= 500) serverErrors.push(`${r.status()} ${r.url().slice(0, 160)}`);
});

async function settle(ms) {
  try {
    await page.waitForLoadState("networkidle", { timeout: 15000 });
  } catch {
    // The app polls for realtime and permission updates, so the network never
    // truly idles on some routes. The fixed wait below is the real settle.
  }
  await page.waitForTimeout(ms);
  try {
    await page
      .locator('[data-slot="skeleton"], .animate-pulse')
      .first()
      .waitFor({ state: "detached", timeout: 8000 });
  } catch {
    // No skeleton on this route, or it never detaches; the wait above covers it.
  }
  await page.waitForTimeout(600);
}

async function login() {
  await page.goto(`${BASE}/login`, { waitUntil: "domcontentloaded" });
  await page.waitForSelector('input[name="emailAddress"]', { timeout: 60000 });
  await page.waitForTimeout(2500);
  await page.screenshot({ path: path.join(OUT, "sign-in.png") });

  await page.fill('input[name="emailAddress"]', USER);
  await page.fill('input[name="password"]', PASSWORD);
  await page.click('button[type="submit"]');

  // Sessions start with no active roles. Until the gate is cleared every route
  // renders `403 Missing <resource>:Read`, so this is not optional.
  const activate = page.locator("button").filter({ hasText: /^Activate \d+ role/ });
  for (let i = 0; i < 60; i++) {
    if (await activate.count()) break;
    await page.waitForTimeout(1000);
    if (i === 20 && page.url().includes("/login")) {
      await page.goto(`${BASE}/`, { waitUntil: "domcontentloaded" });
    }
  }
  if (!(await activate.count())) {
    throw new Error("role activation gate never appeared; captures would all be 403");
  }

  const roles = page.locator("button[aria-pressed]");
  for (let i = 0; i < (await roles.count()); i++) {
    if ((await roles.nth(i).getAttribute("aria-pressed")) !== "true") {
      await roles.nth(i).click();
    }
  }
  await page.waitForTimeout(300);
  await activate.first().click();
  await activate.first().waitFor({ state: "detached", timeout: 30000 });

  // The permission manifest is refetched after activation, so give it a couple
  // of attempts before declaring the session unusable.
  await page.waitForTimeout(2000);
  for (let attempt = 1; ; attempt++) {
    await page.goto(`${BASE}/hr/workers`, { waitUntil: "domcontentloaded" });
    await page.waitForTimeout(5000);
    if (!/Missing worker:Read/.test(await page.locator("body").innerText())) break;
    if (attempt === 3) throw new Error("still unauthorized after activating roles");
    await page.waitForTimeout(3000);
  }
}

await login();
await page.waitForTimeout(2000);

const failed = [];
for (const p of PAGES) {
  const file = path.join(OUT, `${p.name}.png`);
  try {
    await page.goto(`${BASE}${p.path}`, { waitUntil: "domcontentloaded", timeout: 60000 });
    await settle(p.wait ?? 2500);
    await page.screenshot({ path: file });

    // Two failure modes survive a first paint: a stale Vite dep cache renders an
    // error boundary, and a page that never repainted writes a near-blank PNG.
    // Both clear on reload; anything still broken after three tries is real.
    for (let retry = 1; retry <= 3; retry++) {
      const text = await page.locator("body").innerText();
      const broken =
        /Failed to fetch dynamically imported module|ApiRequestError|Try again Copy error/i.test(text);
      const blank = fs.statSync(file).size < 60 * 1024;
      if (!broken && !blank) break;
      await page.reload({ waitUntil: "domcontentloaded", timeout: 60000 });
      await settle((p.wait ?? 2500) + 1500);
      await page.screenshot({ path: file });
      if (retry === 3) failed.push(`${p.name} (${broken ? "error boundary" : "blank"})`);
    }
    console.log(`ok   ${p.name}`);
  } catch (e) {
    failed.push(`${p.name} (${String(e).slice(0, 120)})`);
    console.log(`FAIL ${p.name}: ${String(e).slice(0, 160)}`);
  }
}

await browser.close();

if (serverErrors.length) {
  console.log("\n5xx responses observed:");
  [...new Set(serverErrors)].forEach((e) => console.log("  " + e));
}
if (failed.length) {
  console.log("\nPages that did not capture cleanly:");
  failed.forEach((f) => console.log("  " + f));
  process.exit(1);
}
console.log(`\n${PAGES.length} pages captured to ${OUT}`);
