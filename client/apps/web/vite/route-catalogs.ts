import MagicString from "magic-string";
import { parseSync } from "oxc-parser";
import type { Plugin } from "vite";

// Each folder under src/routes has its own catalog bundle per locale (i18n/tools/bundles.mjs
// puts a string there when nothing outside the folder renders it). This plugin does two
// things to the app's source so no screen ever shows English while its strings download:
//
// 1. Every module in a route folder requires that folder's bundle as it is evaluated, so
//    whichever code imports the module — its own route, or another folder borrowing a
//    component — has asked for the strings it renders. Requiring a folder with no strings
//    is a no-op.
// 2. Every dynamic import of app code waits for the bundles its module graph required
//    (`import("./panel").then(afterCatalogs)`), so a React.lazy component keeps showing its
//    Suspense skeleton until it can render translated. When nothing is downloading — always,
//    in English — the wait is a few microtasks.
//
// The folder rule mirrors featureArea in i18n/tools/extract-ts.mjs: a file directly in
// src/routes is part of the shell and ships its strings in the "web" bundle.

const SOURCE_FILE = /\.[cm]?[jt]sx?$/;
const NOT_APP_CODE = /\.(test|spec|stories)\.[cm]?[jt]sx?$|\/__tests__\//;
const RUNTIME = "@trenova/shared/i18n/runtime";

function toPosix(path: string): string {
  return path.replaceAll("\\", "/");
}

function asDirectory(path: string): string {
  return `${toPosix(path).replace(/\/+$/, "")}/`;
}

/**
 * appSourceFile returns the module's path when it is the app's own code under srcDir, or
 * null. A module id with a query (`?raw`, `?worker`) is an asset or a different module
 * kind, never the app's code.
 */
function appSourceFile(id: string, srcDir: string): string | null {
  if (id.includes("?") || id.includes("\0")) return null;

  const file = toPosix(id);
  if (!file.startsWith(asDirectory(srcDir))) return null;
  if (!SOURCE_FILE.test(file) || NOT_APP_CODE.test(file)) return null;
  return file;
}

/**
 * routeCatalogFor names the bundle a module requires, or null for anything that is not app
 * code inside a route folder.
 */
export function routeCatalogFor(id: string, routesDir: string): string | null {
  const file = appSourceFile(id, routesDir);
  if (file === null) return null;

  const rest = file.slice(asDirectory(routesDir).length);
  const slash = rest.indexOf("/");
  if (slash <= 0) return null;

  return `routes/${rest.slice(0, slash)}`;
}

type DynamicImport = { specifier: string; end: number };

type AstNode = { type?: unknown; [key: string]: unknown };

function isNode(value: unknown): value is AstNode {
  return typeof value === "object" && value !== null;
}

/**
 * dynamicImports finds every `import("literal")` in a module. Type positions
 * (`typeof import("./x")`) are a different node and are not matched; an import of a
 * computed specifier cannot be resolved here and is left alone.
 */
export function dynamicImports(code: string, filename: string): DynamicImport[] {
  if (!code.includes("import(")) return [];

  const { program } = parseSync(filename, code);
  const found: DynamicImport[] = [];
  const stack: unknown[] = [program];
  while (stack.length > 0) {
    const value = stack.pop();
    if (Array.isArray(value)) {
      for (const item of value) stack.push(item);
      continue;
    }
    if (!isNode(value)) continue;

    if (value.type === "ImportExpression" && isNode(value.source)) {
      const { source } = value;
      if (source.type === "Literal" && typeof source.value === "string") {
        found.push({ specifier: source.value, end: value.end as number });
      }
    }
    for (const key in value) {
      if (key !== "parent") stack.push(value[key]);
    }
  }
  return found.sort((a, b) => a.end - b.end);
}

export const REQUIRE_CATALOG = "__trenovaRequireCatalog";
export const AFTER_CATALOGS = "__trenovaAfterCatalogs";

type RouteCatalogsOptions = { srcDir: string };

export function routeCatalogs({ srcDir }: RouteCatalogsOptions): Plugin {
  const routesDir = `${asDirectory(srcDir)}routes`;

  return {
    name: "trenova:route-catalogs",
    enforce: "pre",
    async transform(code, id) {
      const file = appSourceFile(id, srcDir);
      if (file === null) return null;

      const waits: number[] = [];
      for (const { specifier, end } of dynamicImports(code, file)) {
        const resolved = await this.resolve(specifier, id);
        if (resolved === null || resolved.external) continue;
        if (appSourceFile(resolved.id, srcDir) !== null) waits.push(end);
      }

      const bundle = routeCatalogFor(id, routesDir);
      if (bundle === null && waits.length === 0) return null;

      const out = new MagicString(code);
      for (const end of waits) out.appendLeft(end, `.then(${AFTER_CATALOGS})`);

      const names = [
        ...(bundle === null ? [] : [`requireCatalog as ${REQUIRE_CATALOG}`]),
        ...(waits.length === 0 ? [] : [`afterCatalogs as ${AFTER_CATALOGS}`]),
      ];
      // At the end, so the original lines keep their numbers; imports are hoisted, and the
      // require runs once the module body has, still before anything importing it renders.
      out.append(`\nimport { ${names.join(", ")} } from "${RUNTIME}";\n`);
      if (bundle !== null) out.append(`${REQUIRE_CATALOG}(${JSON.stringify(bundle)});\n`);

      return { code: out.toString(), map: out.generateMap({ hires: true, source: id }) };
    },
  };
}
