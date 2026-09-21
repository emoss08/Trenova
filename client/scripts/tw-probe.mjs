#!/usr/bin/env node
/**
 * Compiles the real stylesheet and prints the rule each candidate class emits.
 *
 * Tailwind only generates a class it finds written out in the source, and a custom
 * variant, a theme namespace or an @utility can be subtly wrong without anything
 * failing: the class stays in the markup and simply does nothing. This answers
 * "does this class exist, and what does it resolve to" in a second, without a build.
 *
 * Usage (from client/): node scripts/tw-probe.mjs bleed:px-3 text-warning ui-field
 * A class that prints MISS is not being generated.
 */
import { readFileSync, existsSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, join, resolve } from "node:path";
import { pathToFileURL } from "node:url";

const root = resolve("packages/shared");
const require = createRequire(join(root, "package.json"));
const mod = await import(pathToFileURL(require.resolve("tailwindcss")).href);
const compile = mod.compile ?? mod.default.compile;
const entry = join(root, "src/styles/app.css");

const load = async (id, base) => {
  let p;
  if (id.startsWith(".")) p = resolve(base, id);
  else if (id === "tailwindcss")
    p = require.resolve("tailwindcss/package.json").replace(/package\.json$/, "index.css");
  else {
    try {
      p = require.resolve(id);
    } catch {
      return { base, content: "", path: id };
    }
  }
  if (!existsSync(p) || !p.endsWith(".css")) return { base, content: "", path: id };
  return { base: dirname(p), content: readFileSync(p, "utf8"), path: p };
};

const compiler = await compile(readFileSync(entry, "utf8"), {
  base: dirname(entry),
  loadStylesheet: load,
});
const candidates = process.argv.slice(2);
const css = compiler.build(candidates);

for (const c of candidates) {
  const selector = "." + c.replace(/[^a-zA-Z0-9_-]/g, (ch) => "\\" + ch);
  const at = css.indexOf(selector);
  if (at === -1) {
    console.log(c.padEnd(24), "MISS");
    continue;
  }
  const end = css.indexOf("}", at);
  console.log(c.padEnd(24), css.slice(at, end + 1).replace(/\s+/g, " ").slice(0, 120));
}
