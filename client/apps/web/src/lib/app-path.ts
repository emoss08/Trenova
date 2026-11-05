const RESOLVING_BASE = "https://app.invalid";

/**
 * Whether a link is a path inside this app: one leading slash, and the browser
 * resolves it to this origin.
 *
 * A protocol-relative `//host/x` and a `/\host` start with a slash too, and so
 * does a path with a tab or newline among its first characters, which browsers
 * drop before reading it. So the answer is what the URL parser says rather than
 * a list of prefixes. Anything the app follows or routes on its own (a link in a
 * reply, a navigation the assistant asked for) has to pass this first.
 */
export function isAppPath(href: string | null | undefined): href is string {
  if (typeof href !== "string" || !href.startsWith("/")) {
    return false;
  }
  try {
    return new URL(href, RESOLVING_BASE).origin === RESOLVING_BASE;
  } catch {
    return false;
  }
}
