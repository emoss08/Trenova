import { navigationSections, type NavigationSection } from "@/lib/navigation-sections";
import type { RouteObject } from "react-router";

const CATCH_ALL = "*";

function joinPath(parent: string, child: string): string {
  const joined = child.startsWith("/") ? child : `${parent.replace(/\/$/, "")}/${child}`;
  return joined.length > 1 ? joined.replace(/\/$/, "") : joined;
}

/** The id every mounted section index carries, so it can be told from a page at the same path. */
export function sectionIndexRouteId(path: string): string {
  return `section-index:${path}`;
}

function sectionIndexRoute(
  path: string,
  placement: Pick<RouteObject, "path"> | { index: true },
): RouteObject {
  return {
    ...placement,
    id: sectionIndexRouteId(path),
    async lazy() {
      const { SectionIndexPage } = await import("@/routes/section-index/page");
      return { Component: SectionIndexPage };
    },
  } as RouteObject;
}

interface RouteScan {
  /** Paths a route renders a page at: a leaf, or a layout with an index. */
  pages: Set<string>;
  /** Layout routes with no index, by path: they draw only their frame there. */
  bareLayouts: Map<string, RouteObject>;
  /** The children list that ends in the signed-in app's not-found route. */
  catchAllSiblings: RouteObject[] | null;
}

function cloneTree(routes: readonly RouteObject[]): RouteObject[] {
  return routes.map((route) =>
    route.children ? ({ ...route, children: cloneTree(route.children) } as RouteObject) : route,
  );
}

function scan(routes: RouteObject[], parent: string, found: RouteScan): RouteScan {
  for (const route of routes) {
    if (route.path === CATCH_ALL) {
      found.catchAllSiblings ??= routes;
      continue;
    }
    const path = route.path === undefined ? parent : joinPath(parent, route.path);
    if (route.index) {
      found.pages.add(path);
      continue;
    }
    if (!route.children) {
      if (route.path !== undefined) found.pages.add(path);
      continue;
    }
    if (route.path !== undefined && !route.children.some((child) => child.index)) {
      found.bareLayouts.set(path, route);
    }
    if (route.children.some((child) => child.index)) {
      found.pages.add(path);
    }
    scan(route.children, path, found);
  }
  return found;
}

/**
 * Returns the route tree with the section index mounted at every section the
 * navigation names that no page fills, so a crumb such as "Dispatch
 * management" opens the part of the app it names instead of the not-found
 * page. A section whose path is a layout that draws only its frame (Settings
 * at /admin) gets the index inside that frame; any other sits beside the
 * not-found route. A section a page already fills keeps its page.
 */
export function withSectionIndexes(
  routes: readonly RouteObject[],
  sections: readonly NavigationSection[] = navigationSections,
): RouteObject[] {
  const tree = cloneTree(routes);
  const found = scan(tree, "", {
    pages: new Set(),
    bareLayouts: new Map(),
    catchAllSiblings: null,
  });
  const siblings = found.catchAllSiblings;
  if (siblings === null) {
    throw new Error("withSectionIndexes needs a not-found route to mount sections beside");
  }

  const mounted: RouteObject[] = [];
  for (const section of sections) {
    if (found.pages.has(section.path)) {
      continue;
    }
    const layout = found.bareLayouts.get(section.path);
    if (layout) {
      layout.children = [
        sectionIndexRoute(section.path, { index: true }),
        ...(layout.children ?? []),
      ];
      continue;
    }
    mounted.push(sectionIndexRoute(section.path, { path: section.path }));
  }

  siblings.splice(
    siblings.findIndex((route) => route.path === CATCH_ALL),
    0,
    ...mounted,
  );
  return tree;
}
