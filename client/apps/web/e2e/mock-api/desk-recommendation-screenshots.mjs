// Screenshots of the Desk home suggesting the agent with the tools for a
// question. Run with the mock API and the web app up (see README.md):
//
//   node e2e/mock-api/desk-recommendation-screenshots.mjs [outDir] [name,name,...]
//
// A fresh browser starts on the first agent by name, Billing. The question is
// about a customer's ETA, which Customer service holds the tools for, so the
// suggestion names it. Every shot starts from a fresh scenario.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/desk-recommendation");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 6000);
const QUESTION = "When will the Acme load arrive? Send the customer an ETA reply.";

const composer = (page) => page.getByRole("textbox", { name: /^Message/ }).first();

async function typeQuestion(page) {
  await composer(page).fill(QUESTION);
  // The match is asked for once typing pauses.
  await page.waitForTimeout(2000);
}

function shotsFor(colorScheme, viewport = { width: 1440, height: 900 }, suffix = "") {
  return [
    { name: "home", act: async () => undefined },
    { name: "suggested", act: typeQuestion },
    {
      name: "kept",
      act: async (page) => {
        await typeQuestion(page);
        await page.getByRole("button", { name: /^Keep / }).click();
      },
    },
    {
      name: "switched",
      act: async (page) => {
        await typeQuestion(page);
        await page.getByRole("button", { name: "Switch" }).click();
      },
    },
    {
      name: "finding",
      scenario: { agentMatch: "hold" },
      act: async (page) => {
        await composer(page).fill(QUESTION);
        await page.keyboard.press("Enter");
        await page.waitForTimeout(300);
      },
      shootNow: true,
    },
    { name: "no-match", scenario: { agentMatch: "none" }, act: typeQuestion },
  ].map((shot) => ({
    ...shot,
    name: `${shot.name}-${colorScheme}${suffix}`,
    colorScheme,
    viewport,
  }));
}

async function setScenario(patch) {
  const response = await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ agentMatch: "match", ...patch }),
  });
  return response.json();
}

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium-1194/chrome-linux/chrome",
});
const failures = [];

const shots = [
  ...shotsFor("light"),
  ...shotsFor("dark"),
  ...shotsFor("light", { width: 390, height: 844 }, "-phone").filter((shot) =>
    shot.name.startsWith("suggested"),
  ),
  ...shotsFor("dark", { width: 390, height: 844 }, "-phone").filter((shot) =>
    shot.name.startsWith("suggested"),
  ),
];

for (const shot of shots) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  const { anchor } = await setScenario(shot.scenario ?? {});
  const context = await browser.newContext({
    viewport: shot.viewport,
    deviceScaleFactor: 2,
    colorScheme: shot.colorScheme,
  });
  await context.addInitScript((theme) => {
    window.localStorage.setItem("vite-ui-theme", theme);
  }, shot.colorScheme);
  const page = await context.newPage();
  await page.clock.install({ time: new Date(anchor * 1000) });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().includes("Failed to load resource")) {
      errors.push(message.text());
    }
  });
  await page.goto(`${APP}/desk`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(SETTLE_MS);
  try {
    await shot.act(page);
    if (!shot.shootNow) await page.waitForTimeout(1200);
  } catch (error) {
    errors.push(`step: ${error.message.split("\n")[0]}`);
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
if (failures.length) {
  console.error(`${failures.length} shot(s) had errors: ${failures.join(", ")}`);
  process.exit(1);
}
