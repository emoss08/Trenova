/**
 * Whether a link is a path inside this app: one leading slash, then anything.
 *
 * A protocol-relative `//host/x` and a `/\host` (which browsers read as the
 * same thing) start with a slash too, and both leave the app, so neither is
 * one. Anything the app follows or routes on its own — a link in a reply, a
 * navigation the assistant asked for — has to pass this first.
 */
export function isAppPath(href: string | null | undefined): href is string {
  return (
    typeof href === "string" &&
    href.startsWith("/") &&
    !href.startsWith("//") &&
    !href.startsWith("/\\")
  );
}
