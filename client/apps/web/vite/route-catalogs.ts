import type { Plugin } from "vite";

// Each folder under src/routes has its own catalog bundle per locale (i18n/tools/bundles.mjs
// puts a string there when nothing outside the folder renders it). This plugin makes every
// module in the folder require that bundle as it is evaluated, so whichever route imports
// the module — its own, or another folder borrowing a component — waits for the strings it
// renders (src/lib/route-catalogs.ts). Requiring a folder with no strings is a no-op.
//
// The folder rule mirrors featureArea in i18n/tools/extract-ts.mjs: a file directly in
// src/routes is part of the shell and ships its strings in the "web" bundle.

const SOURCE_FILE = /\.[cm]?[jt]sx?$/;
const NOT_APP_CODE = /\.(test|spec|stories)\.[cm]?[jt]sx?$|\/__tests__\//;

function toPosix(path: string): string {
  return path.replaceAll("\\", "/");
}

/**
 * routeCatalogFor names the bundle a module requires, or null for anything that is not app
 * code inside a route folder. A module id with a query (`?raw`, `?worker`) is an asset or
 * a different module kind, never the folder's own code.
 */
export function routeCatalogFor(id: string, routesDir: string): string | null {
  if (id.includes("?") || id.includes("\0")) return null;

  const file = toPosix(id);
  const root = `${toPosix(routesDir).replace(/\/+$/, "")}/`;
  if (!file.startsWith(root) || !SOURCE_FILE.test(file) || NOT_APP_CODE.test(file)) return null;

  const rest = file.slice(root.length);
  const slash = rest.indexOf("/");
  if (slash <= 0) return null;

  return `routes/${rest.slice(0, slash)}`;
}

/**
 * The statement appended to a route module. It goes at the end so no line moves and the
 * module's source map stays exact; imports are hoisted, and the call runs once the module
 * body has, which is still before anything that imported the module can render.
 */
export function requireCatalogStatement(bundle: string): string {
  return (
    `\nimport { requireCatalog as __trenovaRequireCatalog } from "@trenova/shared/i18n/runtime";` +
    `\n__trenovaRequireCatalog(${JSON.stringify(bundle)});\n`
  );
}

export function routeCatalogs({ routesDir }: { routesDir: string }): Plugin {
  return {
    name: "trenova:route-catalogs",
    enforce: "pre",
    transform(code, id) {
      const bundle = routeCatalogFor(id, routesDir);
      if (bundle === null) return null;
      return { code: code + requireCatalogStatement(bundle), map: null };
    },
  };
}
