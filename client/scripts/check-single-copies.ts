/**
 * Fails when the client resolves a singleton package (see ../singleton-packages.ts) to more
 * than one copy: a different version in bun.lock, or the same version installed twice
 * against different peers in the isolated store. Either puts two copies of its React
 * context in the bundle.
 *
 * Usage: bun scripts/check-single-copies.ts [path/to/bun.lock]
 */
import { existsSync, readdirSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { singletonPackages } from "../singleton-packages.ts";

type LockEntry = [ident: string, ...rest: unknown[]];
type Lock = { packages?: Record<string, LockEntry> };

const lockPath = process.argv[2] ?? fileURLToPath(new URL("../bun.lock", import.meta.url));
const lock = Bun.JSONC.parse(readFileSync(lockPath, "utf8")) as Lock;

const idents = Object.values(lock.packages ?? {}).map((entry) => entry[0]);
if (idents.length === 0) {
  console.error(
    "check-single-copies: no packages in bun.lock; the lockfile format may have changed",
  );
  process.exit(1);
}

// The isolated linker keeps one folder per installed copy in node_modules/.bun, named
// `<name with / as +>@<version>` and suffixed `+<peer hash>` when the package has peers.
// It exists only after an install; the lockfile check above still runs without it.
const store = join(dirname(lockPath), "node_modules", ".bun");
const storeFolders = existsSync(store) ? readdirSync(store) : [];

const copiesOf = (name: string) => {
  const versions = new Set(idents.filter((ident) => ident.startsWith(`${name}@`)));
  const folder = `${name.replace("/", "+")}@`;
  const installed = storeFolders.filter((entry) => entry.startsWith(folder));
  return installed.length > versions.size ? installed : [...versions];
};

let failed = false;
for (const name of singletonPackages) {
  const copies = copiesOf(name);
  if (copies.length > 1) {
    failed = true;
    console.error(`::error::${name} resolves to ${copies.length} copies:`);
    for (const copy of copies) console.error(`  ${copy}`);
  }
}

if (failed) {
  console.error(
    '\nEach of these must resolve once. Reference it as "catalog:" in every package.json ' +
      "(versions live in the workspaces catalog of client/package.json) and run bun install.",
  );
  process.exit(1);
}
console.log(`check-single-copies: ${singletonPackages.length} singleton packages resolve once.`);
