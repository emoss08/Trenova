// Screenshots of the Desk's resource cards, each drawn where people see it:
// beside the conversation that looked the record up, and on its own page at
// full length. Run with the mock API and the web app up (see README.md):
//
//   node e2e/mock-api/desk-cards-screenshots.mjs [outDir] [name,name,...]
//
// Every card is shot loaded (light and dark), with something wrong on it,
// empty, as its skeleton (the mock holds the card's requests open), unreadable,
// at full length and in a narrow window.
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/desk-cards");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 6000);
const THREAD = "thr_desk_cards";

/** Each card's conversations: the record loaded, one with something wrong, one with nothing, one gone. */
const CARDS = [
  {
    card: "tractor",
    loaded: "tractor-t-101",
    attention: "tractor-t-207",
    empty: "tractor-t-318",
    gone: "tractor-t-999",
  },
  {
    card: "trailer",
    loaded: "trailer-tl-501",
    attention: "trailer-tl-622",
    empty: "trailer-tl-700",
    gone: "trailer-tl-999",
  },
  {
    card: "worker",
    loaded: "worker-rosa",
    attention: "worker-ana",
    empty: "worker-jordan",
    gone: "worker-gone",
  },
  {
    card: "location",
    loaded: "location-dallas",
    attention: "location-tyson",
    empty: "location-new",
    gone: "location-gone",
  },
  {
    card: "carrier",
    loaded: "carrier-redriver",
    attention: "carrier-lonestar",
    empty: "carrier-new",
    gone: "carrier-gone",
  },
  {
    card: "order",
    loaded: "order-acme",
    attention: "order-tyson",
    empty: "order-draft",
    gone: "order-gone",
  },
  {
    card: "rate-agreement",
    loaded: "agreement-acme",
    attention: "agreement-lonestar",
    empty: "agreement-globex",
    gone: "agreement-gone",
  },
  {
    card: "service-failure",
    loaded: "failure-tyson-appt",
    attention: "failure-tyson-late",
    empty: "failure-new",
    gone: "failure-gone",
  },
  {
    card: "edi-message",
    loaded: "edi-214-tyson",
    attention: "edi-210-acme",
    empty: "edi-214-old",
    gone: "edi-gone",
  },
  {
    card: "edi-inbound",
    loaded: "edi-204-acme",
    attention: "edi-997-rejected",
    empty: "edi-990-acme",
    gone: "edi-gone",
  },
  {
    card: "edi-file",
    loaded: "edi-file-sim",
    attention: "edi-file-acme",
    empty: "edi-file-old",
    gone: "edi-file-gone",
  },
  {
    card: "edi-file-held",
    loaded: "edi-file-bad",
    attention: "edi-file-dup",
    empty: "edi-file-old",
    gone: "edi-file-gone",
  },
  {
    card: "tender",
    loaded: "tender-accepted",
    attention: "tender-mapping",
    empty: "tender-superseded",
    gone: "tender-gone",
  },
  {
    card: "tender-closed",
    loaded: "tender-outbound",
    attention: "tender-failed",
    empty: "tender-declined",
    gone: "tender-gone",
  },
  {
    card: "invoice",
    loaded: "invoice-acme",
    attention: "invoice-tyson-draft",
    empty: "invoice-riverside",
    gone: "invoice-gone",
  },
  {
    card: "invoice-closed",
    loaded: "invoice-credit",
    attention: "invoice-voided",
    empty: "invoice-riverside",
    gone: "invoice-gone",
  },
  {
    card: "worker-settlement",
    loaded: "pay-rosa",
    attention: "pay-marcus",
    empty: "pay-jordan",
    gone: "pay-gone",
  },
  {
    card: "worker-settlement-closed",
    loaded: "pay-paid",
    attention: "pay-voided",
    empty: "pay-jordan",
    gone: "pay-gone",
  },
  {
    card: "carrier-settlement",
    loaded: "payable-ridge",
    attention: "payable-lonestar",
    empty: "payable-mesa",
    gone: "payable-gone",
  },
  {
    card: "carrier-settlement-closed",
    loaded: "payable-paid",
    attention: "payable-voided",
    empty: "payable-mesa",
    gone: "payable-gone",
  },
  {
    card: "detention",
    loaded: "detention-kroger",
    attention: "detention-lineage",
    empty: "detention-quick",
    gone: "detention-gone",
  },
  {
    card: "detention-closed",
    loaded: "detention-disputed",
    attention: "detention-hd",
    empty: "detention-quick",
    gone: "detention-gone",
  },
  {
    card: "edi-partner",
    loaded: "partner-tyson",
    attention: "partner-acme",
    empty: "partner-kroger",
    gone: "partner-gone",
  },
  {
    card: "edi-partner-internal",
    loaded: "partner-lonestar",
    attention: "partner-acme",
    empty: "partner-kroger",
    gone: "partner-gone",
  },
  {
    card: "journal-entry",
    loaded: "journal-posted",
    attention: "journal-pending",
    empty: "journal-reversed",
    gone: "journal-gone",
  },
  {
    card: "manual-journal",
    loaded: "manual-pending",
    attention: "manual-draft",
    empty: "manual-posted",
    gone: "manual-gone",
  },
  {
    card: "manual-journal-closed",
    loaded: "manual-rejected",
    attention: "manual-draft",
    empty: "manual-posted",
    gone: "manual-gone",
  },
];

function shotsFor({ card, loaded, attention, empty, gone }) {
  const pane = (slug) => `/desk/c/${THREAD}/a/${slug}`;
  const page = (slug) => `/desk/c/${THREAD}/a/${slug}/page`;
  return [
    { name: `${card}-loaded-light`, path: pane(loaded), slug: loaded, colorScheme: "light" },
    { name: `${card}-loaded-dark`, path: pane(loaded), slug: loaded, colorScheme: "dark" },
    {
      name: `${card}-attention-light`,
      path: pane(attention),
      slug: attention,
      colorScheme: "light",
    },
    { name: `${card}-attention-dark`, path: pane(attention), slug: attention, colorScheme: "dark" },
    { name: `${card}-empty-light`, path: pane(empty), slug: empty, colorScheme: "light" },
    { name: `${card}-empty-dark`, path: pane(empty), slug: empty, colorScheme: "dark" },
    {
      name: `${card}-skeleton-light`,
      path: pane(loaded),
      slug: loaded,
      colorScheme: "light",
      desk: "hold",
    },
    {
      name: `${card}-skeleton-dark`,
      path: pane(loaded),
      slug: loaded,
      colorScheme: "dark",
      desk: "hold",
    },
    { name: `${card}-error-light`, path: pane(gone), slug: gone, colorScheme: "light" },
    {
      name: `${card}-page-light`,
      path: page(loaded),
      slug: loaded,
      colorScheme: "light",
      fullPage: true,
    },
    {
      name: `${card}-page-dark`,
      path: page(loaded),
      slug: loaded,
      colorScheme: "dark",
      fullPage: true,
    },
    {
      name: `${card}-attention-page-light`,
      path: page(attention),
      slug: attention,
      colorScheme: "light",
      fullPage: true,
    },
    {
      name: `${card}-empty-page-light`,
      path: page(empty),
      slug: empty,
      colorScheme: "light",
      fullPage: true,
    },
    {
      name: `${card}-narrow`,
      path: pane(attention),
      slug: attention,
      colorScheme: "light",
      viewport: { width: 1000, height: 820 },
    },
  ];
}

async function setScenario(patch) {
  const response = await fetch(`${MOCK}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(patch),
  });
  return response.json();
}

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium-1194/chrome-linux/chrome",
});
const failures = [];

for (const shot of CARDS.flatMap(shotsFor)) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  const { anchor } = await setScenario({ desk: shot.desk ?? "full", deskSlug: shot.slug });
  const context = await browser.newContext({
    viewport: shot.viewport ?? { width: 1440, height: 900 },
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
  await page.goto(`${APP}${shot.path}`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(SETTLE_MS);
  const file = `${OUT}/${shot.name}.png`;
  await page.screenshot({ path: file, fullPage: shot.fullPage ?? false });
  console.log(`${errors.length ? "!" : "✓"} ${shot.name} → ${file}`);
  for (const error of errors) console.log(`    ${error.slice(0, 300)}`);
  if (errors.length) failures.push(shot.name);
  await context.close();
}

await setScenario({ desk: "full", deskSlug: "tractor-t-101" });
await browser.close();
if (failures.length) process.exitCode = 1;
