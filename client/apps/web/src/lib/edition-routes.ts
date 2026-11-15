import { edition } from "@/lib/edition";
import { Route as RootRoute } from "@/pages/__root";
import { Route as AppRoute } from "@/pages/_app";
import { Route as AdminRoute } from "@/pages/_app/admin";
import { Route as GuestRoute } from "@/pages/_guest";
import type { EditionRoute, EditionRouteMount } from "@trenova/edition";
import { afterCatalogs } from "@trenova/shared/i18n/runtime";
import { gates } from "@trenova/shared/lib/router/gates";
import { createRoute, lazyRouteComponent, type AnyRoute } from "@tanstack/react-router";

const MOUNTS: Readonly<Record<EditionRouteMount, AnyRoute>> = {
  protected: AppRoute,
  admin: AdminRoute,
  public: RootRoute,
  guest: GuestRoute,
};

function editionRoute(parent: AnyRoute, route: EditionRoute): AnyRoute {
  return createRoute({
    getParentRoute: () => parent,
    path: route.path,
    beforeLoad: route.gates ? gates(...route.gates) : undefined,
    component: lazyRouteComponent(
      () =>
        route
          .page()
          .then(afterCatalogs)
          .then((page) => ({ page })),
      "page",
    ),
  });
}

/**
 * Adds the installed edition's pages to the layouts they mount in. They are not known when the
 * app is compiled, so they join the generated tree before the router is built and stay out of
 * its types: links to the app's own pages are still checked against the pages that exist.
 */
export function mountEditionRoutes(): void {
  for (const [mount, parent] of Object.entries(MOUNTS) as [EditionRouteMount, AnyRoute][]) {
    const routes = edition.routes[mount];
    if (routes.length === 0) continue;
    const children = (parent.children ?? []) as AnyRoute[];
    parent.addChildren([...children, ...routes.map((route) => editionRoute(parent, route))]);
  }
}
