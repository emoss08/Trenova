/**
 * The layouts every signed-in or guest page stands in, by route id. They are drawn on the first
 * load before any page, so the router plugin leaves them in the entry (vite/file-routes.ts) and
 * the page-access reader counts what they import as the shell every screen already runs.
 */
export const ROUTE_SHELLS = ["/_app", "/_desk", "/_onboarding", "/_guest"];
