// What a first visit downloads before anything renders: the entry script, every module it
// preloads, and the stylesheets the page links. A change that pulls a feature into that set
// (the Desk once arrived with every page) shows here as a number, and fails the build once
// it passes the budget in bundle-budget.json. Raising the budget is a decision made in the
// same change, where a reviewer can see it.

import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { gzipSync } from "node:zlib";
import { startupFiles } from "../src/lib/startup-files.ts";

type StartupFile = { file: string; bytes: number; gzip: number };
type Measured = { files: number; bytes: number; gzip: number; largest: StartupFile[] };
export type Measurement = { scripts: Measured; styles: Measured };
type Limit = { files: number; gzipBytes: number };
export type Budget = { scripts: Limit; styles: Limit };
const KINDS = ["scripts", "styles"] as const;

const gzipBytes = (buffer: Buffer) => gzipSync(buffer, { level: 6 }).length;

/** measure reads each startup file from the build and totals its raw and gzipped size. */
export async function measure(distDir: string): Promise<Measurement> {
  const html = await readFile(join(distDir, "index.html"), "utf8");
  const { scripts, styles } = startupFiles(html);
  const read = async (url: string): Promise<StartupFile> => {
    const buffer = await readFile(join(distDir, url));
    return { file: url, bytes: buffer.length, gzip: gzipBytes(buffer) };
  };
  const total = (files: StartupFile[]): Measured => ({
    files: files.length,
    bytes: files.reduce((sum, file) => sum + file.bytes, 0),
    gzip: files.reduce((sum, file) => sum + file.gzip, 0),
    largest: [...files].sort((a, b) => b.gzip - a.gzip),
  });
  return {
    scripts: total(await Promise.all(scripts.map(read))),
    styles: total(await Promise.all(styles.map(read))),
  };
}

/** overBudget returns a line for every measured figure that exceeds its budget. */
export function overBudget(measured: Measurement, budget: Budget): string[] {
  const failures: string[] = [];
  for (const kind of KINDS) {
    const limits = budget[kind];
    const actual = measured[kind];
    if (actual.gzip > limits.gzipBytes) {
      failures.push(`${kind}: ${kb(actual.gzip)} gzip is over the ${kb(limits.gzipBytes)} budget`);
    }
    if (actual.files > limits.files) {
      failures.push(`${kind}: ${actual.files} files is over the budget of ${limits.files}`);
    }
  }
  return failures;
}

const kb = (bytes: number) => `${(bytes / 1024).toFixed(1)} kB`;

function report(measured: Measurement, budget: Budget): string {
  const lines: string[] = [];
  for (const kind of KINDS) {
    const actual = measured[kind];
    const limits = budget[kind];
    lines.push(
      `${kind.padEnd(8)} ${String(actual.files).padStart(3)} files ` +
        `${kb(actual.bytes).padStart(10)} raw ${kb(actual.gzip).padStart(10)} gzip ` +
        `(budget ${limits.files} files, ${kb(limits.gzipBytes)} gzip)`,
    );
  }
  return lines.join("\n");
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const appDir = join(fileURLToPath(import.meta.url), "../..");
  const budget = JSON.parse(await readFile(join(appDir, "bundle-budget.json"), "utf8")) as Budget;
  const measured = await measure(join(appDir, "dist"));
  console.log("Startup download (before first render):");
  console.log(report(measured, budget));

  const failures = overBudget(measured, budget);
  if (failures.length > 0) {
    console.error(`\nOver budget:\n  ${failures.join("\n  ")}`);
    console.error("\nLargest startup files:");
    for (const file of [...measured.scripts.largest, ...measured.styles.largest]
      .sort((a, b) => b.gzip - a.gzip)
      .slice(0, 8)) {
      console.error(`  ${kb(file.gzip).padStart(10)} gzip  ${file.file}`);
    }
    console.error(
      "\nMove what the first page does not need behind a lazy import, or raise the budget in " +
        "apps/web/bundle-budget.json in this change if the growth is intended.",
    );
    process.exitCode = 1;
  }
}
