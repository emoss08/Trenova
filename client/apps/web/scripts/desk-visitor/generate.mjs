#!/usr/bin/env node
/**
 * Writes the desk visitor's keyframes from the pose in
 * src/components/assistant/voice/desk-visitor-motion.ts.
 *
 * Usage:
 *   node scripts/desk-visitor/generate.mjs           write the CSS
 *   node scripts/desk-visitor/generate.mjs --check   fail if it is stale
 */

import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

import { deskVisitorKeyframes } from "../../src/components/assistant/voice/desk-visitor-motion.ts";

const here = dirname(fileURLToPath(import.meta.url));
const target = join(here, "../../src/components/assistant/voice/desk-visitor.css");
const css = deskVisitorKeyframes();

if (process.argv.includes("--check")) {
  let current = "";
  try {
    current = readFileSync(target, "utf8");
  } catch {
    current = "";
  }
  if (current !== css) {
    console.error(
      "desk-visitor.css is stale: run `pnpm --filter @trenova/web desk-visitor:generate`",
    );
    process.exit(1);
  }
  console.log("desk-visitor.css is current");
} else {
  writeFileSync(target, css);
  console.log(`desk-visitor.css written (${css.length} bytes)`);
}
