#!/usr/bin/env node
/**
 * Builds the product guide the agents read: every page a signed-in person can
 * open, where it sits, what guards it, what it is for and how to do things on
 * it, plus where each kind of record opens.
 *
 * The structure comes from the app itself — the router, the navigation config,
 * each page's header, the record-link registry — so a page cannot be added,
 * moved or re-guarded without the catalog following. The words come from the
 * guides under docs/product-guide, one per page, and the checks below are what
 * keep them honest: a page with no guide, a guide for a page that is gone, a
 * link to nowhere, or a **label** the app no longer shows all fail the build.
 *
 * Usage:
 *   node scripts/product-guide/generate.mjs           write the catalog
 *   node scripts/product-guide/generate.mjs --check   fail if it is stale
 */

import { createHash } from "node:crypto";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import process from "node:process";

import { readGuides, referencesIn } from "./guides.mjs";
import {
  loadMembers,
  readNavigation,
  readPageHeader,
  readRecordLinks,
  readRoutes,
  sourcePaths,
} from "./sources.mjs";

const WEB_ROOT = new URL("../..", import.meta.url).pathname;
const REPO_ROOT = join(WEB_ROOT, "..", "..", "..");
const GUIDES_DIR = join(REPO_ROOT, "docs", "product-guide");
const MESSAGES = join(REPO_ROOT, "i18n", "messages.en.json");
export const OUTPUT = join(REPO_ROOT, "services", "tms", "pkg", "productguide", "catalog_gen.json");

const CHECK = process.argv.includes("--check");

function normalizePath(path) {
  if (path === "/") {
    return path;
  }

  return path.replace(/\/+$/, "");
}

function titleCase(segment) {
  const words = segment.replace(/[-_]+/g, " ").trim();
  return words.charAt(0).toUpperCase() + words.slice(1);
}

function firstSentence(text) {
  const match = /^(.+?[.!?])(\s|$)/.exec(text);
  return (match ? match[1] : text).trim();
}

/** Where each navigable page sits: module, group and the label people see. */
function navigationIndex(navigation) {
  const index = new Map();
  for (const module of navigation.modules) {
    for (const entry of module.navigation ?? []) {
      const items = Array.isArray(entry.items) ? entry.items : [entry];
      const group = Array.isArray(entry.items) ? entry.label : "";
      for (const item of items) {
        if (typeof item.path !== "string" || item.disabled || item.external) {
          continue;
        }
        index.set(normalizePath(item.path), {
          module: module.id,
          moduleLabel: module.label,
          group,
          label: item.label,
        });
      }
    }
  }

  const admin = navigation.modules.find((module) => module.id === "admin");
  for (const link of navigation.adminLinks) {
    if (typeof link.href !== "string" || link.disabled) {
      continue;
    }
    const path = normalizePath(link.href);
    if (!index.has(path)) {
      index.set(path, {
        module: "admin",
        moduleLabel: admin?.label ?? "Organization settings",
        group: link.group ?? "",
        label: link.title,
      });
    }
  }

  return index;
}

function moduleFor(path, navigation) {
  let best = null;
  for (const module of navigation.modules) {
    const prefixes = [...(module.routePrefixes ?? [])];
    if (module.basePath && module.basePath !== "#") {
      prefixes.push(module.basePath);
    }
    for (const prefix of prefixes) {
      const matches =
        prefix === "/" ? path === "/" : path === prefix || path.startsWith(`${prefix}/`);
      if (matches && (best === null || prefix.length > best.prefix.length)) {
        best = { prefix, module };
      }
    }
  }

  return best?.module ?? null;
}

function routePattern(path) {
  return new RegExp(
    `^${path
      .split("/")
      .map((segment) =>
        segment.startsWith(":") ? "[^/]+" : segment.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"),
      )
      .join("/")}$`,
  );
}

function knownLabels(navigation, pages, headers, recordLinks) {
  const labels = new Set();
  const messages = JSON.parse(readFileSync(MESSAGES, "utf8"));
  for (const [text, entry] of Object.entries(messages)) {
    if (entry.scope?.includes("ts")) {
      labels.add(text);
    }
  }
  for (const module of navigation.modules) {
    labels.add(module.label);
    if (module.shortLabel) labels.add(module.shortLabel);
    for (const entry of module.navigation ?? []) {
      labels.add(entry.label);
      for (const item of entry.items ?? []) labels.add(item.label);
    }
  }
  for (const action of navigation.quickActions) labels.add(action.label);
  for (const link of navigation.adminLinks) if (link.title) labels.add(link.title);
  for (const header of headers.values()) if (header?.title) labels.add(header.title);
  for (const page of pages) labels.add(page.name);
  for (const link of Object.values(recordLinks)) labels.add(link.label);

  return labels;
}

/**
 * Whether a bold label is text the app shows. Tables name their own create
 * button "New {thing}" from the table's name, so that form is accepted when
 * the thing is itself a label the app uses.
 */
function isKnownLabel(label, labels, lowerLabels) {
  if (labels.has(label)) {
    return true;
  }
  const generated = /^New (.+)$/.exec(label);
  return generated !== null && lowerLabels.has(generated[1].toLowerCase());
}

function build() {
  const errors = [];
  const paths = sourcePaths(WEB_ROOT);
  const members = loadMembers(paths);
  const routes = readRoutes(paths, members);
  const navigation = readNavigation(paths, members);
  const recordLinks = readRecordLinks(paths);
  const guides = readGuides(GUIDES_DIR, REPO_ROOT, errors);
  const navIndex = navigationIndex(navigation);

  const pagesByPath = new Map();
  const paramRoutes = [];
  for (const route of routes.values()) {
    if (route.hasParams) {
      paramRoutes.push(route);
    } else {
      pagesByPath.set(normalizePath(route.path), route);
    }
  }

  // Every entry in the navigation has to be a page the router serves;
  // otherwise the sidebar and the guide would both send people nowhere.
  for (const path of navIndex.keys()) {
    if (!pagesByPath.has(path)) {
      errors.push(`navigation links to ${path}, which no route serves`);
    }
  }

  const guidesByPath = new Map();
  const coveredBy = new Map();
  for (const guide of guides) {
    const path = normalizePath(guide.path);
    if (!pagesByPath.has(path)) {
      errors.push(`${guide.file}: ${path} is not a page the router serves`);
      continue;
    }
    if (guidesByPath.has(path)) {
      errors.push(`${guide.file}: ${path} already has a guide (${guidesByPath.get(path).file})`);
      continue;
    }
    guidesByPath.set(path, guide);
    for (const covered of guide.covers.map(normalizePath)) {
      const served =
        pagesByPath.has(covered) || paramRoutes.some((route) => route.path === covered);
      if (!served) {
        errors.push(`${guide.file}: covers ${covered}, which no route serves`);
        continue;
      }
      coveredBy.set(covered, path);
    }
  }
  for (const [covered, owner] of coveredBy) {
    if (guidesByPath.has(covered)) {
      errors.push(
        `${guidesByPath.get(covered).file}: ${covered} has its own guide and is also covered by ${owner}`,
      );
    }
  }

  for (const [path, route] of pagesByPath) {
    if (!guidesByPath.has(path) && !coveredBy.has(path)) {
      errors.push(`no guide for ${path} (${relative(REPO_ROOT, route.file)})`);
    }
  }

  const headers = new Map();
  for (const [path, route] of pagesByPath) {
    headers.set(path, readPageHeader(route.file, route.exportName));
  }

  const createActions = new Map();
  for (const action of navigation.quickActions) {
    if (action.query?.panelType === "create" && typeof action.path === "string") {
      createActions.set(normalizePath(action.path), {
        label: action.label,
        query: action.query,
      });
    }
  }

  const pages = [];
  for (const [path, guide] of [...guidesByPath].sort(([a], [b]) => a.localeCompare(b))) {
    const route = pagesByPath.get(path);
    const nav = navIndex.get(path);
    const header = headers.get(path);
    const module = nav ? null : moduleFor(path, navigation);
    const segments = path.split("/").filter(Boolean);
    const name =
      nav?.label ??
      (guide.title || header?.title || titleCase(segments[segments.length - 1] ?? "Home"));
    const moduleLabel = nav?.moduleLabel ?? module?.label ?? "";
    const breadcrumb = [moduleLabel, nav?.group ?? "", name].filter(
      (part, index, all) => part !== "" && all.indexOf(part) === index,
    );

    pages.push({
      path,
      name,
      module: nav?.module ?? module?.id ?? "",
      breadcrumb,
      description: header?.description || firstSentence(guide.summary),
      summary: guide.summary,
      requires: route.requires,
      capabilities: route.capabilities,
      aliases: guide.aliases,
      tasks: guide.tasks,
      notes: guide.notes,
      related: guide.related.map(normalizePath),
      covers: guide.covers.map(normalizePath),
      createAction: createActions.get(path) ?? null,
      inNavigation: nav !== undefined,
    });
  }

  const known = new Set(pages.map((page) => page.path));
  const labels = knownLabels(navigation, pages, headers, recordLinks);
  const lowerLabels = new Set([...labels].map((label) => label.toLowerCase()));
  for (const guide of guides) {
    const { labels: used, links } = referencesIn(guide);
    for (const label of used) {
      if (!isKnownLabel(label, labels, lowerLabels)) {
        const near = lowerLabels.has(label.toLowerCase())
          ? " (the app writes it in another case)"
          : "";
        errors.push(`${guide.file}: **${label}** is not text the app shows${near}`);
      }
    }
    for (const link of links) {
      const target = normalizePath(link.split(/[?#]/)[0]);
      const served =
        known.has(target) || paramRoutes.some((route) => routePattern(route.path).test(target));
      if (!link.startsWith("/") || !served) {
        errors.push(`${guide.file}: link ${link} is not a page in the app`);
      }
    }
    for (const related of guide.related.map(normalizePath)) {
      if (!known.has(related)) {
        errors.push(`${guide.file}: related ${related} is not a page with a guide`);
      }
    }
  }

  const records = [];
  for (const [entity, link] of Object.entries(recordLinks).sort(([a], [b]) => a.localeCompare(b))) {
    const target = normalizePath(link.path.replaceAll("{id}", "x"));
    const served =
      pagesByPath.has(target) || paramRoutes.some((route) => routePattern(route.path).test(target));
    if (!served) {
      errors.push(`record link ${entity} opens on ${link.path}, which no route serves`);
    }
    records.push({ entity, label: link.label, path: link.path, params: link.params ?? {} });
  }

  const modules = navigation.modules.map((module) => ({
    id: module.id,
    label: module.label,
    description: module.description ?? "",
  }));

  const body = { modules, pages, records };
  const version = createHash("sha256").update(JSON.stringify(body)).digest("hex");

  return { errors, catalog: { version: `sha256:${version}`, ...body } };
}

const { errors, catalog } = build();
if (errors.length > 0) {
  for (const error of errors) {
    console.error(`product guide: ${error}`);
  }
  console.error(`\n${errors.length} problem(s). See docs/engineering/product-guide.md.`);
  process.exit(1);
}

const serialized = `${JSON.stringify(catalog, null, 2)}\n`;
if (CHECK) {
  const current = existsSync(OUTPUT) ? readFileSync(OUTPUT, "utf8") : "";
  if (current !== serialized) {
    console.error(
      "product guide: the catalog is stale. Run `pnpm --filter @trenova/web guide:generate` and commit.",
    );
    process.exit(1);
  }
  console.log(`product guide: up to date (${catalog.pages.length} pages)`);
} else {
  mkdirSync(dirname(OUTPUT), { recursive: true });
  writeFileSync(OUTPUT, serialized);
  console.log(
    `product guide: wrote ${catalog.pages.length} pages to ${relative(REPO_ROOT, OUTPUT)}`,
  );
}
