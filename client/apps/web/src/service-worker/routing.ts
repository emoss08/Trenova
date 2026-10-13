// Which requests the service worker answers, and how. Everything it does not name it leaves
// to the network untouched: the API, GraphQL and their event streams, Dash (which has its
// own worker), anything another origin serves, and any file whose name is not a content
// hash. It never sees a person's data, only the app's own code and its HTML shell.

/** What the worker does with a request. */
export type Route =
  /** A content-hashed build file: served from the cache, fetched once on a miss. */
  | "asset"
  /** A page of the app: answered with the cached HTML shell. */
  | "shell"
  /** Not the worker's: the browser fetches it as if there were no worker. */
  | "network";

/** The parts of a Request the routing reads, so it can be tested without a browser. */
export type RoutedRequest = {
  method: string;
  url: string;
  mode: string;
  headers: { get: (name: string) => string | null };
};

const NEVER_HANDLED = ["api/", "graphql", "dash/", ".well-known/"];
const FILE_EXTENSION = /\.[A-Za-z0-9]+$/;

/**
 * Routes a request made under `base` (the app's mount path, "/" for the web app) on
 * `origin`. Fonts live under assets/ without a content hash, so they are left to HTTP
 * caching, where a new release can replace them.
 */
export function routeRequest(request: RoutedRequest, origin: string, base: string): Route {
  if (request.method !== "GET") return "network";
  if (request.headers.get("accept")?.includes("text/event-stream")) return "network";

  const url = new URL(request.url);
  if (url.origin !== origin || !url.pathname.startsWith(base)) return "network";

  const path = url.pathname.slice(base.length);
  if (
    NEVER_HANDLED.some((prefix) => path === prefix.replace(/\/$/, "") || path.startsWith(prefix))
  ) {
    return "network";
  }

  if (path.startsWith("assets/")) {
    return path.startsWith("assets/fonts/") ? "network" : "asset";
  }

  if (request.mode === "navigate" && !FILE_EXTENSION.test(path)) return "shell";
  return "network";
}

/** The parts of a Response the worker reads to decide whether to keep it. */
export type CachedResponse = {
  status: number;
  type: string;
  headers: { get: (name: string) => string | null };
};

/**
 * Whether a response for a build file may be kept. A host that serves the app as a
 * single-page app can answer a missing file with index.html and a 200 (Cloudflare's static
 * assets do); kept, that HTML would stand in for the file until the next release.
 */
export function isCacheableAsset(response: CachedResponse): boolean {
  if (response.status !== 200 || response.type !== "basic") return false;
  return !(response.headers.get("content-type") ?? "").includes("text/html");
}
