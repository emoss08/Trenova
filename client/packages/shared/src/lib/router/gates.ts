import { announceSignedIn } from "@trenova/shared/lib/session-context";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { redirect } from "@tanstack/react-router";

/** What a gate is told about the navigation it stands in front of. */
export interface RouteGateContext {
  readonly location: { readonly pathname: string; readonly searchStr: string };
  readonly params: Readonly<Record<string, string | undefined>>;
  readonly cause: "preload" | "enter" | "stay";
  /** Aborted when the navigation is superseded, for a gate that asks the server. */
  readonly abortController: AbortController;
}

/**
 * A check a route runs before its page loads (its `beforeLoad`). It lets the navigation on by
 * returning, sends the person elsewhere by throwing `redirect`, and refuses the page by
 * throwing a RouteRefusal, which the route's error component presents.
 */
export type RouteGate = (context: RouteGateContext) => void | Promise<void>;

/**
 * Runs gates in order, the first to throw deciding, and only when the path changes. A page
 * that keeps its state in the search (Triage's open message, a table's filters) would
 * otherwise run the session and permission checks on every keypress, and a quick run of
 * them exhausts the per-user rate limit. A route stays matched while a page below it changes
 * (the app shell under every signed-in page), so its gates run again for each new path: the
 * session is checked on every change of page, as it was when each page carried the check.
 * Neither check depends on the search, and a 401 on any request still ends the session.
 */
export function gates(...list: readonly RouteGate[]): RouteGate {
  let passedFor: string | null = null;

  return async (context) => {
    const { pathname } = context.location;
    if (context.cause === "stay" && passedFor === pathname) {
      return;
    }

    for (const gate of list) {
      await gate(context);
    }

    if (context.cause !== "preload") {
      passedFor = pathname;
    }
  };
}

/**
 * The session gate a signed-in route runs. Only a session the server refused (a 401) goes to
 * the login form. A check that could not reach a verdict — a rate limit, a server error, a
 * dropped connection — leaves a working session standing mid-navigation, so the page's own
 * queries retry, and on a cold load, where nothing proves a session, throws the original
 * failure to the route's error component, which presents it with a retry. A session that
 * passes has reached the app, so a tab still on a sign-in page is told it can follow.
 */
export function requireSession(loginPath: string): RouteGate {
  return async () => {
    const session = await useAuthStore.getState().checkSession();

    switch (session.status) {
      case "signed_in":
        announceSignedIn();
        return;
      case "signed_out":
        throw redirect({ href: loginPath });
      case "unreachable":
        if (useAuthStore.getState().isAuthenticated) {
          return;
        }
        throw session.error;
    }
  };
}

/** Sends somebody already signed in away from a guest page (the sign-in form) to `home`. */
export function requireGuest(home: string): RouteGate {
  return async () => {
    if (await useAuthStore.getState().checkAuth()) {
      throw redirect({ href: home });
    }
  };
}

/** An address that moved: always sends the person on to `href`. */
export function redirectTo(href: string): RouteGate {
  return () => {
    throw redirect({ href, replace: true });
  };
}

/**
 * An address that moved, for links already sent (stored notifications, bookmarks): sends the
 * person on to `path` with the query they came with.
 */
export function redirectKeepingSearch(path: string): RouteGate {
  return ({ location }) => {
    throw redirect({ href: `${path}${location.searchStr}`, replace: true });
  };
}
