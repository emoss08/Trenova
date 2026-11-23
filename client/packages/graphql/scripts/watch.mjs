// Regenerates the GraphQL client while `bun run dev` runs. graphql-codegen --watch subscribes to
// the highest folder its schema and documents share, which is the repository root: it put an
// inotify watch on every folder (node_modules, Go, Rust, docs; about 40,000), held 400 MB, and
// matched every file event in the repository against its globs. This watches only the files
// codegen and its hooks read, runs one codegen at a time, folds a burst of saves into one run
// and prints a line per run. Turbo runs the cached codegen task before this starts, so there is
// no generation at startup.

import { spawn } from "node:child_process";
import { watch } from "node:fs";
import { basename, dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = resolve(packageRoot, "../../..");
const DEBOUNCE_MS = 200;

const GO_SOURCES = [
  "services/tms/pkg/errortypes/enums.go",
  "services/tms/internal/api/helpers/problemtypes.go",
  "services/tms/internal/modules/identity/domain/permission/resource_gen.go",
  "services/tms/internal/modules/identity/domain/permission/operations.go",
  "services/tms/internal/modules/identity/domain/permission/sensitivity.go",
  "services/tms/internal/modules/identity/domain/recordkind/recordid.go",
  "services/tms/internal/modules/identity/domain/recordkind/recordkind.go",
  "services/tms/internal/modules/agent/domain/agent/enums.go",
  "services/tms/internal/modules/agent/domain/agent/subjectrecord.go",
  "services/tms/internal/modules/agent/app/assistantservice/assistantthreads/mentions.go",
  "services/tms/internal/modules/agent/ports/agenttoolports/toolnamespace.go",
];

const targets = [
  {
    dir: join(repoRoot, "services/tms/internal/api/graphql/schema"),
    matches: (name) => name.endsWith(".graphqls"),
  },
  {
    dir: join(packageRoot, "src/operations"),
    recursive: true,
    matches: (name) => name.endsWith(".graphql"),
  },
  { dir: packageRoot, matches: (name) => name === "codegen.ts" },
  {
    dir: join(packageRoot, "scripts"),
    matches: (name) => name.endsWith(".mjs") && !name.endsWith(".test.mjs") && name !== "watch.mjs",
  },
  ...Object.entries(Object.groupBy(GO_SOURCES, (file) => dirname(join(repoRoot, file)))).map(
    ([dir, files]) => {
      const names = new Set(files.map((file) => basename(file)));
      return { dir, matches: (name) => names.has(name) };
    },
  ),
];

let timer;
let running;
let pending = new Set();

function log(message) {
  console.log(`graphql: ${message}`);
}

function schedule(path) {
  pending.add(relative(repoRoot, path));
  clearTimeout(timer);
  timer = setTimeout(run, DEBOUNCE_MS);
}

function run() {
  if (running !== undefined || pending.size === 0) return;
  const changed = [...pending];
  pending = new Set();
  const started = performance.now();
  log(`generating (${changed.length === 1 ? changed[0] : `${changed.length} files changed`})`);

  const output = [];
  running = spawn("bun", ["run", "codegen"], {
    cwd: packageRoot,
    stdio: ["ignore", "pipe", "pipe"],
  });
  running.stdout.on("data", (chunk) => output.push(chunk));
  running.stderr.on("data", (chunk) => output.push(chunk));
  running.on("close", (code, signal) => {
    running = undefined;
    const seconds = ((performance.now() - started) / 1000).toFixed(1);
    if (code === 0) {
      log(`generated in ${seconds}s`);
    } else if (signal === null) {
      process.stdout.write(Buffer.concat(output));
      log(`codegen failed after ${seconds}s; fix the error and save again`);
    }
    if (pending.size > 0) run();
  });
}

const watchers = targets.map(({ dir, recursive = false, matches }) =>
  watch(dir, { recursive }, (_event, filename) => {
    if (filename && matches(basename(filename))) schedule(join(dir, filename));
  }),
);

function stop() {
  clearTimeout(timer);
  for (const watcher of watchers) watcher.close();
  running?.kill("SIGTERM");
  process.exit(0);
}

process.on("SIGINT", stop);
process.on("SIGTERM", stop);
log(`watching ${targets.length} folders for schema, operation and generator changes`);
