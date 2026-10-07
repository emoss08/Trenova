// bundles.mjs decides which runtime catalog a client string ships in.
//
// One catalog per locale grew past a megabyte, and the web app blocked its first render on
// downloading all of it. A string now ships in the smallest catalog that every screen using
// it is guaranteed to have loaded:
//
//   core          used by packages/shared, or by both apps. Loaded at startup by every app.
//   web           used by the web app's shell, or by more than one of its route folders.
//                 Loaded at startup by the web app.
//   dash          used only by the driver portal. Never downloaded by the web app.
//   routes/<dir>  used only inside client/apps/web/src/routes/<dir>/. Loaded alongside that
//                 folder's code, because every module in the folder asks for it as it is
//                 evaluated (client/packages/shared/node/i18n-route-catalogs.ts).
//
// The route rule and featureArea in extract-ts.mjs are one convention seen from two sides:
// a string extracted from client/apps/web/src/routes/<dir>/ is labelled `routes/<dir>`, and
// a module in that folder requires the `routes/<dir>` catalog. Change both or neither.

export const CORE_BUNDLE = "core";
export const WEB_BUNDLE = "web";
export const DASH_BUNDLE = "dash";

const ROUTE_AREA = /^routes\/[^/]+$/;

/**
 * catalogBundle names the runtime catalog for a string from the areas it was found in. Go
 * areas (domain/…, service/…) are ignored: the browser only needs the string where the
 * client renders it. A string with no client area at all falls back to core so it can never
 * be stranded in a catalog nothing loads.
 */
export function catalogBundle(areas) {
  let shared = false;
  let web = false;
  let webShell = false;
  let dash = false;
  const routes = new Set();

  for (const area of areas) {
    if (area.startsWith("packages/")) {
      shared = true;
    } else if (area.startsWith("apps/dash/")) {
      dash = true;
    } else if (area.startsWith("apps/web/")) {
      web = true;
      webShell = true;
    } else if (ROUTE_AREA.test(area)) {
      web = true;
      routes.add(area);
    }
  }

  if (shared || (web && dash) || (!web && !dash)) return CORE_BUNDLE;
  if (dash) return DASH_BUNDLE;
  if (!webShell && routes.size === 1) return routes.values().next().value;
  return WEB_BUNDLE;
}

/**
 * bundleFileName flattens a bundle name into one file name, so every catalog for a locale
 * sits in one directory and the chunk the bundler emits for it says which bundle it is
 * (`routes.shipment.<hash>.js`, not a second `shipment.<hash>.js` beside the page's own).
 */
export function bundleFileName(bundle) {
  return `${bundle.replaceAll("/", ".")}.json`;
}

/**
 * compareBundles orders the two startup catalogs first and the route catalogs after them by
 * name, so the generated loader table reads in the order an app loads it.
 */
export function compareBundles(a, b) {
  const rank = (name) => (name === CORE_BUNDLE ? 0 : name === WEB_BUNDLE ? 1 : name === DASH_BUNDLE ? 2 : 3);
  return rank(a) - rank(b) || (a < b ? -1 : a > b ? 1 : 0);
}
