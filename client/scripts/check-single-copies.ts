/**
 * Fails when pnpm-lock.yaml resolves a singleton package (see
 * ../singleton-packages.ts) to more than one copy: a different version, or
 * the same version installed twice against different peers. Either puts two
 * copies of its React context in the bundle.
 *
 * Usage: node scripts/check-single-copies.ts [path/to/pnpm-lock.yaml]
 */
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { singletonPackages } from "../singleton-packages.ts";

const lockPath = process.argv[2] ?? fileURLToPath(new URL("../pnpm-lock.yaml", import.meta.url));
const lock = readFileSync(lockPath, "utf8");

// pnpm can write several YAML documents; the project's own resolution is the
// last one that has importers.
const documents = lock.split(/^---$/m).filter((doc) => /^importers:/m.test(doc));
const project = documents.at(-1);
if (!project) {
  console.error("check-single-copies: no importers section in pnpm-lock.yaml");
  process.exit(1);
}

const snapshotsStart = project.search(/^snapshots:$/m);
if (snapshotsStart < 0) {
  console.error("check-single-copies: no snapshots section in pnpm-lock.yaml");
  process.exit(1);
}
// The section runs from the line after "snapshots:" to the next top-level key.
const afterHeader = project.slice(project.indexOf("\n", snapshotsStart) + 1);
const snapshots = afterHeader.split(/^\S/m)[0] ?? "";

// Snapshot keys are two-space indented: "  name@version(peers...):", quoted
// when they contain special characters.
const keys = [...snapshots.matchAll(/^ {2}'?([^\s'][^']*?)'?:/gm)].map((m) => m[1]);

if (keys.length === 0) {
  console.error(
    "check-single-copies: found no snapshot entries; the lockfile format may have changed",
  );
  process.exit(1);
}

let failed = false;
for (const name of singletonPackages) {
  const copies = keys.filter((key) => key.startsWith(`${name}@`));
  if (copies.length > 1) {
    failed = true;
    console.error(`::error::${name} resolves to ${copies.length} copies:`);
    for (const copy of copies) console.error(`  ${copy}`);
  }
}

if (failed) {
  console.error(
    '\nEach of these must resolve once. Reference it as "catalog:" in every package.json ' +
      "(versions live in pnpm-workspace.yaml) and run pnpm install.",
  );
  process.exit(1);
}
console.log(`check-single-copies: ${singletonPackages.length} singleton packages resolve once.`);
