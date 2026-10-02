// Screenshots the Desk stories from a running Storybook (pnpm storybook --no-open).
//   node scripts/desk-shots.mjs <outDir> [theme=dark|light] [width=1920] [height=1080] [story...]
import { chromium } from "playwright";
import { mkdir } from "node:fs/promises";

const [outDir = "./shots", theme = "dark", width = "1920", height = "1080", ...only] = process.argv.slice(2);
const base = process.env.STORYBOOK_URL ?? "http://localhost:6006";
const stories = [
  "home",
  "conversation",
  "conversation-rail-folded",
  "conversation-no-workspace",
  "conversation-decisions-tab",
  "conversation-activity-tab",
  "conversation-composer",
  "decisions",
  "watchtower",
].filter((s) => only.length === 0 || only.includes(s));

await mkdir(outDir, { recursive: true });
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || undefined });
const page = await browser.newPage({ viewport: { width: Number(width), height: Number(height) }, colorScheme: theme === "dark" ? "dark" : "light" });
page.on("console", (msg) => {
  if (msg.type() === "warning" || msg.type() === "error") console.log(`[browser:${msg.type()}]`, msg.text().slice(0, 300));
});
for (const story of stories) {
  const url = `${base}/iframe.html?id=desk-desk--${story}&viewMode=story&globals=theme:${theme}`;
  await page.goto(url, { waitUntil: "networkidle" });
  await page.waitForTimeout(1200);
  await page.screenshot({ path: `${outDir}/${story}-${theme}.png` });
  console.log("shot", story, theme);
}
await browser.close();
