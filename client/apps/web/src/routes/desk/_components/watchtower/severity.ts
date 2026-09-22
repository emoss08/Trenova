import type { WatchtowerSeverity } from "@/lib/graphql/watchtower";

/**
 * A severity is an ordering, so it is a tone rather than a categorical
 * accent: info reads quieter than warning, warning quieter than critical, and
 * a reader scanning the feed is sorting by loudness whether they mean to or
 * not.
 */
export const SEVERITY_TONE: Record<WatchtowerSeverity, "info" | "warning" | "danger"> = {
  Info: "info",
  Warning: "warning",
  Critical: "danger",
};

/** How loudly each severity ranks, for sorting a set of them. */
export const SEVERITY_ORDER: Record<WatchtowerSeverity, number> = {
  Critical: 0,
  Warning: 1,
  Info: 2,
};

export const ALL_SEVERITIES: WatchtowerSeverity[] = ["Critical", "Warning", "Info"];

/**
 * Whether an item arrived after the reader's cursor.
 *
 * The server already answers this per item, but a feed that has just been
 * marked seen has to stop drawing the line without refetching every row —
 * so the comparison lives here too, against whatever cursor the client
 * currently holds.
 */
export function isUnseen(occurredAt: number, seenAt: number): boolean {
  return occurredAt > seenAt;
}

/**
 * Toggles one value in a filter set.
 *
 * An empty set means "everything", which is also what the server takes, so
 * turning the last chip off is the same as turning them all on rather than
 * asking for nothing.
 */
export function toggleFilter<T>(current: readonly T[], value: T): T[] {
  return current.includes(value)
    ? current.filter((candidate) => candidate !== value)
    : [...current, value];
}
