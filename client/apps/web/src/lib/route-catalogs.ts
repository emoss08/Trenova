import { whenCatalogsReady } from "@trenova/shared/i18n/runtime";
import type { RouteObject } from "react-router";

// A route's strings ship in its folder's catalog bundle, which every module in the folder
// requires as it is evaluated (vite/route-catalogs.ts injects the call). By the time a
// route's lazy() import resolves, its whole module graph has evaluated, so every bundle the
// page can render from has been asked for — including those of other route folders it
// borrows components from. Waiting for them here is what lets the page's first frame render
// translated instead of in English and then again.
//
// A bundle that fails to download must not fail the navigation: the page renders in
// English, and the bundle is retried on the next navigation.
async function afterCatalogs<T>(loaded: Promise<T>): Promise<T> {
  const value = await loaded;
  await whenCatalogsReady().catch(() => undefined);
  return value;
}

function withCatalogs(route: RouteObject): RouteObject {
  const { lazy, children } = route;
  const next: RouteObject = { ...route };

  if (typeof lazy === "function") {
    next.lazy = () => afterCatalogs(lazy());
  } else if (lazy !== undefined) {
    next.lazy = Object.fromEntries(
      Object.entries(lazy).map(([key, load]) => [
        key,
        typeof load === "function" ? () => afterCatalogs(load()) : load,
      ]),
    );
  }

  if (children !== undefined) {
    next.children = children.map(withCatalogs);
  }

  return next as RouteObject;
}

/**
 * withRouteCatalogs returns the route tree with every lazy route waiting for the catalog
 * bundles its code requires before it resolves. Edition routes are mounted inside the same
 * tree, so they are covered too.
 */
export function withRouteCatalogs(routes: readonly RouteObject[]): RouteObject[] {
  return routes.map(withCatalogs);
}
