import { useAuthStore } from "@trenova/shared/stores/auth-store";
import {
  redirect,
  type LoaderFunction,
  type RouteObject,
  type ShouldRevalidateFunction,
} from "react-router";

/**
 * The session gate a signed-in route's loader runs. Only a session the server refused (a
 * 401) goes to the login form. A check that could not reach a verdict — a rate limit, a
 * server error, a dropped connection — leaves a working session standing mid-navigation, so
 * the page's own queries retry, and on a cold load, where nothing proves a session, throws
 * the original failure to the route's error boundary, which presents it with a retry.
 */
export async function requireSession(loginPath: string): Promise<Response | null> {
  const session = await useAuthStore.getState().checkSession();

  switch (session.status) {
    case "signed_in":
      return null;
    case "signed_out":
      return redirect(loginPath);
    case "unreachable":
      if (useAuthStore.getState().isAuthenticated) {
        return null;
      }
      throw session.error;
  }
}

/**
 * Skips a revalidation that stays on the same path, and defers to the router for a new
 * path or a form submission. A page that keeps its state in the search — Triage's open
 * message, a table's filters — would otherwise re-run the session and permission checks on
 * every keypress, and a quick run of them exhausts the per-user rate limit. A navigation to
 * the URL already shown is skipped too: keys pressed faster than navigations commit produce
 * one each, and the router cannot tell them from revalidate(). Neither check depends on
 * the search, and a 401 on any request still ends the session.
 */
export const revalidateOnPathChange: ShouldRevalidateFunction = ({
  currentUrl,
  nextUrl,
  formMethod,
  actionResult,
  defaultShouldRevalidate,
}) => {
  const samePath =
    formMethod === undefined &&
    actionResult === undefined &&
    currentUrl.pathname === nextUrl.pathname;

  return samePath ? false : defaultShouldRevalidate;
};

const pathScopedLoaders = new WeakSet<LoaderFunction>();

/**
 * Marks a loader whose answer depends on the path and the session, never on the search,
 * such as the session gate and the permission gate. A route carrying one gets
 * `revalidateOnPathChange` from `withPathScopedRevalidation`.
 */
export function pathScopedLoader<T extends LoaderFunction>(loader: T): T {
  pathScopedLoaders.add(loader);
  return loader;
}

export function isPathScopedLoader(loader: unknown): boolean {
  return typeof loader === "function" && pathScopedLoaders.has(loader as LoaderFunction);
}

function withRevalidation(route: RouteObject): RouteObject {
  const next: RouteObject = { ...route };

  if (next.shouldRevalidate === undefined && isPathScopedLoader(next.loader)) {
    next.shouldRevalidate = revalidateOnPathChange;
  }

  if (route.children !== undefined) {
    next.children = route.children.map(withRevalidation);
  }

  return next;
}

/**
 * Returns the route tree with `revalidateOnPathChange` on every route whose loader is
 * path-scoped and that does not decide its own revalidation. Applied to the whole tree, so
 * a route added later is covered without anybody remembering to.
 */
export function withPathScopedRevalidation(routes: readonly RouteObject[]): RouteObject[] {
  return routes.map(withRevalidation);
}
