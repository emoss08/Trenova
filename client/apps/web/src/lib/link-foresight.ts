import { containsRecordId } from "@/lib/record-ids";
import { routeRequest } from "@/service-worker/routing";
import type { PredictingRouter } from "@trenova/shared/lib/router/link-foresight";
import { whenPageSettles } from "@trenova/shared/lib/when-page-settles";

const NO_HEADERS = { get: () => null };

/**
 * Whether a same-origin URL is a page of the app: the same rule the service worker answers
 * navigations by, so the API, GraphQL, Dash and files are never treated as pages.
 */
export function isAppPage(url: URL): boolean {
  const request = { method: "GET", url: url.href, mode: "navigate", headers: NO_HEADERS };
  return routeRequest(request, window.location.origin, import.meta.env.BASE_URL) === "shell";
}

/**
 * Starts pages ahead of their links (`watchLinks`) once the page has loaded and the browser
 * is idle; the prediction library is imported only then, so none of it is in the
 * startup bundle. If it cannot start, links still preload on hover and focus.
 */
export function predictLinksWhenIdle(router: PredictingRouter): void {
  // The dev server serves a page as a thousand separate modules, so a predicted page's modules
  // queue behind the open page's own and make it slower; built, a preload is a few chunks.
  if (import.meta.env.DEV) return;
  whenPageSettles(() => {
    import("@trenova/shared/lib/router/link-foresight")
      .then(({ watchLinks }) => {
        watchLinks({ router, isAppPage, namesRecord: (url) => containsRecordId(url.search) });
      })
      .catch((error: unknown) =>
        console.error("Could not start link prediction; links still preload on hover:", error),
      );
  });
}
