// Screenshots of the command palette (design_handoff_command_palette) against the mock API.
//
//   MOCK_PALETTE=1 node e2e/mock-api/server.mjs
//   pnpm --filter @trenova/web dev
//   node e2e/mock-api/palette-screenshots.mjs [outDir] [only,names]

import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const APP = process.env.APP_URL ?? "http://localhost:5173";
const PAGE = `${APP}/shipment-management/shipments`;
const OUT = resolve(process.argv[2] ?? "e2e/mock-api/screenshots/palette");
const ONLY = process.argv[3] ? new Set(process.argv[3].split(",")) : null;
const SETTLE_MS = Number(process.env.SETTLE_MS ?? 6000);
const MOD = process.platform === "darwin" ? "Meta" : "Control";

const MINUTE = 60_000;

/** What the person opened last, so "Jump back in" has the handoff's five records. */
function recentRecords(now) {
  const shipment = (id, pro, meta, minutesAgo) => ({
    entityType: "shipment",
    id,
    title: pro,
    href: `/shipment-management/shipments?expanded=${id}&panelEntityId=${id}&panelType=edit`,
    metadata: meta,
    visitedAt: now - minutesAgo * MINUTE,
  });
  return [
    shipment(
      "shp_cp_240193",
      "S-240193",
      {
        proNumber: "S-240193",
        status: "InTransit",
        customerName: "Acme Foods",
        customerId: "cus_cp_acme",
        origin: "Dallas, TX",
        destination: "Atlanta, GA",
        driverName: "Marcus Bell",
        bol: "88213",
      },
      8,
    ),
    {
      entityType: "customer",
      id: "cus_cp_acme",
      title: "Acme Foods",
      subtitle: "ACME",
      href: "/billing/configuration-files/customers?panelEntityId=cus_cp_acme&panelType=edit",
      metadata: {
        status: "Active",
        code: "ACME",
        city: "Dallas, TX",
        openBalance: "48210.00",
        currencyCode: "USD",
      },
      visitedAt: now - 25 * MINUTE,
    },
    {
      entityType: "worker",
      id: "wrk_cp_1042",
      title: "Marcus Bell",
      href: "/hr/workers?panelEntityId=wrk_cp_1042&panelType=edit",
      metadata: {
        status: "Active",
        externalId: "W-1042",
        workerType: "Employee",
        driverType: "OTR",
        city: "Dallas, TX",
        currentLoad: "S-240193",
        currentLoadId: "shp_cp_240193",
        currentLoadStatus: "InTransit",
      },
      visitedAt: now - 62 * MINUTE,
    },
    {
      entityType: "document",
      id: "doc_cp_9921",
      title: "BOL-88213.pdf",
      href: "/shipment-management/shipments?expanded=shp_cp_240193&panelEntityId=shp_cp_240193&panelType=edit",
      metadata: {
        resourceType: "shipment",
        resourceId: "shp_cp_240193",
        detectedKind: "BillOfLading",
        attachedLabel: "S-240193",
        receivedAt: String(Math.round((now - 27.6 * 60 * MINUTE) / 1000)),
      },
      visitedAt: now - 130 * MINUTE,
    },
    shipment(
      "shp_cp_240201",
      "S-240201",
      {
        proNumber: "S-240201",
        status: "Delayed",
        customerName: "Halvorsen Steel",
        customerId: "cus_cp_halv",
        origin: "Gary, IN",
        destination: "Nashville, TN",
        driverName: "Leon Park",
      },
      190,
    ),
  ];
}

const PALETTE_INPUT = "[role=dialog] [cmdk-input]";

async function open(page) {
  for (let attempt = 0; attempt < 4; attempt++) {
    await page.locator("body").click({ position: { x: 5, y: 5 } });
    await page.keyboard.press(`${MOD}+k`);
    try {
      await page.locator(PALETTE_INPUT).waitFor({ timeout: 2500 });
      break;
    } catch {
      continue;
    }
  }
  await page.locator(PALETTE_INPUT).focus();
  await page.waitForTimeout(900);
}

async function type(page, text) {
  await page.locator(PALETTE_INPUT).focus();
  await page.keyboard.type(text, { delay: 30 });
  await page.waitForTimeout(1400);
}

const SHOTS = [
  { name: "home-dark", act: open },
  { name: "home-light", colorScheme: "light", act: open },
  {
    name: "query-ai",
    act: async (p) => {
      await open(p);
      await type(p, "ai");
    },
  },
  {
    name: "query-ai-light",
    colorScheme: "light",
    act: async (p) => {
      await open(p);
      await type(p, "ai");
    },
  },
  {
    name: "query-acme-customer",
    act: async (p) => {
      await open(p);
      await type(p, "acme");
    },
  },
  {
    name: "query-acme-shipment",
    act: async (p) => {
      await open(p);
      await type(p, "acme");
      await p.keyboard.press("ArrowDown");
      await p.waitForTimeout(900);
    },
  },
  {
    name: "query-leon-worker",
    act: async (p) => {
      await open(p);
      await type(p, "leon");
    },
  },
  {
    name: "query-bol-document",
    act: async (p) => {
      await open(p);
      await type(p, "bol-88213");
    },
  },
  {
    name: "query-bq",
    act: async (p) => {
      await open(p);
      await type(p, "bq");
    },
  },
  {
    name: "mention-doc",
    act: async (p) => {
      await open(p);
      await type(p, "@doc");
    },
  },
  {
    name: "command-mode-theme",
    act: async (p) => {
      await open(p);
      await type(p, ">theme");
    },
  },
  {
    name: "scope-shipments-browse",
    act: async (p) => {
      await open(p);
      await p.keyboard.press("Tab");
      await p.waitForTimeout(1500);
    },
  },
  {
    name: "page-preview",
    act: async (p) => {
      await open(p);
      await type(p, "shipments");
    },
  },
  {
    name: "actions-menu",
    act: async (p) => {
      await open(p);
      await type(p, "acme");
      await p.keyboard.press("ArrowDown");
      await p.keyboard.press(`${MOD}+k`);
      await p.waitForTimeout(700);
    },
  },
  {
    name: "actions-menu-light",
    colorScheme: "light",
    act: async (p) => {
      await open(p);
      await type(p, "acme");
      await p.keyboard.press("ArrowDown");
      await p.keyboard.press(`${MOD}+k`);
      await p.waitForTimeout(700);
    },
  },
  {
    name: "copy-confirmation",
    act: async (p) => {
      await open(p);
      await type(p, "acme");
      await p.keyboard.press("ArrowDown");
      await p.keyboard.press(`${MOD}+l`);
      await p.waitForTimeout(400);
    },
  },
  {
    name: "subject-questions",
    act: async (p) => {
      await open(p);
      await p.keyboard.press(`${MOD}+i`);
      await p.waitForTimeout(800);
    },
  },
  {
    name: "subject-answer",
    act: async (p) => {
      await open(p);
      await p.keyboard.press(`${MOD}+i`);
      await p.waitForTimeout(500);
      await p.keyboard.press("Enter");
      await p.waitForTimeout(6000);
    },
  },
  {
    name: "nova-working",
    act: async (p) => {
      await open(p);
      await type(p, "any loads at risk today?");
      await p.keyboard.press("Enter");
      await p.waitForTimeout(1500);
    },
  },
  {
    name: "nova-answer",
    act: async (p) => {
      await open(p);
      await type(p, "any loads at risk today?");
      await p.keyboard.press("Enter");
      await p.waitForTimeout(7000);
    },
  },
  {
    name: "nova-answer-light",
    colorScheme: "light",
    act: async (p) => {
      await open(p);
      await type(p, "any loads at risk today?");
      await p.keyboard.press("Enter");
      await p.waitForTimeout(7000);
    },
  },
  {
    name: "nova-hover-chip",
    act: async (p) => {
      await open(p);
      await type(p, "any loads at risk today?");
      await p.keyboard.press("Enter");
      await p.waitForTimeout(7000);
      await p.getByRole("button", { name: "S-240201" }).first().hover();
      await p.waitForTimeout(1800);
    },
  },
  {
    name: "empty-state",
    act: async (p) => {
      await open(p);
      await type(p, "zzqx");
    },
  },
  {
    name: "narrow-860",
    viewport: { width: 860, height: 760 },
    act: async (p) => {
      await open(p);
      await type(p, "acme");
    },
  },
  {
    name: "narrow-700",
    viewport: { width: 700, height: 760 },
    act: async (p) => {
      await open(p);
      await type(p, "acme");
    },
  },
];

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/opt/pw-browsers/chromium",
});
const failures = [];
const { anchor } = await (await fetch(`${MOCK}/__mock/scenario`)).json();

for (const shot of SHOTS) {
  if (ONLY && !ONLY.has(shot.name)) continue;
  const colorScheme = shot.colorScheme ?? "dark";
  const page = await browser.newPage({
    viewport: shot.viewport ?? { width: 1440, height: 900 },
    colorScheme,
  });
  const now = anchor * 1000;
  await page.addInitScript(
    ({ records, theme }) => {
      localStorage.setItem("trenova-ui-theme", theme);
      localStorage.setItem(
        "recent-records-storage",
        JSON.stringify({ state: { recordsByOrganization: { org_mock: records } }, version: 1 }),
      );
    },
    { records: recentRecords(now), theme: colorScheme },
  );
  await page.clock.install({ time: new Date(now) });
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
    await page.waitForTimeout(600);
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
