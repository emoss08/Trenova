import type { InfiniteData, UseInfiniteQueryResult } from "@tanstack/react-query";
import { useMemo } from "react";

/** One page of a list a card reads, and how many there are in all, read with the first page. */
export type RowsPage<T> = { rows: T[]; total: number | null; totalExact?: boolean };

/** A list a card reads a page at a time, as its section draws it. */
export type PagedRows<T> = {
  /** Every page read so far; undefined until the first page lands. */
  rows: T[] | undefined;
  /** How many there are in all, from the first page. */
  total: number | null;
  /** False when total is the server's estimate of a list too large to count. */
  totalExact: boolean;
  hasMore: boolean;
  loadingMore: boolean;
  loadMore: () => void;
  pending: boolean;
  error: unknown;
};

/** The pages an infinite query has read, as one list with its count and the way to the next page. */
export function usePagedRows<T, TPage extends RowsPage<T>>(
  query: UseInfiniteQueryResult<InfiniteData<TPage, unknown>, unknown>,
): PagedRows<T> {
  const rows = useMemo(() => query.data?.pages.flatMap((page) => page.rows), [query.data]);
  const { fetchNextPage } = query;
  return {
    rows,
    total: query.data?.pages[0]?.total ?? null,
    totalExact: query.data?.pages[0]?.totalExact !== false,
    hasMore: query.hasNextPage,
    loadingMore: query.isFetchingNextPage,
    loadMore: () => void fetchNextPage(),
    pending: query.isPending,
    error: query.error,
  };
}
