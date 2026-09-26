import { API_BASE_URL } from "./constants";

/** The versioned prefix the server writes into every path it hands out. */
const SERVER_API_PREFIX = "/api/v1";

/**
 * A path the server returned (a page's content, a webhook) as a URL the
 * browser can load. The server writes paths under its own `/api/v1`; the app
 * may be pointed at an API on another origin, so the prefix is swapped for the
 * configured base rather than resolved against the page.
 */
export function apiUrl(serverPath: string): string {
  const relative = serverPath.startsWith(SERVER_API_PREFIX)
    ? serverPath.slice(SERVER_API_PREFIX.length)
    : serverPath;

  return `${API_BASE_URL}${relative.startsWith("/") ? relative : `/${relative}`}`;
}
