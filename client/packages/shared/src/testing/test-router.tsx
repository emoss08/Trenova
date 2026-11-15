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

/**
 * A router over `routeTree` that starts at `initialEntries`, reading the query as the app does.
 * `context` is what the app's router hands its routes (its query client), for a test that runs
 * the routes' loaders.
 */
export function createTestRouter(
  routeTree: AnyRoute = createRootRoute({ component: Outlet }),
  initialEntries: readonly string[] = ["/"],
  context?: Record<string, unknown>,
): AnyRouter {
  return createRouter({
    routeTree,
    context,
    history: createMemoryHistory({ initialEntries: [...initialEntries] }),
    parseSearch,
    stringifySearch,
  });
}

/** An address the component under test wrote, as `onUrlUpdate` reports it. */
export interface UrlUpdateEvent {
  searchParams: URLSearchParams;
  queryString: string;
  options: { history: "push" | "replace" };
}

export type OnUrlUpdateFunction = (event: UrlUpdateEvent) => void;

/** A query to start at: "?a=1", "a=1", a record or URLSearchParams. */
export type TestSearch = string | Readonly<Record<string, string>> | URLSearchParams;

function withSearch(entry: string, search: TestSearch | undefined): string {
  if (search === undefined) return entry;
  const query = new URLSearchParams(
    typeof search === "string" ? search.replace(/^\?/, "") : search,
  ).toString();
  const path = entry.split("?")[0] ?? "/";
  return query ? `${path}?${query}` : path;
}

export interface TestRouterProps {
  children: ReactNode;
  initialEntries?: readonly string[];
  /** The query the first entry starts with, in place of any it names. */
  search?: TestSearch;
  /** Called with every address the router writes, as it writes it. */
  onUrlUpdate?: OnUrlUpdateFunction;
  router?: AnyRouter;
}

/**
 * The router a component under test reads: links, navigation, the location, the history and
 * the state a page keeps in the address work, and nothing is matched or rendered by the router
 * itself, so `children` render on the first pass. `router` is for a test that inspects where
 * the component navigated (`router.state.location`).
 */
export function TestRouter({
  children,
  initialEntries = ["/"],
  search,
  onUrlUpdate,
  router: given,
}: TestRouterProps) {
  const [router] = useState(() => {
    const entries = [...initialEntries];
    entries[0] = withSearch(entries[0] ?? "/", search);
    const created = given ?? createTestRouter(undefined, entries);
    if (onUrlUpdate) {
      // From the router's own event, not a history listener: a router drawn without a
      // RouterProvider loads itself only while nothing else listens to its history.
      const entryIndex = () => created.history.location.state.__TSR_index ?? 0;
      let lastIndex = entryIndex();
      created.subscribe("onBeforeLoad", ({ toLocation }) => {
        const index = entryIndex();
        const history = index > lastIndex ? "push" : "replace";
        lastIndex = index;
        onUrlUpdate({
          searchParams: new URLSearchParams(toLocation.searchStr),
          queryString: toLocation.searchStr,
          options: { history },
        });
      });
    }
    return created;
  });
  return <RouterContextProvider router={router}>{children}</RouterContextProvider>;
}

/** A `wrapper` for renderHook that puts the hook under a TestRouter. */
export function withTestRouter(props: Omit<TestRouterProps, "children"> = {}) {
  return function TestRouterWrapper({ children }: { children: ReactNode }) {
    return <TestRouter {...props}>{children}</TestRouter>;
  };
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
