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
  count: DeskShelfCount;
  /** The conversations read on this shelf so far, newest first. */
  threads: readonly AssistantThread[];
  paging: DeskShelfPaging;
};

/** One case on the rail: a record, what it needs, and the conversation asked on it. */
export type DeskRailCase = {
  to: string;
  /** The record the row is a case of; empty for a view. */
  recordId: string;
  title: string;
  /** A plan or a single change about the record waits on a person. */
  waiting: boolean;
  /** Something an agent composed, opened on the canvas, rather than a record's case. */
  view: boolean;
  /** The person's conversation on the case, when they have asked on it. */
  thread: AssistantThread | null;
};

/**
 * One line of the rail's list: a shelf's heading, a conversation, a case, or
 * the place the rest of a shelf arrives.
 */
export type DeskRailRow =
  | { kind: "shelf"; key: DeskShelfKey }
  | { kind: "thread"; thread: AssistantThread }
  | { kind: "case"; case: DeskRailCase }
  | { kind: "more"; shelf: DeskShelfKey; loading: boolean; failed: boolean };

/**
 * Whether a shelf has a heading: it holds something, by its count or by what
 * was read of it. A shelf whose count could not be read keeps its heading.
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
 */
export function deskRailRows(shelves: readonly DeskRailShelf[]): DeskRailRow[] {
  const rows: DeskRailRow[] = [];
  for (const shelf of shelves) {
    if (!shelfShown(shelf)) {
      continue;
    }
    rows.push({ kind: "shelf", key: shelf.key });
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
 * A row's identity across renders, which the list keys its rows by.
 */
export function deskRailRowKey(row: DeskRailRow): string {
  switch (row.kind) {
    case "shelf":
      return `s:${row.key}`;
    case "thread":
      return `c:${row.thread.id}`;
    case "case":
      return `r:${row.case.to}`;
    default:
      return `more:${row.shelf}`;
  }
}
