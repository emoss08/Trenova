import type { CredentialHealth } from "../types/worker-credential";

export type CredentialHealthTone = "ok" | "soon" | "overdue" | "missing";

export type CredentialHealthMeta = {
  label: string;
  tone: CredentialHealthTone;
  badgeVariant: "active" | "warning" | "inactive" | "outline";
  textClass: string;
  ringClass: string;
  dotClass: string;
  /** Weight used to sort the worst slot first. */
  rank: number;
};

const HEALTH_META: Record<CredentialHealth, CredentialHealthMeta> = {
  Missing: {
    label: "Missing",
    tone: "missing",
    badgeVariant: "outline",
    textClass: "text-muted-foreground",
    ringClass: "border-dashed border-muted-foreground/40",
    dotClass: "bg-muted-foreground/60",
    rank: 0,
  },
  Expired: {
    label: "Expired",
    tone: "overdue",
    badgeVariant: "inactive",
    textClass: "text-red-600 dark:text-red-400",
    ringClass: "border-red-500/40 bg-red-500/5",
    dotClass: "bg-red-500",
    rank: 1,
  },
  ExpiringSoon: {
    label: "Expiring soon",
    tone: "soon",
    badgeVariant: "warning",
    textClass: "text-amber-600 dark:text-amber-400",
    ringClass: "border-amber-500/40 bg-amber-500/5",
    dotClass: "bg-amber-500",
    rank: 2,
  },
  Valid: {
    label: "Valid",
    tone: "ok",
    badgeVariant: "active",
    textClass: "text-green-600 dark:text-green-400",
    ringClass: "border-border",
    dotClass: "bg-green-500",
    rank: 3,
  },
};

export function credentialHealthMeta(health: CredentialHealth): CredentialHealthMeta {
  return HEALTH_META[health] ?? HEALTH_META.Missing;
}

/**
 * Human phrase for a days-until-expiry value. Mirrors the server's whole-day
 * arithmetic: 0 is "today", negatives are already past.
 */
export function describeDaysUntil(days: number | null | undefined): string {
  if (days == null) return "No expiry";
  if (days === 0) return "Expires today";
  if (days === 1) return "Expires tomorrow";
  if (days === -1) return "Expired yesterday";
  if (days < 0) return `Expired ${-days} days ago`;
  return `${days} days left`;
}

/**
 * Expiry suggested by a credential type's validity: the issue date plus N
 * calendar months, clamped to the last day of a shorter month so Jan 31 + 1
 * month is Feb 28, never Mar 3. Works in UTC dates like the server's day math.
 */
export function suggestExpiryUnix(
  issuedAt: number | null | undefined,
  validityMonths: number | null | undefined,
): number | null {
  if (issuedAt == null || issuedAt <= 0 || validityMonths == null || validityMonths <= 0) {
    return null;
  }
  const issued = new Date(issuedAt * 1000);
  const year = issued.getUTCFullYear();
  const monthIndex = issued.getUTCMonth() + validityMonths;
  const lastDayOfTarget = new Date(Date.UTC(year, monthIndex + 1, 0)).getUTCDate();
  const day = Math.min(issued.getUTCDate(), lastDayOfTarget);
  return Date.UTC(year, monthIndex, day) / 1000;
}

/**
 * Compact chip text: "Today", "3d", "-2d". Used where a full sentence would
 * not fit.
 */
export function shortDaysUntil(days: number | null | undefined): string {
  if (days == null) return "—";
  if (days === 0) return "Today";
  return `${days}d`;
}
