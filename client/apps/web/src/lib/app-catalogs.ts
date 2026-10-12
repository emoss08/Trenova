import type { CatalogBundle } from "@trenova/shared/i18n/generated/locales";

/**
 * The catalog bundles the web app's shell renders from, loaded before its first translated
 * frame. Each route folder's own bundle arrives with that folder's code (route-catalogs.ts).
 */
export const APP_CATALOGS = ["core", "web"] as const satisfies readonly CatalogBundle[];
