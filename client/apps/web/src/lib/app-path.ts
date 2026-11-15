import type { RegisteredRouter, ValidateFromPath } from "@tanstack/react-router";

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

type WithoutTrailingSlash<P extends string> = P extends "/"
  ? P
  : P extends `${infer Head}/`
    ? Head
    : P;

/**
 * Every page path the router registers, params written `$name` ("/hr/workers",
 * "/desk/r/$kind/$id"). `ValidateFromPath` with nothing to validate is TanStack's
 * `RoutePaths` for the registered tree: the full path of every route, whether the tree
 * is built in code or generated from files. An index route's trailing slash is dropped,
 * so a page reads the same either way.
 */
export type AppRoutePath = Exclude<
  WithoutTrailingSlash<ValidateFromPath<RegisteredRouter, never>>,
  ""
>;

/** A registered path with no params. */
export type StaticAppPath = Exclude<AppRoutePath, `${string}$${string}`>;

/** A registered path with at least one `$param` segment. */
export type ParamAppPath = Extract<AppRoutePath, `${string}$${string}`>;

/** The names of the `$param` segments in a path pattern. */
export type AppPathParamNames<P extends string> = P extends `${string}$${infer Name}/${infer Rest}`
  ? Name | AppPathParamNames<`/${Rest}`>
  : P extends `${string}$${infer Name}`
    ? Name
    : never;

export type AppPathParams<P extends string> = { readonly [K in AppPathParamNames<P>]: string };

const PARAM = /\$([A-Za-z0-9_]+)/g;

/**
 * The address of a page the router registers, as a plain string. A path the router does
 * not have fails to typecheck, so renaming or removing a route breaks the build rather
 * than the link. A pattern takes its params, each encoded as one path segment; any query
 * is appended by the caller after this.
 */
export function appPath<P extends StaticAppPath>(path: P): P;
export function appPath<P extends ParamAppPath>(path: P, params: AppPathParams<P>): string;
export function appPath(path: string, params?: Readonly<Record<string, string>>): string {
  if (params === undefined) {
    return path;
  }
  return path.replace(PARAM, (_, name: string) => encodeURIComponent(params[name] ?? ""));
}
