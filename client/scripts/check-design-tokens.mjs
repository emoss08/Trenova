#!/usr/bin/env node
/**
 * Fails the build on styling that bypasses the design tokens.
 *
 * These four rules are the ones the codebase actually broke. Before the token
 * layer was rebuilt there were 1,914 raw palette classes, 961 arbitrary font
 * sizes and 253 hand-written line-height patches across 267 files, and nothing
 * stopped any of them landing. Tokens alone do not hold; the check does.
 *
 * Each rule names the token to reach for instead, because a violation is nearly
 * always someone who could not find the right one.
 *
 * Genuine exceptions exist -- a third-party brand mark has to be its own hex.
 * Put `design-tokens-ignore: <reason>` in a comment on the line or the line
 * above. It is deliberately visible in review rather than silent.
 *
 * Usage: node scripts/check-design-tokens.mjs [--format=github]
 */

import { readFileSync } from "node:fs";
import { join, relative } from "node:path";
import { glob } from "node:fs/promises";

const ROOT = new URL("..", import.meta.url).pathname;
const GITHUB = process.argv.includes("--format=github");

const PALETTE_FAMILIES =
  "red|rose|amber|yellow|orange|emerald|green|lime|blue|sky|cyan|violet|purple|indigo|fuchsia|pink|teal|zinc|gray|slate|neutral|stone";
const PALETTE_PROPS =
  "text|bg|border|ring|fill|stroke|divide|outline|decoration|caret|from|to|via|shadow|accent";

const RULES = [
  {
    id: "raw-palette",
    // `dark:hover:bg-red-500/20` and friends, but not `bg-danger/20`.
    pattern: new RegExp(
      String.raw`\b(?:[a-z-]+:)*(?:${PALETTE_PROPS})-(?:${PALETTE_FAMILIES})-\d{2,3}\b`,
      "g",
    ),
    message: (m) =>
      `\`${m}\` uses the raw Tailwind palette. Use a tone (danger/warning/success/info/neutral), a categorical accent (accent-teal, accent-violet, ...), or a surface/foreground token. See docs/engineering/design-system.md.`,
  },
  {
    id: "arbitrary-font-size",
    // text-[11px]. Hero numerals in rem are allowed; body text in px is not.
    pattern: /\btext-\[\d+(?:\.\d+)?px\]/g,
    message: (m) =>
      `\`${m}\` sets a font size off the scale. Use text-3xs/2xs/xs/sm/base/lg/xl/2xl/3xl -- they carry matching line-heights, which an arbitrary value does not.`,
  },
  {
    id: "hex-in-styling",
    // Only inside className strings and style={{ ... }}; an SVG `fill="#F1511B"`
    // on a brand mark is data, not styling, and is not flagged.
    pattern: /(?:className\s*=\s*\{?["'`][^"'`]*|style\s*=\s*\{\{[^}]*)#[0-9a-fA-F]{3,8}\b/g,
    message: () =>
      `a hex colour in a className or style prop will not follow the theme. Use a token; add one to styles/tokens.css if none fits.`,
  },
  {
    id: "legacy-badge-variant",
    pattern:
      /\bvariant\s*[:=]\s*["'](?:active|inactive|purple|orange|teal|pink|indigo)["']/g,
    message: (m) =>
      `\`${m}\` is a retired Badge variant. Badge takes a tone (neutral/brand/info/success/warning/danger) or a categorical accent (accent-*), crossed with an appearance (subtle/solid/outline).`,
  },
];

const IGNORE = /design-tokens-ignore/;
const SKIP_DIRS = [
  "node_modules",
  "dist",
  "storybook-static",
  "/generated/",
  "/logos/",
  ".turbo",
];

const files = [];
for await (const f of glob(["apps/**/*.{ts,tsx}", "packages/**/*.{ts,tsx}"], { cwd: ROOT })) {
  if (!SKIP_DIRS.some((d) => `/${f}`.includes(d))) files.push(f);
}

let violations = 0;
const byRule = new Map();

for (const rel of files.sort()) {
  const lines = readFileSync(join(ROOT, rel), "utf8").split("\n");
  for (const [i, line] of lines.entries()) {
    // The marker may sit a few lines up, since the reason usually wraps.
    if (lines.slice(Math.max(0, i - 4), i + 1).some((l) => IGNORE.test(l))) continue;
    for (const rule of RULES) {
      rule.pattern.lastIndex = 0;
      const m = rule.pattern.exec(line);
      if (!m) continue;
      violations += 1;
      byRule.set(rule.id, (byRule.get(rule.id) ?? 0) + 1);
      const col = m.index + 1;
      const text = rule.message(m[0]);
      if (GITHUB) {
        console.log(`::error file=client/${rel},line=${i + 1},col=${col}::${text}`);
      } else {
        console.log(`${rel}:${i + 1}:${col}\n  ${rule.id}: ${text}\n`);
      }
    }
  }
}

if (violations === 0) {
  console.log(`design tokens: clean (${files.length} files)`);
  process.exit(0);
}

console.log(`\ndesign tokens: ${violations} violation(s) in ${files.length} files`);
for (const [id, n] of [...byRule].sort((a, b) => b[1] - a[1])) {
  console.log(`  ${id.padEnd(22)} ${n}`);
}
console.log(
  "\nEvery rule above has a token that replaces it; docs/engineering/design-system.md maps them.\n" +
    "For a genuine exception, put `design-tokens-ignore: <reason>` in a comment on or above the line.",
);
process.exit(1);
