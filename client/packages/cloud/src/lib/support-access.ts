import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { AccessMode } from "../types/support-access";

export const SUPPORT_CONSOLE_PATH = "/support-console";
export const SUPPORT_ACCESS_ADMIN_PATH = "/admin/support-access";
export const SUPPORT_SESSION_POLL_MS = 30_000;

const MINUTE = 60;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** How long is left, in the coarse units a banner reads at a glance. */
export function formatRemaining(seconds: number, t: TranslateFn): string {
  if (seconds <= 0) {
    return t("now");
  }
  if (seconds < MINUTE) {
    return t("under a minute");
  }
  if (seconds < HOUR) {
    return t("{0}m", Math.ceil(seconds / MINUTE));
  }
  if (seconds < DAY) {
    const hours = Math.floor(seconds / HOUR);
    const minutes = Math.floor((seconds % HOUR) / MINUTE);
    return minutes > 0 ? t("{0}h {1}m", hours, minutes) : t("{0}h", hours);
  }
  const days = Math.floor(seconds / DAY);
  const hours = Math.floor((seconds % DAY) / HOUR);
  return hours > 0 ? t("{0}d {1}h", days, hours) : t("{0}d", days);
}

/**
 * Seconds until `expiresAt`, measured on the server's clock: the response carries the
 * server time it was produced at, so a skewed client clock does not move the deadline.
 */
export function secondsUntil(
  expiresAt: number,
  serverTime: number,
  fetchedAtSeconds: number,
  nowSeconds: number,
): number {
  const elapsed = Math.max(nowSeconds - fetchedAtSeconds, 0);
  return expiresAt - (serverTime + elapsed);
}

export function accessModeLabel(mode: AccessMode, t: TranslateFn): string {
  return mode === "read_write" ? t("Read-write") : t("Read-only");
}

export function durationLabel(hours: number, t: TranslateFn): string {
  if (hours % 24 === 0) {
    return t("{0, plural, one {# day} other {# days}}", hours / 24);
  }
  return t("{0, plural, one {# hour} other {# hours}}", hours);
}

export function endReasonLabel(reason: string, t: TranslateFn): string {
  switch (reason) {
    case "exited":
      return t("Staff member exited");
    case "expired":
      return t("Time limit reached");
    case "grant_revoked":
      return t("Access revoked");
    case "grant_expired":
      return t("Access expired");
    case "staff_removed":
      return t("No longer Trenova staff");
    case "signed_out":
      return t("Staff member signed out");
    case "replaced":
      return t("Replaced by a newer session");
    case "assurance_lost":
      return t("Two-factor sign-in lapsed");
    default:
      return reason;
  }
}
