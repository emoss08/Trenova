import { RecentActivityDocument, type RecentActivityQuery } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { queries } from "@/lib/queries";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";

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

export function useRecentActivityInfinite(pageSize = RECENT_ACTIVITY_DEFAULT_PAGE_SIZE) {
  const canReadAuditLog = usePermissionStore((state) =>
    state.hasPermission(Resource.AuditLog, Operation.Read),
  );

  return useInfiniteQuery({
    queryKey: [...queries.attention.recentActivity.queryKey, pageSize],
    initialPageParam: null as string | null,
    queryFn: async ({ pageParam }) =>
      requestGraphQL({
        document: RecentActivityDocument,
        operationName: "RecentActivity",
        variables: { first: pageSize, after: pageParam },
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
    select: (data) => data.pages.flatMap((page) => page.auditEntries.edges.map((edge) => edge.node)),
  });
}
