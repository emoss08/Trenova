#!/usr/bin/env node
// i18n.mjs is the single entry point for the translation pipeline.
//
//   report                 inventory of source strings, per locale and per area
//   sync                   rebuild i18n/messages.en.json from source; create locale files
//   pending <locale>       print strings still needing translation (--area, --limit)
//   merge <locale> <file>  merge a batch of translations into that locale's catalog
//   emit                   write the per-scope runtime catalogs the apps load
//   check                  CI gate: fail on missing or orphaned entries
//   codemod <dir> [--write]  wrap user-facing literals in t() under <dir>
//
// Both extractors feed one merged catalog keyed by the English source string, so a message
// written once in Go and again in React is translated once.
import { execFile } from "node:child_process";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";
import { runCodemod } from "./codemod.mjs";
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
    const reason = reject(item.message, {});
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
  const [go, ts] = await Promise.all([extractGo(), extractTypeScript(repoRoot)]);

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
  const [go, ts] = await Promise.all([extractGo(), extractTypeScript(repoRoot)]);
  const source = {};
  for (const entry of [...go.entries, ...ts.entries]) {
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
  return { source: sortedObject(source), parseErrors: ts.errors };
}

async function sync() {
  const { source, parseErrors } = await buildSource();
  if (parseErrors.length > 0) {
    console.error(`i18n: ${parseErrors.length} files failed to parse:`);
    for (const e of parseErrors.slice(0, 10)) console.error(`  ${e.file}: ${e.message}`);
    process.exit(1);
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
  const { source, parseErrors } = await buildSource();
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
  if (failed) process.exit(1);
  console.log("i18n: catalogs are complete and up to date");
}


// Runtime catalogs are split by scope so the browser never downloads 3,500 backend error
// strings and the Go binary never embeds UI labels. English is emitted empty on purpose:
// the key IS the English text, so both runtimes fall back to it without a lookup table.
const EMIT_TARGETS = [
  { scope: "go", dir: "shared/i18n/catalogs" },
  { scope: "ts", dir: "client/packages/shared/src/i18n/catalogs" },
];

async function emit() {
  const source = await readJSONIfExists(join(CATALOG_DIR, "messages.en.json"), null);
  if (source === null) {
    console.error("i18n: run `sync` first");
    process.exit(1);
  }

  const { locales } = await loadLocales();

  for (const { scope, dir } of EMIT_TARGETS) {
    const keys = Object.keys(source).filter((k) => source[k].scope.includes(scope));

    for (const locale of locales) {
      const out = join(repoRoot, dir, `${locale}.json`);

      if (locale === "en") {
        await writeJSON(out, {});
        continue;
      }

      const translated = await readJSONIfExists(localePath(locale), {});
      const subset = {};
      for (const key of keys) {
        if (key in translated) subset[key] = translated[key];
      }
      await writeJSON(out, sortedObject(subset));
    }

    console.log(`i18n: ${scope} catalogs -> ${dir} (${keys.length} strings in scope)`);
  }

  await emitLocaleModule(locales);
}

// The frontend needs the locale list as a module rather than JSON so the tags are a union
// type, and it needs one dynamic import per locale so the bundler can split each catalog
// into its own chunk. Generating both keeps locales.json the only place a language is added.
async function emitLocaleModule(locales) {
  const { names } = await loadLocales();
  const quoted = locales.map((l) => `"${l}"`).join(", ");

  const body = `// Code generated by i18n/tools/i18n.mjs. DO NOT EDIT.
//
// Source of truth: i18n/locales.json
// Regenerate with: task i18n

export const LOCALES = [${quoted}] as const;

export type Locale = (typeof LOCALES)[number];

export const DEFAULT_LOCALE: Locale = "en";

export const LOCALE_NAMES: Record<Locale, string> = {
${locales.map((l) => `  "${l}": ${JSON.stringify(names[l] ?? l)},`).join("\n")}
};

export const CATALOG_LOADERS: Record<Locale, () => Promise<Record<string, string>>> = {
${locales.map((l) => `  "${l}": () => import("../catalogs/${l}.json").then((m) => m.default),`).join("\n")}
};

export function isLocale(value: string): value is Locale {
  return (LOCALES as readonly string[]).includes(value);
}
`;

  const out = join(repoRoot, "client/packages/shared/src/i18n/generated/locales.ts");
  await mkdir(dirname(out), { recursive: true });
  await writeFile(out, body, "utf8");
  console.log(`i18n: locale module -> client/packages/shared/src/i18n/generated/locales.ts`);
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
    const result = await runCodemod(repoRoot, target, { dryRun: !write });
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
