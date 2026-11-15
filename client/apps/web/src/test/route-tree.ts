import { routeTree } from "@/router";
import type { RouteGateContext } from "@trenova/shared/lib/router/gates";
import { enterContext } from "@trenova/shared/testing/gate-context";
import { createTestRouter } from "@trenova/shared/testing/test-router";
import { isRedirect, type AnyRoute } from "@tanstack/react-router";

const router = createTestRouter(routeTree);

/** Whether the app registers a page at `fullPath` (params written `$id`). */
export function hasRoute(fullPath: string): boolean {
  return fullPath in router.routesByPath;
}

/** The app's route at `fullPath`; fails the test when there is none. */
export function routeAt(fullPath: string): AnyRoute {
  const route = (router.routesByPath as Record<string, AnyRoute | undefined>)[fullPath];
  if (!route) {
    throw new Error(`route ${fullPath} is not registered`);
  }
  return route;
}

/** The app's route with id `routeId` (a pathless layout: `/app`, `/deskShell`). */
export function routeById(routeId: string): AnyRoute {
  const route = (router.routesById as Record<string, AnyRoute | undefined>)[routeId];
  if (!route) {
    throw new Error(`route id ${routeId} is not registered`);
  }
  return route;
}

/**
 * Whether a concrete pathname opens a page of its own, params included, rather than falling to
 * the signed-in catch-all (the section index or not-found).
 */
export function servesPage(pathname: string): boolean {
  const [, , foundRoute] = router.getMatchedRoutes(pathname);
  return foundRoute !== undefined && !foundRoute.fullPath.endsWith("$");
}

/** The components of the layouts a route sits in, outermost first. */
export function layoutsAbove(fullPath: string): unknown[] {
  const layouts: unknown[] = [];
  for (
    let route = routeAt(fullPath).parentRoute as AnyRoute | undefined;
    route;
    route = route.parentRoute as AnyRoute | undefined
  ) {
    if (route.options.component) layouts.unshift(route.options.component);
  }
  return layouts;
}

/** Every full path the app registers. */
export function registeredPaths(): string[] {
  return Object.keys(router.routesByPath);
}

/**
 * Runs the gates of the route at `fullPath` as a first visit to `pathname` + `search` would,
 * and returns what they threw (a redirect, a RouteRefusal, a failure), or undefined when they
 * let the person in.
 */
export async function gateOutcome(
  fullPath: string,
  options: { pathname?: string; search?: string; params?: Record<string, string> } = {},
): Promise<unknown> {
  return runGatesOf(routeAt(fullPath), { pathname: fullPath, ...options });
}

/** gateOutcome for a layout, named by its route id. */
export async function gateOutcomeById(
  routeId: string,
  options: { pathname: string; search?: string; params?: Record<string, string> },
): Promise<unknown> {
  return runGatesOf(routeById(routeId), options);
}

async function runGatesOf(
  route: AnyRoute,
  {
    pathname,
    search = "",
    params = {},
  }: { pathname: string; search?: string; params?: Record<string, string> },
): Promise<unknown> {
  const beforeLoad = route.options.beforeLoad as
    | ((context: RouteGateContext) => unknown)
    | undefined;
  if (!beforeLoad) {
    return undefined;
  }
  try {
    await beforeLoad(enterContext(pathname, { search, params }));
    return undefined;
  } catch (error) {
    return error;
  }
}

/** Where a gate's redirect sends the person, or null when it did not redirect. */
export function redirectTarget(outcome: unknown): string | null {
  if (!isRedirect(outcome)) {
    return null;
  }
  return (
    outcome.options.href ?? (typeof outcome.options.to === "string" ? outcome.options.to : null)
  );
}
