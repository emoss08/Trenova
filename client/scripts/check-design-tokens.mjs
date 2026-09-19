#!/usr/bin/env node
/**
 * Fails the build on styling that bypasses the design tokens.
 *
 * These five rules are the ones the codebase actually broke. Before the token
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
import { createRequire } from "node:module";
import { join, relative } from "node:path";
import { pathToFileURL } from "node:url";
import { glob } from "node:fs/promises";

import { auditPalette } from "./palette-audit.mjs";

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
    id: "hand-rolled-focus-ring",
    // focus-visible:relative / focus-visible:z-10 are stacking, not a ring, and
    // are not matched.
    pattern: /\bfocus-(?:visible|within):(?:ring|border|outline)[a-z0-9/.[\]_-]*/g,
    message: (m) =>
      `\`${m}\` builds a focus ring by hand. Use ui-focus-ring, ui-container-focus-ring (focus lands on a child) or ui-inset-focus-ring (no room to bloom outward). For an invalid control add \`aria-invalid:[--ring:var(--ring-danger)]\` rather than a second ring.`,
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

/* ---------------------------------------------------------------------------
   The token layer itself.

   A stylesheet fails in a way no type checker or test sees. A stray comment
   terminator in tokens.css once ended a comment early, and Tailwind silently
   dropped every `@utility` after it: the build stayed green, the classes stayed
   in the markup, and the focus indicator was gone. So the file is compiled here
   and its output checked, rather than trusted.
   --------------------------------------------------------------------------- */

const TOKENS = join(ROOT, "packages/shared/src/styles/tokens.css");

/** Utility names declared outside a comment, so one a broken comment has
 *  swallowed does not count as declared. */
function declaredUtilities(src) {
  const stripped = src.replace(/\/\*[\s\S]*?\*\//g, "");
  return [...stripped.matchAll(/@utility\s+([a-z][a-z0-9-]*)/g)].map((m) => m[1]);
}

async function auditTokenLayer() {
  const src = readFileSync(TOKENS, "utf8");
  const problems = [];

  // `/* */` does not nest, so an extra terminator means a comment ended early
  // and the CSS after it was swallowed.
  if (src.split("/*").length !== src.split("*/").length) {
    problems.push(
      "tokens.css has unbalanced comment markers. A comment terminator inside a comment body ends it early, and every @utility after it stops emitting.",
    );
  }

  // Every themed token must exist in both blocks; one defined only in :root is
  // the bug that left light mode with no --shadow-* at all.
  const stripped = src.replace(/\/\*[\s\S]*?\*\//g, "");
  const darkAt = stripped.indexOf(".dark {");
  if (darkAt !== -1) {
    const names = (block) =>
      new Set([...block.matchAll(/^\s{2}(--[a-z0-9-]+):/gm)].map((m) => m[1]));
    const dark = names(stripped.slice(darkAt));
    // Geometry and the hue plan are the same in both themes by definition; a
    // corner radius does not get darker.
    const themeIndependent =
      /^--(hue-|radius|ring-width|ring-opacity|kpi-|row-|cell-|elevation-flat)/;
    for (const name of names(stripped.slice(0, darkAt))) {
      if (!dark.has(name) && !themeIndependent.test(name)) {
        problems.push(`${name} is defined for light only; a themed token needs a value in .dark too.`);
      }
    }
  }

  // The colours themselves: AA on every pair the design leans on, every value
  // inside sRGB, and the three hues around the brand still apart.
  problems.push(...auditPalette(src));

  // Compile for real and confirm each declared utility reaches the output.
  const names = declaredUtilities(src);
  if (names.length === 0) {
    problems.push("tokens.css declares no @utility at all, which means the file did not parse.");
  } else {
    const require = createRequire(join(ROOT, "packages/shared/package.json"));
    // require.resolve lands on the CJS entry, whose ESM namespace carries the
    // API under `default`; the ESM entry exposes it directly.
    const mod = await import(pathToFileURL(require.resolve("tailwindcss")).href);
    const compile = mod.compile ?? mod.default?.compile;
    if (typeof compile !== "function") {
      problems.push("could not load tailwindcss to compile the token layer.");
      return problems;
    }
    const sheets = {
      // resolve via package.json to land on the package root, not dist/
      tailwindcss: readFileSync(
        require.resolve("tailwindcss/package.json").replace(/package\.json$/, "index.css"),
        "utf8",
      ),
      "./tokens.css": src,
    };
    const compiler = await compile('@import "tailwindcss";\n@import "./tokens.css";', {
      base: "/",
      loadStylesheet: async (id, base) => ({ base, content: sheets[id], path: id }),
    });
    const css = compiler.build(names);
    for (const name of names) {
      if (!new RegExp(`\\.${name}[:{\\s]`).test(css)) {
        problems.push(
          `@utility ${name} is declared in tokens.css but generates no rule. A utility that does not emit removes its styling without failing a build, a test or a type check.`,
        );
      }
    }
  }
  return problems;
}

const tokenProblems = await auditTokenLayer();
for (const problem of tokenProblems) {
  violations += 1;
  byRule.set("token-layer", (byRule.get("token-layer") ?? 0) + 1);
  if (GITHUB) {
    console.log(`::error file=client/packages/shared/src/styles/tokens.css::${problem}`);
  } else {
    console.log(`packages/shared/src/styles/tokens.css\n  token-layer: ${problem}\n`);
  }
}

if (violations === 0) {
  console.log(`design tokens: clean (${files.length} files, ${declaredUtilities(readFileSync(TOKENS, "utf8")).length} utilities verified)`);
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
