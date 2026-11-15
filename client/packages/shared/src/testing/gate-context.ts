import type { RouteGateContext } from "@trenova/shared/lib/router/gates";

/** What a gate is told on a first visit to `pathname` with `search` (written `?a=b`). */
export function enterContext(
  pathname: string,
  { search = "", params = {} }: { search?: string; params?: Record<string, string> } = {},
): RouteGateContext {
  return {
    location: { pathname, searchStr: search },
    params,
    cause: "enter",
    abortController: new AbortController(),
  };
}
