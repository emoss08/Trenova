import { parseAsInteger, parseAsString, parseAsStringLiteral } from "nuqs";

/** The page's views: the trail, or one of the tools kept beside it. */
export const AUDIT_VIEWS = ["trail", "exports", "holds", "alerts"] as const;
export type AuditView = (typeof AUDIT_VIEWS)[number];

export const auditViewParser = parseAsStringLiteral(AUDIT_VIEWS)
  .withDefault("trail")
  .withOptions({ history: "push" });

/** The overview's windows, in whole days of the organization's time zone, today included. */
export const AUDIT_WINDOWS = [1, 7, 30, 90] as const;
export type AuditWindow = (typeof AUDIT_WINDOWS)[number];
export const DEFAULT_AUDIT_WINDOW: AuditWindow = 7;

export const auditWindowParser = parseAsInteger.withDefault(DEFAULT_AUDIT_WINDOW);

export function auditWindowOf(value: number): AuditWindow {
  return (AUDIT_WINDOWS as readonly number[]).includes(value)
    ? (value as AuditWindow)
    : DEFAULT_AUDIT_WINDOW;
}

/** The trail's quick views: server-side narrowings above the person's own filters. */
export const AUDIT_QUICK_VIEWS = ["all", "changes", "security", "agents", "deletes"] as const;
export type AuditQuickView = (typeof AUDIT_QUICK_VIEWS)[number];

export const auditQuickViewParser = parseAsStringLiteral(AUDIT_QUICK_VIEWS).withDefault("all");

/** The entry open in the inspector; alert notifications link here with `?entry=`. */
export const auditEntryParser = parseAsString;

export const AUDIT_PARAMS = {
  view: "view",
  window: "window",
  quick: "quick",
  entry: "entry",
} as const;

/** Per viewer: whether the overview is folded into its one line. */
export const OVERVIEW_FOLDED_STORAGE_KEY = "trenova:audit-log:overview-folded";

/** Per viewer: whether the trail lists one row per request while nothing narrows it. */
export const ONE_PER_REQUEST_STORAGE_KEY = "trenova:audit-log:one-per-request";
