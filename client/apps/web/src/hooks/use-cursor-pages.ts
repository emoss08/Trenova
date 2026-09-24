import { useCallback, useState } from "react";

/**
 * Where a cursor-paged list is, for one scope (a filter, a search, a page
 * size). The cursor for page n is the end cursor of page n - 1, learned as
 * each page lands, so the pages already read can be walked back and forth
 * without the server offering an offset.
 */
export type CursorPages = {
  scopeKey: string;
  pageIndex: number;
  cursors: Readonly<Record<number, string | null>>;
  /** Read with the first page of the scope; null until it lands. */
  totalCount: number | null;
};

export type CursorPageLanded = {
  pageIndex: number;
  endCursor: string | null;
  hasNextPage: boolean;
  totalCount: number | null;
};

export function initialCursorPages(scopeKey: string): CursorPages {
  return { scopeKey, pageIndex: 0, cursors: { 0: null }, totalCount: null };
}

/** The same state when nothing new was learned, so a render is not spent on it. */
export function recordCursorPage(state: CursorPages, page: CursorPageLanded): CursorPages {
  const nextIndex = page.pageIndex + 1;
  const nextCursor = page.hasNextPage && page.endCursor ? page.endCursor : undefined;
  const totalCount = page.totalCount ?? state.totalCount;
  const cursorKnown = nextCursor === undefined || state.cursors[nextIndex] === nextCursor;
  if (cursorKnown && totalCount === state.totalCount) {
    return state;
  }

  return {
    ...state,
    cursors: cursorKnown ? state.cursors : { ...state.cursors, [nextIndex]: nextCursor },
    totalCount,
  };
}

/** A page is reachable only once the page before it has said where it ends. */
export function canReachPage(state: CursorPages, pageIndex: number): boolean {
  return pageIndex >= 0 && state.cursors[pageIndex] !== undefined;
}

export function moveToPage(state: CursorPages, pageIndex: number): CursorPages {
  if (pageIndex === state.pageIndex || !canReachPage(state, pageIndex)) {
    return state;
  }

  return { ...state, pageIndex };
}

export type UseCursorPages = {
  pageIndex: number;
  /** The cursor the current page is read after; null for the first page. */
  after: string | null;
  totalCount: number | null;
  goToPage: (pageIndex: number) => void;
  recordPage: (page: CursorPageLanded) => void;
};

/**
 * Cursor pages that start over whenever the scope changes: a new search or
 * filter goes back to the first page and forgets the cursors of the old one.
 */
export function useCursorPages(scopeKey: string): UseCursorPages {
  const [state, setState] = useState(() => initialCursorPages(scopeKey));
  const scoped = state.scopeKey === scopeKey ? state : initialCursorPages(scopeKey);

  const goToPage = useCallback(
    (pageIndex: number) => {
      setState((current) =>
        moveToPage(
          current.scopeKey === scopeKey ? current : initialCursorPages(scopeKey),
          pageIndex,
        ),
      );
    },
    [scopeKey],
  );

  const recordPage = useCallback(
    (page: CursorPageLanded) => {
      setState((current) =>
        recordCursorPage(
          current.scopeKey === scopeKey ? current : initialCursorPages(scopeKey),
          page,
        ),
      );
    },
    [scopeKey],
  );

  return {
    pageIndex: scoped.pageIndex,
    after: scoped.cursors[scoped.pageIndex] ?? null,
    totalCount: scoped.totalCount,
    goToPage,
    recordPage,
  };
}
