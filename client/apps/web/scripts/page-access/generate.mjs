#!/usr/bin/env node
/**
 * Writes the screens half of the page access manifest: for every page a
 * signed-in person can open, what its route demands and what its code runs —
 * the GraphQL operations its modules import and the REST endpoints they call.
 *
 * The route tree is the product guide's reading of the router
 * (`../product-guide/sources.mjs`). A screen's code is every module reachable
 * from its page module, through static and dynamic imports, short of the app
 * shell: what `main.tsx` loads before any route (the navigation, the command
 * palette, the assistant) runs on every screen and belongs to none.
 *
 * `go run ./internal/api/graphql/gqlext/refgen` (from services/tms) then
 * reads this file and the operations to write manifest_gen.json, which says
 * which kinds each screen reads, shows through reference views and declares
 * with @requiresRead. See docs/engineering/reference-views.md.
 *
 * Usage:
 *   node scripts/page-access/generate.mjs           write screens_gen.json
 *   node scripts/page-access/generate.mjs --check   fail if it is stale
 */

import { existsSync, readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { join, relative } from "node:path";
import process from "node:process";

import {
  loadMembers,
  readNavigation,
  readPageHeader,
  readRoutes,
  sourcePaths,
} from "../product-guide/sources.mjs";
import { ModuleGraph, graphPaths } from "./imports.mjs";

const WEB_ROOT = new URL("../..", import.meta.url).pathname;
const REPO_ROOT = join(WEB_ROOT, "..", "..", "..");
const OPERATIONS_DIR = join(REPO_ROOT, "client", "packages", "graphql", "src", "operations");
export const OUTPUT = join(REPO_ROOT, "services", "tms", "pkg", "pageaccess", "screens_gen.json");

const OPERATION = /^\s*(query|mutation|subscription)\s+([A-Za-z_][A-Za-z0-9_]*)/gm;

const nameKey = (name) => name.replace(/[^A-Za-z0-9]/g, "").toLowerCase();

function graphqlFiles(directory) {
  const files = [];
  for (const entry of readdirSync(directory)) {
    const full = join(directory, entry);
    if (statSync(full).isDirectory()) {
      files.push(...graphqlFiles(full));
    } else if (entry.endsWith(".graphql")) {
      files.push(full);
    }
  }
  return files;
}

/** Each generated document's name, by its letters, to the operation it runs. */
function operationsByDocument() {
  const operations = new Map();
  for (const file of graphqlFiles(OPERATIONS_DIR)) {
    for (const match of readFileSync(file, "utf8").matchAll(OPERATION)) {
      operations.set(nameKey(`${match[2]}Document`), match[2]);
    }
  }
  return operations;
}

function normalizePath(path) {
  return path === "/" ? path : path.replace(/\/+$/, "");
}

function navigationLabels(navigation) {
  const labels = new Map();
  for (const module of navigation.modules) {
    for (const entry of module.navigation ?? []) {
      for (const item of Array.isArray(entry.items) ? entry.items : [entry]) {
        if (typeof item.path === "string" && typeof item.label === "string") {
          labels.set(normalizePath(item.path), item.label);
        }
      }
    }
  }
  for (const link of navigation.adminLinks) {
    if (typeof link.href === "string" && typeof link.title === "string") {
      const path = normalizePath(link.href);
      if (!labels.has(path)) labels.set(path, link.title);
    }
  }
  return labels;
}

function build() {
  const errors = [];
  const paths = sourcePaths(WEB_ROOT);
  const members = loadMembers(paths);
  const routes = readRoutes(paths, members);
  const labels = navigationLabels(readNavigation(paths, members));
  const graph = new ModuleGraph(graphPaths(WEB_ROOT));
  const documents = operationsByDocument();

  // The shell: everything main.tsx loads, except the pages the router loads
  // lazily, which are the screens.
  const router = graph.paths.router;
  const shell = graph.closure([graph.paths.main], {
    follow: (from, _target, dynamic) => !(dynamic && from === router),
  });

  const screens = [];
  for (const route of [...routes.values()].sort((a, b) => a.path.localeCompare(b.path))) {
    const files = graph.closure([route.file], { stop: shell });
    const uses = graph.uses(files, [route.file]);
    const operations = new Set();
    for (const document of uses.documents) {
      const operation = documents.get(nameKey(document));
      if (operation === undefined) {
        errors.push(`${relative(REPO_ROOT, route.file)}: ${document} is no operation's document`);
        continue;
      }
      operations.add(operation);
    }
    const path = normalizePath(route.path);
    const name = labels.get(path) ?? readPageHeader(route.file, route.exportName)?.title ?? "";
    screens.push({
      path,
      name,
      requires: route.requires,
      operations: [...operations].sort(),
      rest: uses.rest
        .map(({ method, path: endpoint }) => ({ method, path: endpoint }))
        .sort((a, b) => `${a.path} ${a.method}`.localeCompare(`${b.path} ${b.method}`)),
    });
  }

  return { errors, screens };
}

const CHECK = process.argv.includes("--check");
const { errors, screens } = build();
if (errors.length > 0) {
  for (const error of errors) console.error(`page access: ${error}`);
  process.exit(1);
}

const serialized = `${JSON.stringify({ screens }, null, 2)}\n`;
if (CHECK) {
  const current = existsSync(OUTPUT) ? readFileSync(OUTPUT, "utf8") : "";
  if (current !== serialized) {
    console.error(
      "page access: screens_gen.json is stale. Run `pnpm --filter @trenova/web access:generate`, " +
        "then `go run ./internal/api/graphql/gqlext/refgen` from services/tms, and commit both.",
    );
    process.exit(1);
  }
  console.log(`page access: up to date (${screens.length} screens)`);
} else {
  writeFileSync(OUTPUT, serialized);
  console.log(
    `page access: wrote ${screens.length} screens to ${relative(REPO_ROOT, OUTPUT)}; ` +
      "now run `go run ./internal/api/graphql/gqlext/refgen` from services/tms",
  );
}
