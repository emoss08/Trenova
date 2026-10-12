import type { CaseSummary } from "@/types/assistant";

/**
 * Where a case stands, read against now: a snooze that has run out reads as
 * the state under it, because the server only says so on its next read.
 */
export function caseStateOf(summary: CaseSummary, now: number): CaseSummary["state"] {
  if (summary.state === "Snoozed" && (summary.snoozedUntil ?? 0) <= now) {
    return summary.openWaits > 0 ? "Waiting" : "Working";
  }

  return summary.state;
}
