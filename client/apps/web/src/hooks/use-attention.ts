import {
  RecentActivityDocument,
  type RecentActivityQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { ATTENTION_ROWS_BY_KEY, type AttentionRowConfig } from "@/config/attention-rows";
import { useSidebarPreferences } from "@/hooks/use-sidebar-preferences";
import { queries } from "@/lib/queries";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

const ATTENTION_REFETCH_INTERVAL = 60_000;
const RECENT_ACTIVITY_DEFAULT_PAGE_SIZE = 5;
const RECENT_ACTIVITY_MAX_PAGES = 6;

export type RecentActivityEntry = RecentActivityQuery["auditEntries"]["edges"][number]["node"];

export function useAttentionSummary() {
  return useQuery({
    ...queries.attention.summary(),
    refetchInterval: ATTENTION_REFETCH_INTERVAL,
    select: (data) => data.attentionSummary,
  });
}

export interface AttentionRow {
  row: AttentionRowConfig;
  count: number;
}

/**
 * The attention metrics a person chose to watch, paired with their counts,
 * in the order they arranged them. Rows the summary does not carry are
 * dropped rather than shown as zero.
 */
export function useAttentionRows(): { rows: AttentionRow[]; isLoading: boolean } {
  const { data: summary, isLoading } = useAttentionSummary();
  const { data: preferences } = useSidebarPreferences();
  const metrics = preferences?.attentionMetrics;

  const rows = useMemo(
    () =>
      (metrics ?? [])
        .map((key) => ATTENTION_ROWS_BY_KEY.get(key))
        .filter((row): row is AttentionRowConfig => row != null && summary?.[row.key] != null)
        .map((row) => ({ row, count: summary?.[row.key] ?? 0 })),
    [metrics, summary],
  );

  return { rows, isLoading };
}

export function useRecentActivityInfinite(pageSize = RECENT_ACTIVITY_DEFAULT_PAGE_SIZE) {
  const canReadAuditLog = usePermissionStore((state) =>
    state.hasPermission(Resource.AuditLog, Operation.Read),
  );

  return useInfiniteQuery({
    queryKey: [...queries.attention.recentActivity.queryKey, pageSize],
    initialPageParam: null as string | null,
    queryFn: async ({ pageParam, signal }) =>
      requestGraphQL({
        document: RecentActivityDocument,
        operationName: "RecentActivity",
        variables: { first: pageSize, after: pageParam },
        signal,
      }),
    getNextPageParam: (lastPage) => {
      const { hasNextPage, endCursor } = lastPage.auditEntries.pageInfo;
      return hasNextPage && endCursor ? endCursor : undefined;
    },
    maxPages: RECENT_ACTIVITY_MAX_PAGES,
    refetchInterval: ATTENTION_REFETCH_INTERVAL,
    refetchOnWindowFocus: false,
    retry: false,
    enabled: canReadAuditLog,
    select: (data) =>
      data.pages.flatMap((page) => page.auditEntries.edges.map((edge) => edge.node)),
  });
}
