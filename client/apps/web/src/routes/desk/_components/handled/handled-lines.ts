import type { DeskActivityEntry } from "@/lib/graphql/desk-attention";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/** The records a line of the ledger names, the rest counted. */
export function recordLine(t: TranslateFn, entry: DeskActivityEntry): string {
  const named = entry.records.filter((record) => record.readable && record.label !== "");
  const unnamed = entry.records.length - named.length + entry.recordsMore;
  const names = named.map((record) => record.label).join(", ");
  if (names === "") {
    return unnamed > 0 ? t("{0, plural, one {# record} other {# records}}", unnamed) : "";
  }
  return unnamed > 0
    ? t("{0} and {1, plural, one {# more} other {# more}}", names, unnamed)
    : names;
}

/** Who stood behind a write: the person who approved it, or the rule it ran under. */
export function decidedLine(t: TranslateFn, entry: DeskActivityEntry): string {
  switch (entry.outcome) {
    case "Approved":
      return entry.decidedBy ? t("Approved by {0}", entry.decidedBy.name) : t("Approved");
    case "OnItsOwn":
      return t("Ran under AI control");
    case "Previewed":
      return t("Previewed, never made");
    default:
      return entry.detail;
  }
}

/** Where a line of the ledger opens: the write, with why it could run and what it changed. */
export function handledPath(entry: Pick<DeskActivityEntry, "id">): string {
  return `/desk/handled/${encodeURIComponent(entry.id)}`;
}
