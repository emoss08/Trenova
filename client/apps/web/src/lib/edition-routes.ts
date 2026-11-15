import { edition } from "@/lib/edition";
import { lazyPage } from "@/lib/route-catalogs";
import type { EditionRouteMount } from "@trenova/edition";
import { gates } from "@trenova/shared/lib/router/gates";
import { createRoute, type AnyRoute } from "@tanstack/react-router";

/**
 * The routes the installed edition adds at `mount`, under `parent`. They are not known when the
 * app is compiled, so they are typed `never` and stay out of the route tree's types: links to
 * the app's own pages are still checked against the pages that exist.
 */
export function editionRoutes(parent: AnyRoute, mount: EditionRouteMount): never[] {
  return edition.routes[mount].map((route) =>
    createRoute({
      getParentRoute: () => parent,
      path: route.path,
      beforeLoad: route.gates ? gates(...route.gates) : undefined,
      component: lazyPage(() => route.page().then((page) => ({ page })), "page"),
    }),
  ) as never[];
}
