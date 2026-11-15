import { parseSearch, stringifySearch } from "@trenova/shared/lib/router/search";
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterContextProvider,
  RouterProvider,
  type AnyRoute,
  type AnyRouter,
  type RouteComponent,
  type StaticDataRouteOption,
} from "@tanstack/react-router";
import { render, type RenderResult } from "@testing-library/react";
import { useState, type ComponentType, type ReactNode } from "react";

/** A router over `routeTree` that starts at `initialEntries`, reading the query as the app does. */
export function createTestRouter(
  routeTree: AnyRoute = createRootRoute({ component: Outlet }),
  initialEntries: readonly string[] = ["/"],
): AnyRouter {
  return createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [...initialEntries] }),
    parseSearch,
    stringifySearch,
  });
}

/**
 * The router a component under test reads: links, navigation, the location and the history
 * work, and nothing is matched or rendered by the router itself, so `children` render on the
 * first pass. `router` is for a test that inspects where the component navigated
 * (`router.state.location`).
 */
export function TestRouter({
  children,
  initialEntries = ["/"],
  router: given,
}: {
  children: ReactNode;
  initialEntries?: readonly string[];
  router?: AnyRouter;
}) {
  const [router] = useState(() => given ?? createTestRouter(undefined, initialEntries));
  return <RouterContextProvider router={router}>{children}</RouterContextProvider>;
}

/** A page of the test's own at `path`, as `<Route path element>` drew one. */
export interface TestRoute {
  path: string;
  component: RouteComponent;
  /** What the route tells the app about itself (the shell's frame). */
  staticData?: StaticDataRouteOption;
}

/**
 * Renders the router at `initialEntries` over the given pages, for a test that needs real
 * matching: params read with useParams, a navigation that lands on another page. Resolves once
 * the first page has loaded, so assertions can follow synchronously.
 */
export async function renderAtRoutes(
  routes: readonly TestRoute[],
  {
    initialEntries = ["/"],
    wrapper: Wrapper,
    layout = Outlet,
  }: {
    initialEntries?: readonly string[];
    /** Providers drawn around the router. */
    wrapper?: ComponentType<{ children: ReactNode }>;
    /** A layout every page sits in; it renders `<Outlet />` where the page goes. */
    layout?: RouteComponent;
  } = {},
): Promise<RenderResult & { router: AnyRouter }> {
  const rootRoute = createRootRoute({ component: layout });
  const routeTree = rootRoute.addChildren(
    routes.map(({ path, component, staticData }) =>
      createRoute({ getParentRoute: () => rootRoute, path, component, staticData }),
    ),
  );
  const router = createTestRouter(routeTree, initialEntries);
  await router.load();
  const ui = <RouterProvider router={router} />;
  const result = render(Wrapper ? <Wrapper>{ui}</Wrapper> : ui);
  return Object.assign(result, { router });
}
