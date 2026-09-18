import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";

import { compile } from "tailwindcss";
import { describe, expect, it } from "vitest";

/* The token layer, compiled for real.
 *
 * A stylesheet fails quietly in a way TypeScript and the test suite cannot see:
 * a stray comment terminator ended a comment early here once, and every
 * `@utility` after it stopped generating a rule. Nothing failed — the build was
 * green, the classes were still in the markup, and the focus indicator was
 * simply gone. These tests compile the real file and assert the output.
 */

const STYLES_DIR = join(dirname(fileURLToPath(import.meta.url)), "..");
const TOKENS = join(STYLES_DIR, "tokens.css");

/* tailwindcss/compile resolves no imports on its own, so the test supplies a
   loader: bare `tailwindcss` resolves to the package's own entry, everything
   else is read relative to the importing sheet. */
async function loadStylesheet(id: string, base: string) {
  const path = id.startsWith(".")
    ? join(base, id)
    : createRequire(import.meta.url).resolve(`${id}${id.includes(".css") ? "" : "/index.css"}`);
  return { base: dirname(path), content: readFileSync(path, "utf8"), path };
}

async function buildTokens(candidates: string[]): Promise<string> {
  const compiler = await compile(`@import "tailwindcss";\n@import "./tokens.css";`, {
    base: STYLES_DIR,
    loadStylesheet,
  });
  return compiler.build(candidates);
}

function declaredUtilities(): string[] {
  // Read from the comment-stripped source, so a name that has been commented
  // out (or swallowed by a broken comment) is not counted as declared.
  const stripped = readFileSync(TOKENS, "utf8").replace(/\/\*[\s\S]*?\*\//g, "");
  return [...stripped.matchAll(/@utility\s+([a-z][a-z0-9-]*)/g)].map((m) => m[1]);
}

describe("tokens.css", () => {
  it("has balanced comment markers", () => {
    const src = readFileSync(TOKENS, "utf8");
    // `/* */` does not nest, so an extra terminator means a comment ended early
    // and live CSS after it was swallowed.
    expect(src.split("/*").length).toBe(src.split("*/").length);
  });

  it("declares the three focus utilities", () => {
    expect(declaredUtilities()).toEqual(
      expect.arrayContaining(["ui-focus-ring", "ui-container-focus-ring", "ui-inset-focus-ring"]),
    );
  });

  it("emits a rule for every @utility it declares", async () => {
    const names = declaredUtilities();
    expect(names.length).toBeGreaterThan(0);

    const css = await buildTokens(names);
    const missing = names.filter((n) => !new RegExp(`\\.${n}[:{\\s]`).test(css));

    expect(missing).toEqual([]);
  });

  it("gives the focus ring a visible box-shadow", async () => {
    const css = await buildTokens(["ui-focus-ring"]);
    expect(css).toMatch(/\.ui-focus-ring:focus-visible\s*\{[^}]*box-shadow/);
  });

  it("defines every themed token in both light and dark", () => {
    const src = readFileSync(TOKENS, "utf8").replace(/\/\*[\s\S]*?\*\//g, "");
    const dark = src.slice(src.indexOf(".dark {"));
    const light = src.slice(0, src.indexOf(".dark {"));

    const names = (block: string) =>
      new Set([...block.matchAll(/^\s{2}(--[a-z0-9-]+):/gm)].map((m) => m[1]));

    const lightOnly = [...names(light)].filter((n) => !names(dark).has(n));

    // The aliases and the hue primitives are theme-independent by design; a
    // colour that varies by theme must appear in both blocks, which is the bug
    // that left light mode with no --shadow-* at all.
    const themeIndependent = /^--(hue-|radius$|ring-width|ring-opacity|kpi-|elevation-flat)/;
    expect(lightOnly.filter((n) => !themeIndependent.test(n))).toEqual([]);
  });
});
