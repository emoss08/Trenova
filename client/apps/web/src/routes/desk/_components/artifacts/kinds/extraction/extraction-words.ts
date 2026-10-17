import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { FieldRow, StopGroup } from "./extraction-groups";

/** A row's id in the page, so the list can point at the selected one for a screen reader. */
export function rowDomId(listId: string, key: string): string {
  return `${listId}-${key.replace(/[^A-Za-z0-9_-]/g, "_")}`;
}

/** What a stop is called above its rows: its role and its place in the route. */
export function stopTitle(stop: Pick<StopGroup, "role" | "sequence">, t: TranslateFn): string {
  switch (stop.role) {
    case "pickup":
      return t("Pickup · stop {0}", stop.sequence);
    case "delivery":
      return t("Delivery · stop {0}", stop.sequence);
    default:
      return t("Stop {0}", stop.sequence);
  }
}

/** Why a row needs a person, in the words the row shows under its value. */
export function rowReason(row: FieldRow, t: TranslateFn): string | null {
  if (row.kind === "missing") return t("Not on the document");
  if (row.state !== "doubtful") return null;
  const { field } = row;
  switch (field.reason) {
    case "conflict":
      return field.readings.length > 1
        ? t("Two readings disagree")
        : t("Disagrees with another reading");
    case "review":
      return t("Marked for review · {0}%", Math.round(field.confidence * 100));
    default:
      return t("Low confidence · {0}%", Math.round(field.confidence * 100));
  }
}

/** What a screen reader says for a row: its name, its value and how it was read. */
export function rowAnnouncement(row: FieldRow, multiPage: boolean, t: TranslateFn): string {
  if (row.kind === "missing") return t("{0}: not on the document", row.label);
  const { field } = row;
  const percent = Math.round(field.confidence * 100);
  const how =
    row.state === "doubtful"
      ? field.reason === "conflict" && field.readings.length > 1
        ? t("readings disagree: {0}", field.readings.join(" / "))
        : t("needs a look, {0} percent confident", percent)
      : t("read with {0} percent confidence", percent);
  if (!multiPage || field.page < 1) return t("{0}: {1}, {2}", row.label, field.value, how);
  return t("{0}: {1}, {2}, page {3}", row.label, field.value, how, field.page);
}

/**
 * Why a stop needs a look, when every part read from the stop itself needs it for the same
 * reason, as a stop marked for review does; it is said once on the stop, not on each part.
 */
export function sharedStopReason(stop: StopGroup, t: TranslateFn): string | null {
  const parts = stop.rows.filter((row) => row.kind === "read" && row.field.stop !== null);
  if (parts.length < 2 || parts.some((row) => row.state !== "doubtful")) return null;
  const reasons = new Set(parts.map((row) => rowReason(row, t)));
  return reasons.size === 1 ? ([...reasons][0] ?? null) : null;
}
