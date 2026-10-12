import type { AssistantThread } from "@/types/assistant";
import type { DeskShelfKey } from "./desk-threads";

/**
 * How many conversations a shelf holds as the server counted them; still
 * being counted; or not known, because the count could not be read.
 */
export type DeskShelfCount = number | "counting" | "unknown";

/** Where reading a shelf stands. */
export type DeskShelfPaging = {
  /** Its first page, or its next one, is being read. */
  loading: boolean;
  /** More of it remains to be read. */
  hasMore: boolean;
  /** The last page asked for could not be read; it is asked for again only on request. */
  failed: boolean;
};

/** One shelf of the rail with what has been read of it. */
export type DeskRailShelf = {
  key: DeskShelfKey;
  /** Folded away: its heading stays, its conversations go, and none are read. */
  collapsed: boolean;
  count: DeskShelfCount;
  /** The conversations read on this shelf so far, newest first. */
  threads: readonly AssistantThread[];
  paging: DeskShelfPaging;
};

/**
 * One line of the rail's conversation list: a shelf's heading, a
 * conversation, or the place the rest of a shelf arrives.
 */
export type DeskRailRow =
  | { kind: "shelf"; key: DeskShelfKey; collapsed: boolean; count: DeskShelfCount }
  | { kind: "thread"; thread: AssistantThread }
  | { kind: "more"; shelf: DeskShelfKey; loading: boolean; failed: boolean };

/**
 * Whether a shelf has a heading: it holds something, by its count or by what
 * was read of it. A shelf whose count could not be read keeps its heading,
 * so it can still be opened.
 */
function shelfShown(shelf: DeskRailShelf): boolean {
  if (shelf.threads.length > 0) {
    return true;
  }

  return typeof shelf.count === "number" ? shelf.count > 0 : shelf.count === "unknown";
}

/**
 * The shelves laid out as rows, each heading above its conversations and the
 * place the rest of it arrives while more remains or a page is being read.
 *
 * A folded shelf keeps its heading and drops its conversations, all but the
 * open one, so the place the reader is in never disappears from the rail;
 * nothing more of it is read until it opens again.
 */
export function deskRailRows(
  shelves: readonly DeskRailShelf[],
  activeThreadId: string | null,
): DeskRailRow[] {
  const rows: DeskRailRow[] = [];
  for (const shelf of shelves) {
    if (!shelfShown(shelf)) {
      continue;
    }
    rows.push({ kind: "shelf", key: shelf.key, collapsed: shelf.collapsed, count: shelf.count });
    if (shelf.collapsed) {
      const open = shelf.threads.find((thread) => thread.id === activeThreadId);
      if (open) {
        rows.push({ kind: "thread", thread: open });
      }
      continue;
    }
    for (const thread of shelf.threads) {
      rows.push({ kind: "thread", thread });
    }
    const { loading, hasMore, failed } = shelf.paging;
    if (loading || hasMore || failed) {
      rows.push({ kind: "more", shelf: shelf.key, loading, failed });
    }
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
      return `more:${row.shelf}`;
  }
}
