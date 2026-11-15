import { useRouterState } from "@tanstack/react-router";

/** The part of the router's state that says whether it is on its way to another page. */
export interface PageChangeState {
  status: "idle" | "pending";
  location: { pathname: string };
  resolvedLocation?: { pathname: string };
}

/**
 * Whether the router is on its way to another page: a navigation is pending and its path is not
 * the one on screen. Every change of the query is a navigation too (a filter, a table's page, a
 * dialog opened over a list), and none of them is a change of page.
 */
export function isChangingPage(state: PageChangeState): boolean {
  return (
    state.status === "pending" &&
    state.resolvedLocation !== undefined &&
    state.location.pathname !== state.resolvedLocation.pathname
  );
}

export function usePageChanging(): boolean {
  return useRouterState({ select: isChangingPage });
}
