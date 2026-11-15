import { useCallback, useState } from "react";

/** What a page of a cursor-paged connection says about the pages after it. */
export interface CursorPageInfo {
  mode?: string;
  hasNextPage?: boolean;
  endCursor?: string | null;
  totalCount?: number | null;
  totalCountExact?: boolean | null;
}

type KnownPages = {
  scopeKey: string;
  cursors: Readonly<Record<number, string | null>>;
  totalCount: number | null;
  totalCountExact: boolean;
};

/** The first page of any scope: no cursor, and no total known yet. */
export const FIRST_PAGE_CURSOR = null;

const NO_PAGES: KnownPages = {
  scopeKey: "",
  cursors: { 0: FIRST_PAGE_CURSOR },
  totalCount: null,
  totalCountExact: true,
};

export interface CursorPages {
  /** The cursor the page at `pageIndex` starts after; undefined when it is not known yet. */
  cursor: string | null | undefined;
  /** Whether the page can be asked for: the first page always, a later one once reached. */
  canFetch: boolean;
  /** The last total the server sent for this scope; it sends one with the first page only. */
  totalCount: number | null;
  totalCountExact: boolean;
  /** Records what the page now showing said: where the next one starts, and any total. */
  learn: (pageInfo: CursorPageInfo | null | undefined) => void;
}

/**
 * The cursors of a cursor-paged table, learned page by page as the person moves forward: each
 * page names where the next one starts. They belong to one scope (`scopeKey`: the query,
 * filters, sort and page size), so changing any of those starts again from the first page.
 */
export function useCursorPages(scopeKey: string, pageIndex: number): CursorPages {
  const [pages, setPages] = useState<KnownPages>(NO_PAGES);
  const scoped = pages.scopeKey === scopeKey ? pages : NO_PAGES;

  const learn = useCallback(
    (pageInfo: CursorPageInfo | null | undefined) => {
      if (pageInfo?.mode !== "cursor") return;

      const nextPageIndex = pageIndex + 1;
      const nextCursor = pageInfo.hasNextPage && pageInfo.endCursor ? pageInfo.endCursor : null;
      const pageTotalCount = pageInfo.totalCount ?? null;
      const pageTotalExact = pageInfo.totalCountExact !== false;
      setPages((current) => {
        const inScope = current.scopeKey === scopeKey;
        const known = inScope ? current : NO_PAGES;
        const totalCount = pageTotalCount ?? known.totalCount;
        const totalCountExact = pageTotalCount !== null ? pageTotalExact : known.totalCountExact;
        const cursorKnown = nextCursor === null || known.cursors[nextPageIndex] === nextCursor;
        if (
          inScope &&
          cursorKnown &&
          totalCount === known.totalCount &&
          totalCountExact === known.totalCountExact
        ) {
          return current;
        }

        return {
          scopeKey,
          cursors: cursorKnown ? known.cursors : { ...known.cursors, [nextPageIndex]: nextCursor },
          totalCount,
          totalCountExact,
        };
      });
    },
    [scopeKey, pageIndex],
  );

  const cursor = scoped.cursors[pageIndex];
  return {
    cursor,
    canFetch: pageIndex === 0 || cursor !== undefined,
    totalCount: scoped.totalCount,
    totalCountExact: scoped.totalCountExact,
    learn,
  };
}
