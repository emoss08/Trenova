import type { AssistantThread } from "@/types/assistant";
import type { DeskShelfKey, DeskThreadShelf } from "./desk-threads";

/**
 * One line of the rail's conversation list: a shelf's heading, a
 * conversation, or the place the next page of conversations arrives.
 */
export type DeskRailRow =
  | { kind: "shelf"; key: DeskShelfKey }
  | { kind: "thread"; thread: AssistantThread }
  | { kind: "more" };

/** Shelves that hold cases put away rather than conversations by date. */
const PARKED_SHELVES: ReadonlySet<DeskShelfKey> = new Set(["snoozed", "settled"]);

/**
 * The shelves laid out as rows. While more conversations remain to be read,
 * the place they arrive sits at the end of the dated shelves, above the
 * snoozed and settled cases: the server pages by date, so that is where the
 * next page lands, and the list grows where the reader is looking rather
 * than above the shelves they have already passed.
 */
export function deskRailRows(
  shelves: readonly DeskThreadShelf[],
  hasMore: boolean,
): DeskRailRow[] {
  const rows: DeskRailRow[] = [];
  let morePlaced = !hasMore;
  for (const shelf of shelves) {
    if (!morePlaced && PARKED_SHELVES.has(shelf.key)) {
      rows.push({ kind: "more" });
      morePlaced = true;
    }
    rows.push({ kind: "shelf", key: shelf.key });
    for (const thread of shelf.threads) {
      rows.push({ kind: "thread", thread });
    }
  }
  if (!morePlaced) {
    rows.push({ kind: "more" });
  }

  return rows;
}

/**
 * A row's identity across renders. A conversation's is the key the rail's
 * raised card looks for, so the card and the list agree on what is current.
 */
export function deskRailRowKey(row: DeskRailRow): string {
  switch (row.kind) {
    case "shelf":
      return `s:${row.key}`;
    case "thread":
      return `c:${row.thread.id}`;
    default:
      return "more";
  }
}
