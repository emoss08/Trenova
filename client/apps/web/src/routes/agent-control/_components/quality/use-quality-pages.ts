import { SECTION_TABLE_PAGE_SIZE } from "@/components/data-table/section-table";
import { useCursorPages } from "@/hooks/use-cursor-pages";
import type { QualityPage, QualityPageRequest } from "@/lib/graphql/agent-quality";
import {
  keepPreviousData,
  useQuery,
  type QueryFunction,
  type QueryKey,
} from "@tanstack/react-query";
import { useEffect, useState } from "react";

/** A section's figures move with the nightly sweep; a minute old is still true. */
export const QUALITY_STALE_MS = 60_000;

type PagedQueryOptions<T, TKey extends QueryKey> = (page: QualityPageRequest) => {
  queryKey: TKey;
  queryFn: QueryFunction<QualityPage<T>, TKey>;
};

/**
 * One server-paged section: the page it is on, the cursor that page is read
 * after, and what the server said about the page it read. The count is asked
 * for only with a scope's first page, and a new scope starts over.
 */
export function useQualityPages<T, TKey extends QueryKey>(
  scopeKey: string,
  options: PagedQueryOptions<T, TKey>,
) {
  const [pageSize, setPageSize] = useState<number>(SECTION_TABLE_PAGE_SIZE);
  const pages = useCursorPages(`${scopeKey}:${pageSize}`);
  const { pageIndex, recordPage } = pages;

  const query = useQuery({
    ...options({ first: pageSize, after: pages.after, includeTotalCount: pageIndex === 0 }),
    placeholderData: keepPreviousData,
    staleTime: QUALITY_STALE_MS,
  });

  const landed = query.data && !query.isPlaceholderData ? query.data : undefined;
  useEffect(() => {
    if (!landed) {
      return;
    }
    recordPage({
      pageIndex,
      endCursor: landed.endCursor,
      hasNextPage: landed.hasNextPage,
      totalCount: landed.totalCount,
    });
  }, [landed, pageIndex, recordPage]);

  return {
    query,
    rows: query.data?.items,
    pagination: {
      mode: "cursor" as const,
      pageIndex,
      pageSize,
      hasNextPage: query.data?.hasNextPage ?? false,
      totalCount: pages.totalCount,
      onPageChange: pages.goToPage,
      onPageSizeChange: setPageSize,
    },
  };
}
