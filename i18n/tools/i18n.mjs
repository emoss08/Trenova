#!/usr/bin/env node
// i18n.mjs is the single entry point for the translation pipeline.
//
//   report                 inventory of source strings, per locale and per area
//   sync                   rebuild i18n/messages.en.json from source; create locale files;
//                          refresh an edition overlay's own catalog when it is in the tree
//   pending <locale>       print strings still needing translation (--area, --limit)
//   merge <locale> <file>  merge a batch of translations into that locale's catalog
//   emit                   write the per-scope, per-bundle runtime catalogs the apps load
//   check                  CI gate: fail on missing or orphaned entries, or stale runtime catalogs
//   codemod <dir> [--write]  wrap user-facing literals in t() under <dir>
//
// Both extractors feed one merged catalog keyed by the English source string, so a message
// written once in Go and again in React is translated once.
import { execFile } from "node:child_process";
import { mkdir, readdir, readFile, rm, writeFile } from "node:fs/promises";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";
import {
  bundleFileName,
  catalogBundle,
  compareBundles,
  CORE_BUNDLE,
  WEB_BUNDLE,
} from "./bundles.mjs";
import { runCodemod } from "./codemod.mjs";
import { extractTemplates } from "./extract-templates.mjs";
import { extractTypeScript } from "./extract-ts.mjs";
import { reject } from "./filter.mjs";

const execFileAsync = promisify(execFile);
const toolsDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(toolsDir, "../..");

async function extractGo() {
  const { stdout } = await execFileAsync(
    "go",
    ["run", "./shared/cmd/i18n-extract", "-root", "."],
    { cwd: repoRoot, maxBuffer: 64 * 1024 * 1024 },
  );
  const raw = JSON.parse(stdout).entries;

  const entries = [];
  const rejected = [];
  for (const item of raw) {
    // Every Go entry comes from a position the extractor knows is user-facing.
    const reason = reject(item.message, { positional: true });
    if (reason !== null) {
      rejected.push({ ...item, reason });
      continue;
    }
    entries.push({
      message: item.message.trim().replace(/\s+/g, " "),
      file: item.file,
      line: item.line,
      kind: item.callee === "Error" ? "validation-rule" : "error",
      area: goArea(item.file),
      scope: "go",
    });
  }
  return { entries, rejected };
}

// goArea buckets Go messages by the domain or service they belong to, mirroring how the
// frontend buckets by route, so one feature can be translated across both tiers together.
function goArea(file) {
  const domain = /internal\/core\/domain\/([^/]+)\//.exec(file);
  if (domain) return `domain/${domain[1]}`;
  const service = /internal\/core\/services\/([^/]+)\//.exec(file);
  if (service) return `service/${service[1]}`;
  const api = /internal\/api\/([^/]+)\//.exec(file);
  if (api) return `api/${api[1]}`;
  return "go/other";
}

function groupCount(items, key) {
  const counts = new Map();
  for (const item of items) {
    const k = typeof key === "function" ? key(item) : item[key];
    counts.set(k, (counts.get(k) ?? 0) + 1);
  }
  return [...counts.entries()].sort((a, b) => b[1] - a[1]);
}

async function loadLocales() {
  const raw = await readFile(join(repoRoot, "i18n/locales.json"), "utf8");
  return JSON.parse(raw);
}

async function report({ showRejected }) {
  const [go, ts, templates] = await Promise.all([
    extractGo(),
    extractTypeScript(repoRoot),
    extractTemplates(repoRoot),
  ]);
  go.entries.push(...templates.entries);

  if (ts.errors.length > 0) {
    console.error(`\nParse failures (${ts.errors.length}):`);
    for (const e of ts.errors.slice(0, 10)) console.error(`  ${e.file}: ${e.message}`);
  }

  const all = [...go.entries, ...ts.entries];
  const unique = new Map();
  for (const entry of all) {
    let record = unique.get(entry.message);
    if (record === undefined) {
      record = { message: entry.message, sites: 0, scopes: new Set(), areas: new Set() };
      unique.set(entry.message, record);
    }
    record.sites += 1;
    record.scopes.add(entry.scope);
    record.areas.add(entry.area);
  }

  const shared = [...unique.values()].filter((r) => r.scopes.size > 1);
  const goOnly = [...unique.values()].filter((r) => r.scopes.has("go") && r.scopes.size === 1);
  const tsOnly = [...unique.values()].filter((r) => r.scopes.has("ts") && r.scopes.size === 1);

  const { locales } = await loadLocales();
  const targets = locales.filter((l) => l !== "en");

  console.log("\n=== Trenova i18n inventory ===\n");
  console.log(`  Go message sites          ${go.entries.length}`);
  console.log(`  TypeScript message sites  ${ts.entries.length}`);
  console.log(`  ---------------------------------`);
  console.log(`  Total sites               ${all.length}`);
  console.log(`  UNIQUE STRINGS            ${unique.size}   <- the translation workload`);
  console.log(`    backend only            ${goOnly.length}`);
  console.log(`    frontend only           ${tsOnly.length}`);
  console.log(`    shared by both          ${shared.length}`);
  console.log(`  Dedup saving              ${all.length - unique.size} repeat sites collapsed`);
  console.log(
    `\n  To translate: ${unique.size} x ${targets.length} locales = ` +
      `${unique.size * targets.length} entries (${targets.join(", ")})`,
  );

  console.log("\n--- TypeScript sites by kind ---");
  for (const [kind, n] of groupCount(ts.entries, "kind")) console.log(`  ${String(n).padStart(6)}  ${kind}`);

  console.log("\n--- Go sites by kind ---");
  for (const [kind, n] of groupCount(go.entries, "kind")) console.log(`  ${String(n).padStart(6)}  ${kind}`);

  console.log("\n--- Top 25 feature areas by unique strings ---");
  const byArea = new Map();
  for (const record of unique.values()) {
    for (const area of record.areas) {
      if (!byArea.has(area)) byArea.set(area, new Set());
      byArea.get(area).add(record.message);
    }
  }
  const areaRows = [...byArea.entries()].sort((a, b) => b[1].size - a[1].size);
  for (const [area, set] of areaRows.slice(0, 25)) {
    console.log(`  ${String(set.size).padStart(6)}  ${area}`);
  }
  console.log(`  (${areaRows.length} areas total)`);

  const allRejected = [...go.rejected, ...ts.rejected];
  console.log(`\n--- Filtered out: ${allRejected.length} literals ---`);
  for (const [reason, n] of groupCount(allRejected, "reason")) {
    console.log(`  ${String(n).padStart(6)}  ${reason}`);
  }

  if (showRejected) {
    console.log("\n--- Filter audit: samples per reason ---");
    const byReason = new Map();
    for (const item of allRejected) {
      if (!byReason.has(item.reason)) byReason.set(item.reason, []);
      byReason.get(item.reason).push(item);
    }
    for (const [reason, items] of byReason) {
      console.log(`\n  [${reason}] ${items.length}`);
      const seen = new Set();
      for (const item of items) {
        if (seen.size >= 12) break;
        if (seen.has(item.value)) continue;
        seen.add(item.value);
        console.log(`    ${JSON.stringify(item.value).slice(0, 90)}  (${item.file}:${item.line})`);
      }
    }
  }
  console.log();
}


const CATALOG_DIR = join(repoRoot, "i18n");

function sortedObject(obj) {
  return Object.fromEntries(Object.keys(obj).sort((a, b) => (a < b ? -1 : a > b ? 1 : 0)).map((k) => [k, obj[k]]));
}

async function writeJSON(path, value) {
  await mkdir(dirname(path), { recursive: true });
  await writeFile(path, `${JSON.stringify(value, null, 2)}\n`, "utf8");
}

async function readJSONIfExists(path, fallback) {
  try {
    return JSON.parse(await readFile(path, "utf8"));
  } catch (err) {
    if (err.code === "ENOENT") return fallback;
    throw err;
  }
}

function localePath(locale) {
  return join(CATALOG_DIR, `messages.${locale}.json`);
}

// buildSource merges both extractors into one inventory keyed by the English string. Areas
// and scope travel with each entry so translation can be batched by screen and so the emit
// step can keep backend messages out of the browser bundle.
async function buildSource() {
  const [go, ts, templates] = await Promise.all([
    extractGo(),
    extractTypeScript(repoRoot),
    extractTemplates(repoRoot),
  ]);
  go.entries.push(...templates.entries);
  const editionGo = new Map();
  const publicGo = [];
  for (const entry of go.entries) {
    const edition = goEditionFor(entry.file);
    if (edition === null) {
      publicGo.push(entry);
      continue;
    }
    if (!editionGo.has(edition.dir)) editionGo.set(edition.dir, []);
    editionGo.get(edition.dir).push(entry);
  }
  const source = {};
  for (const entry of [...publicGo, ...ts.entries]) {
    let record = source[entry.message];
    if (record === undefined) {
      record = { sites: 0, scope: [], areas: [] };
      source[entry.message] = record;
    }
    record.sites += 1;
    if (!record.scope.includes(entry.scope)) record.scope.push(entry.scope);
    if (!record.areas.includes(entry.area)) record.areas.push(entry.area);
  }
  for (const record of Object.values(source)) {
    record.scope.sort();
    record.areas.sort();
  }
  return { source: sortedObject(source), parseErrors: ts.errors, editionGo };
}

// An edition overlay (docs/engineering/editions.md) keeps the strings that exist only in its
// own code in its own catalog, beside its source, so the public catalogs neither carry them
// nor lose their translations when the overlay is not in the tree. A string the public app
// also uses stays public — the runtime layers the edition catalog over the app's — unless
// the public app only ships it with one route folder (see publicCovers).
// A Go edition is extracted with the rest of the Go tree and split off by path: its strings
// never reach the public catalogs, and emit writes it runtime catalogs it embeds itself.
const EDITIONS = [
  { dir: "client/packages/cloud", scope: "ts", marker: "package.json" },
  { dir: "services/tms/internal/cloud", scope: "go", marker: "module.go", runtimeDir: "i18n/catalogs" },
];

function goEditionFor(file) {
  const rel = file.replace(/^\.\//, "");
  return EDITIONS.find((e) => e.scope === "go" && rel.startsWith(`${e.dir}/`)) ?? null;
}

async function presentEditions() {
  const present = [];
  for (const edition of EDITIONS) {
    try {
      await readFile(join(repoRoot, edition.dir, edition.marker), "utf8");
      present.push(edition);
    } catch (err) {
      if (err.code !== "ENOENT") throw err;
    }
  }
  return present;
}

// publicCovers reports whether the public runtime catalog already carries a string wherever
// the edition renders it. The Go catalog is whole, so any public Go string is covered. The
// web app's catalog is split, and an edition page only has the startup bundles for certain:
// a string the public app keeps in one route folder's bundle goes in the edition's own
// catalog as well, or the edition page would render it in English.
function publicCovers(edition, record) {
  if (record === undefined || !record.scope.includes(edition.scope)) return false;
  if (edition.scope !== "ts") return true;
  const bundle = catalogBundle(record.areas);
  return bundle === CORE_BUNDLE || bundle === WEB_BUNDLE;
}

async function buildEditionSource(edition, publicSource, editionGo) {
  let entries;
  let parseErrors = [];
  if (edition.scope === "go") {
    entries = editionGo.get(edition.dir) ?? [];
  } else {
    const ts = await extractTypeScript(repoRoot, [join(edition.dir, "src")]);
    entries = ts.entries;
    parseErrors = ts.errors;
  }
  const source = {};
  for (const entry of entries) {
    if (publicCovers(edition, publicSource[entry.message])) continue;
    let record = source[entry.message];
    if (record === undefined) {
      record = { sites: 0, scope: [edition.scope], areas: [] };
      source[entry.message] = record;
    }
    record.sites += 1;
    if (!record.areas.includes(entry.area)) record.areas.push(entry.area);
  }
  for (const record of Object.values(source)) record.areas.sort();
  return { source: sortedObject(source), parseErrors };
}

function editionLocalePath(dir, locale) {
  return join(repoRoot, dir, "i18n", `messages.${locale}.json`);
}

// syncEdition runs before the public catalogs are pruned, so a string that moved from the
// app into the overlay takes its existing translation with it instead of losing it.
async function syncEdition(edition, publicSource, editionGo) {
  const { dir } = edition;
  const { source, parseErrors } = await buildEditionSource(edition, publicSource, editionGo);
  if (parseErrors.length > 0) {
    console.error(`i18n: ${dir}: ${parseErrors.length} files failed to parse`);
    process.exit(1);
  }

  await writeJSON(editionLocalePath(dir, "en"), source);
  const keys = Object.keys(source);
  console.log(`i18n: ${keys.length} edition strings -> ${dir}/i18n/messages.en.json`);

  const { locales } = await loadLocales();
  for (const locale of locales.filter((l) => l !== "en")) {
    const existing = await readJSONIfExists(editionLocalePath(dir, locale), {});
    const inherited = await readJSONIfExists(localePath(locale), {});
    const kept = {};
    for (const key of keys) {
      if (key in existing) kept[key] = existing[key];
      else if (key in inherited) kept[key] = inherited[key];
    }
    await writeJSON(editionLocalePath(dir, locale), sortedObject(kept));
    const missing = keys.filter((k) => !(k in kept)).length;
    console.log(
      `  ${locale.padEnd(6)} ${String(Object.keys(kept).length).padStart(6)} translated, ` +
        `${String(missing).padStart(6)} missing`,
    );
  }
}

async function checkEdition(edition, publicSource, editionGo) {
  const { dir } = edition;
  const { source, parseErrors } = await buildEditionSource(edition, publicSource, editionGo);
  if (parseErrors.length > 0) {
    console.error(`i18n: ${dir}: ${parseErrors.length} files failed to parse`);
    return false;
  }

  const committed = await readJSONIfExists(editionLocalePath(dir, "en"), null);
  if (committed === null || JSON.stringify(committed) !== JSON.stringify(source)) {
    console.error(`i18n: ${dir}/i18n/messages.en.json is out of date — run \`task i18n\` and commit`);
    return false;
  }

  const { locales } = await loadLocales();
  let ok = true;
  for (const locale of locales.filter((l) => l !== "en")) {
    const translated = await readJSONIfExists(editionLocalePath(dir, locale), {});
    const missing = Object.keys(source).filter((k) => !(k in translated));
    const orphans = Object.keys(translated).filter((k) => !(k in source));
    if (missing.length > 0 || orphans.length > 0) {
      ok = false;
      console.error(`i18n: ${dir} ${locale} — ${missing.length} missing, ${orphans.length} orphaned`);
      for (const key of missing.slice(0, 5)) console.error(`    missing: ${JSON.stringify(key)}`);
    }
  }
  return ok;
}

async function sync() {
  const { source, parseErrors, editionGo } = await buildSource();
  if (parseErrors.length > 0) {
    console.error(`i18n: ${parseErrors.length} files failed to parse:`);
    for (const e of parseErrors.slice(0, 10)) console.error(`  ${e.file}: ${e.message}`);
    process.exit(1);
  }

  for (const edition of await presentEditions()) {
    await syncEdition(edition, source, editionGo);
  }

  await writeJSON(join(CATALOG_DIR, "messages.en.json"), source);
  const keys = Object.keys(source);
  console.log(`i18n: ${keys.length} source strings -> i18n/messages.en.json`);

  const { locales } = await loadLocales();
  for (const locale of locales.filter((l) => l !== "en")) {
    const existing = await readJSONIfExists(localePath(locale), {});
    // Drop translations whose English source no longer exists. Changing the English text
    // changes the key, so the stale entry is an orphan by construction and pruning it here
    // is what keeps the catalogs from silently accumulating dead weight.
    const kept = {};
    let orphans = 0;
    for (const [key, value] of Object.entries(existing)) {
      if (key in source) kept[key] = value;
      else orphans += 1;
    }
    await writeJSON(localePath(locale), sortedObject(kept));
    const missing = keys.filter((k) => !(k in kept)).length;
    console.log(
      `  ${locale.padEnd(6)} ${String(Object.keys(kept).length).padStart(6)} translated, ` +
        `${String(missing).padStart(6)} missing, ${orphans} orphans pruned`,
    );
  }
}

async function pending(locale, { area, limit }) {
  const source = await readJSONIfExists(join(CATALOG_DIR, "messages.en.json"), null);
  if (source === null) {
    console.error("i18n: run `sync` first — i18n/messages.en.json does not exist");
    process.exit(1);
  }
  const translated = await readJSONIfExists(localePath(locale), {});

  let keys = Object.keys(source).filter((k) => !(k in translated));
  if (area) keys = keys.filter((k) => source[k].areas.some((a) => a.startsWith(area)));
  const total = keys.length;
  if (limit) keys = keys.slice(0, limit);

  console.error(`i18n: ${locale} — ${total} pending${area ? ` in ${area}` : ""}, showing ${keys.length}`);
  console.log(JSON.stringify(keys, null, 2));
}

async function merge(locale, file) {
  const source = await readJSONIfExists(join(CATALOG_DIR, "messages.en.json"), null);
  if (source === null) {
    console.error("i18n: run `sync` first");
    process.exit(1);
  }
  const batch = JSON.parse(await readFile(file, "utf8"));
  const existing = await readJSONIfExists(localePath(locale), {});

  let added = 0;
  let updated = 0;
  const unknown = [];
  for (const [key, value] of Object.entries(batch)) {
    if (!(key in source)) {
      unknown.push(key);
      continue;
    }
    if (typeof value !== "string" || value.trim() === "") {
      console.error(`i18n: refusing empty translation for ${JSON.stringify(key)}`);
      process.exit(1);
    }
    if (key in existing) updated += 1;
    else added += 1;
    existing[key] = value;
  }

  if (unknown.length > 0) {
    // A key that is not in the source inventory would sit in the catalog forever without
    // ever being looked up, so this is a mistake worth stopping for rather than pruning.
    console.error(`i18n: ${unknown.length} keys are not source strings, e.g.:`);
    for (const key of unknown.slice(0, 5)) console.error(`  ${JSON.stringify(key)}`);
    process.exit(1);
  }

  await writeJSON(localePath(locale), sortedObject(existing));
  const missing = Object.keys(source).filter((k) => !(k in existing)).length;
  console.log(`i18n: ${locale} +${added} new, ${updated} updated, ${missing} still missing`);
}

async function check() {
  const { source, parseErrors, editionGo } = await buildSource();
  if (parseErrors.length > 0) {
    console.error(`i18n: ${parseErrors.length} files failed to parse`);
    process.exit(1);
  }

  const committed = await readJSONIfExists(join(CATALOG_DIR, "messages.en.json"), null);
  if (committed === null || JSON.stringify(committed) !== JSON.stringify(source)) {
    console.error("i18n: messages.en.json is out of date — run `task i18n` and commit");
    process.exit(1);
  }

  const { locales } = await loadLocales();
  let failed = false;
  for (const locale of locales.filter((l) => l !== "en")) {
    const translated = await readJSONIfExists(localePath(locale), {});
    const missing = Object.keys(source).filter((k) => !(k in translated));
    const orphans = Object.keys(translated).filter((k) => !(k in source));
    if (missing.length > 0 || orphans.length > 0) {
      failed = true;
      console.error(`i18n: ${locale} — ${missing.length} missing, ${orphans.length} orphaned`);
      for (const key of missing.slice(0, 5)) console.error(`    missing: ${JSON.stringify(key)}`);
    }
  }
  for (const edition of await presentEditions()) {
    if (!(await checkEdition(edition, source, editionGo))) failed = true;
  }
  if (!(await checkRuntimeOutputs())) failed = true;
  if (failed) process.exit(1);
  console.log("i18n: catalogs are complete and up to date");
}


// Runtime catalogs are split by scope so the browser never downloads 3,500 backend error
// strings and the Go binary never embeds UI labels. English is emitted empty (Go) or not at
// all (client) on purpose: the key IS the English text, so both runtimes fall back to it
// without a lookup table.
//
// The client side is split again by bundle (bundles.mjs), so the web app downloads the
// strings for its shell at startup and each route folder's strings with that folder's code.
const GO_CATALOG_DIR = "shared/i18n/catalogs";
const CLIENT_CATALOG_DIR = "client/packages/shared/src/i18n/catalogs";
const LOCALE_MODULE = "client/packages/shared/src/i18n/generated/locales.ts";

function jsonText(value) {
  return `${JSON.stringify(value, null, 2)}\n`;
}

function translatedSubset(keys, translated) {
  const subset = {};
  for (const key of keys) {
    if (key in translated) subset[key] = translated[key];
  }
  return sortedObject(subset);
}

// buildRuntimeOutputs computes every file emit owns, keyed by repo-relative path, together
// with the directories it owns outright. Emit writes the files and deletes anything else in
// those directories; check compares them, so a catalog nobody re-emitted fails CI instead of
// shipping a screen in English.
async function buildRuntimeOutputs() {
  const source = await readJSONIfExists(join(CATALOG_DIR, "messages.en.json"), null);
  if (source === null) {
    console.error("i18n: run `sync` first");
    process.exit(1);
  }

  const { locales } = await loadLocales();
  const targets = locales.filter((l) => l !== "en");
  const translations = new Map();
  for (const locale of targets) {
    translations.set(locale, await readJSONIfExists(localePath(locale), {}));
  }

  const files = new Map();
  const ownedDirs = [{ dir: CLIENT_CATALOG_DIR, match: () => true }];
  const summary = [];

  const goKeys = Object.keys(source).filter((k) => source[k].scope.includes("go"));
  files.set(join(GO_CATALOG_DIR, "en.json"), jsonText({}));
  for (const locale of targets) {
    files.set(join(GO_CATALOG_DIR, `${locale}.json`), jsonText(translatedSubset(goKeys, translations.get(locale))));
  }
  ownedDirs.push({ dir: GO_CATALOG_DIR, match: (name) => name.endsWith(".json") });
  summary.push(`i18n: go catalogs -> ${GO_CATALOG_DIR} (${goKeys.length} strings in scope)`);

  const bundles = new Map();
  for (const key of Object.keys(source)) {
    if (!source[key].scope.includes("ts")) continue;
    const bundle = catalogBundle(source[key].areas);
    if (!bundles.has(bundle)) bundles.set(bundle, []);
    bundles.get(bundle).push(key);
  }
  const bundleNames = [...bundles.keys()].sort(compareBundles);
  for (const locale of targets) {
    for (const bundle of bundleNames) {
      files.set(
        join(CLIENT_CATALOG_DIR, locale, bundleFileName(bundle)),
        jsonText(translatedSubset(bundles.get(bundle), translations.get(locale))),
      );
    }
  }
  const tsTotal = [...bundles.values()].reduce((n, keys) => n + keys.length, 0);
  summary.push(
    `i18n: ts catalogs -> ${CLIENT_CATALOG_DIR} (${tsTotal} strings in scope, ${bundleNames.length} bundles)`,
  );

  for (const edition of await presentEditions()) {
    if (edition.runtimeDir === undefined) continue;
    const editionSource = await readJSONIfExists(editionLocalePath(edition.dir, "en"), {});
    const keys = Object.keys(editionSource);
    const runtimeDir = join(edition.dir, edition.runtimeDir);
    files.set(join(runtimeDir, "en.json"), jsonText({}));
    for (const locale of targets) {
      const translated = await readJSONIfExists(editionLocalePath(edition.dir, locale), {});
      files.set(join(runtimeDir, `${locale}.json`), jsonText(translatedSubset(keys, translated)));
    }
    ownedDirs.push({ dir: runtimeDir, match: (name) => name.endsWith(".json") });
    summary.push(`i18n: ${edition.scope} edition catalogs -> ${runtimeDir} (${keys.length} strings)`);
  }

  files.set(LOCALE_MODULE, await localeModule(locales, bundleNames));
  summary.push(`i18n: locale module -> ${LOCALE_MODULE}`);

  return { files, ownedDirs, summary };
}

// ownedFiles lists what is on disk in emit's directories, so a catalog for a bundle that no
// longer exists is found and removed rather than left for nothing to load.
async function ownedFiles(ownedDirs) {
  const found = [];
  for (const { dir, match } of ownedDirs) {
    let names;
    try {
      names = await readdir(join(repoRoot, dir), { recursive: true, withFileTypes: true });
    } catch (err) {
      if (err.code === "ENOENT") continue;
      throw err;
    }
    for (const entry of names) {
      if (!entry.isFile() || !match(entry.name)) continue;
      found.push(relative(repoRoot, join(entry.parentPath, entry.name)));
    }
  }
  return found;
}

async function emit() {
  const { files, ownedDirs, summary } = await buildRuntimeOutputs();

  for (const path of await ownedFiles(ownedDirs)) {
    if (!files.has(path)) await rm(join(repoRoot, path));
  }
  for (const [path, content] of files) {
    const out = join(repoRoot, path);
    await mkdir(dirname(out), { recursive: true });
    await writeFile(out, content, "utf8");
  }

  for (const line of summary) console.log(line);
}

// checkRuntimeOutputs reports every emitted file that differs from what emit would write now,
// and every file in emit's directories that it would delete.
async function checkRuntimeOutputs() {
  const { files, ownedDirs } = await buildRuntimeOutputs();
  const stale = [];
  for (const [path, content] of files) {
    let current = null;
    try {
      current = await readFile(join(repoRoot, path), "utf8");
    } catch (err) {
      if (err.code !== "ENOENT") throw err;
    }
    if (current !== content) stale.push(path);
  }
  for (const path of await ownedFiles(ownedDirs)) {
    if (!files.has(path)) stale.push(path);
  }
  if (stale.length === 0) return true;

  console.error(`i18n: ${stale.length} runtime catalog files are out of date — run \`task i18n\` and commit`);
  for (const path of stale.slice(0, 10)) console.error(`    ${path}`);
  return false;
}

// The frontend needs the locale list as a module rather than JSON so the tags are a union
// type, and it needs one dynamic import per catalog so the bundler can split each into its
// own chunk. Generating both keeps locales.json the only place a language is added.
async function localeModule(locales, bundleNames) {
  const { names, regions } = await loadLocales();
  const quoted = locales.map((l) => `"${l}"`).join(", ");

  const loaders = locales
    .map((l) => {
      if (l === "en") return `  "${l}": {},`;
      const entries = bundleNames.map(
        (b) => `    "${b}": load(() => import("../catalogs/${l}/${bundleFileName(b)}")),`,
      );
      return `  "${l}": {\n${entries.join("\n")}\n  },`;
    })
    .join("\n");

  return `// Code generated by i18n/tools/i18n.mjs. DO NOT EDIT.
//
// Source of truth: i18n/locales.json and the areas in i18n/messages.en.json
// Regenerate with: task i18n

export const LOCALES = [${quoted}] as const;

export type Locale = (typeof LOCALES)[number];

export const DEFAULT_LOCALE: Locale = "en";

export const LOCALE_NAMES: Record<Locale, string> = {
${locales.map((l) => `  "${l}": ${JSON.stringify(names[l] ?? l)},`).join("\n")}
};

// A flag is a country, not a language. These name the region each translation is written
// for, and exist to pick the switcher's flag alone - never to infer a locale from.
export const LOCALE_REGIONS: Record<Locale, string> = {
${locales.map((l) => `  "${l}": ${JSON.stringify((regions ?? {})[l] ?? "")},`).join("\n")}
};

// Every locale's strings are split into the same bundles (i18n/tools/bundles.mjs): "core"
// and "web" load at startup, "dash" only in the driver portal, and each "routes/<dir>"
// with that route folder's code. English has none: its keys are its text.
export const CATALOG_BUNDLES = [
${bundleNames.map((b) => `  "${b}",`).join("\n")}
] as const;

export type CatalogBundle = (typeof CATALOG_BUNDLES)[number];

export type CatalogLoader = () => Promise<Record<string, string>>;

function load(importer: () => Promise<{ default: Record<string, string> }>): CatalogLoader {
  return () => importer().then((m) => m.default);
}

export const CATALOG_LOADERS: Record<Locale, Partial<Record<CatalogBundle, CatalogLoader>>> = {
${loaders}
};

export function isLocale(value: string): value is Locale {
  return (LOCALES as readonly string[]).includes(value);
}

export function isCatalogBundle(value: string): value is CatalogBundle {
  return (CATALOG_BUNDLES as readonly string[]).includes(value);
}
`;
}

const command = process.argv[2] ?? "report";
const showRejected = process.argv.includes("--rejected");

function flag(name) {
  const idx = process.argv.indexOf(`--${name}`);
  return idx === -1 ? null : process.argv[idx + 1];
}

switch (command) {
  case "report":
    await report({ showRejected });
    break;
  case "sync":
    await sync();
    break;
  case "pending":
    await pending(process.argv[3], { area: flag("area"), limit: Number(flag("limit")) || 0 });
    break;
  case "merge":
    await merge(process.argv[3], process.argv[4]);
    break;
  case "emit":
    await emit();
    break;
  case "codemod": {
    const target = process.argv[3];
    if (!target) {
      console.error("i18n: codemod needs a directory, e.g. client/apps/web/src/routes/auth");
      process.exit(1);
    }
    const write = process.argv.includes("--write");
    const labels = process.argv.includes("--labels");
    const result = await runCodemod(repoRoot, target, { dryRun: !write, labels });
    console.log(
      `i18n: ${write ? "rewrote" : "would rewrite"} ${result.changedFiles}/${result.files} files, ` +
        `${result.replacements} literals wrapped`,
    );
    if (result.skipped.length > 0) {
      console.log(`\n  ${result.skipped.length} sites skipped:`);
      const byReason = new Map();
      for (const s of result.skipped) byReason.set(s.reason, (byReason.get(s.reason) ?? 0) + 1);
      for (const [reason, n] of byReason) console.log(`    ${String(n).padStart(5)}  ${reason}`);
      for (const s of result.skipped.slice(0, 5)) {
        console.log(`      ${s.file}: ${s.detail ?? ""}`.slice(0, 140));
      }
    }
    break;
  }
  case "check":
    await check();
    break;
  default:
    console.error(`unknown command: ${command}`);
    process.exit(1);
}
