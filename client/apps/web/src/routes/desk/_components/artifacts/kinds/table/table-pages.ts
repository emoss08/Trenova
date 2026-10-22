import type { ArtifactRows } from "@/types/assistant";

/** The page sizes a person picks between; the first is the default. */
export const PAGE_SIZES = [25, 50] as const;
export type PageSize = (typeof PAGE_SIZES)[number];

export function isPageSize(value: unknown): value is PageSize {
  return PAGE_SIZES.some((size) => size === value);
}

/** Where a read of a list starts: after a cursor, at an offset, or from its start. */
export type RowsPosition = { after?: string; offset?: number; first: boolean };

/** Where the list goes on from a page it read, or null when the page was its last. */
export function positionAfter(page: ArtifactRows): RowsPosition | null {
  if (!page.hasMore) {
    return null;
  }
  if (page.nextCursor !== "") {
    return { after: page.nextCursor, first: false };
  }
  if (page.nextOffset !== null) {
    return { offset: page.nextOffset, first: false };
  }

  return null;
}

/** The rows of a page, from the rows read so far. */
export function pageSlice<T>(rows: readonly T[], page: number, size: number): T[] {
  return rows.slice(page * size, (page + 1) * size);
}

/**
 * How many pages a list fills. From its total when that is known, or from the
 * rows read when nothing follows them; otherwise the rows read so far and at
 * least one more page, which is not yet a count.
 */
export function pageCount({
  total,
  read,
  hasMore,
  size,
}: {
  total: number | null;
  read: number;
  hasMore: boolean;
  size: number;
}): { pages: number; exact: boolean } {
  if (total !== null) {
    return { pages: Math.max(1, Math.ceil(total / size)), exact: true };
  }
  const filled = Math.max(1, Math.ceil(read / size));
  if (!hasMore) {
    return { pages: filled, exact: true };
  }

  return { pages: read % size === 0 ? filled + 1 : filled, exact: false };
}

/** How much of a page is rows the agent read: all of it, part of it, or none. */
export function agentShare(
  page: number,
  size: number,
  agentRows: number,
  shownRows: number,
): "all" | "part" | "none" {
  const start = page * size;
  if (agentRows <= start || shownRows === 0) {
    return "none";
  }

  return agentRows >= start + shownRows ? "all" : "part";
}

/** How many pages of the person's size the agent's rows fill, counting a part as one. */
export function agentPages(agentRows: number, size: number): number {
  return Math.ceil(agentRows / size);
}

/** One segment of the strip of pages under a table. */
export type RailPage = {
  index: number;
  current: boolean;
  /** Rows the agent read are on it. */
  agent: boolean;
  /** Its rows have been read, by the agent or the person. */
  read: boolean;
  /** Past what the person may read. */
  locked: boolean;
};

/** The most segments the strip draws; a longer list shows those around the page open. */
export const RAIL_MOST = 12;

/**
 * The strip of pages: one segment per page, the one open, the ones whose
 * rows the agent read and the ones read since. A list of more pages than the
 * strip holds shows the window around the open page.
 */
export function railPages({
  pages,
  current,
  size,
  agentRows,
  readRows,
  lockedFrom,
}: {
  pages: number;
  current: number;
  size: number;
  agentRows: number;
  readRows: number;
  /** The first page the person may not read, or null when they may read every page. */
  lockedFrom: number | null;
}): RailPage[] {
  const shown = Math.min(pages, RAIL_MOST);
  const first = Math.min(Math.max(0, current - Math.floor(shown / 2)), pages - shown);

  return Array.from({ length: shown }, (_, offset) => {
    const index = first + offset;
    return {
      index,
      current: index === current,
      agent: agentRows > index * size,
      read: readRows > index * size,
      locked: lockedFrom !== null && index >= lockedFrom,
    };
  });
}
