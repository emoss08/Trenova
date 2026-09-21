import { queries } from "@/lib/queries";
import { fetchAgentActivityCounts } from "@/lib/graphql/agent-activity";
import { useQuery } from "@tanstack/react-query";

const COUNTS_QUERY_KEY = ["ai-control", "activity-counts"] as const;

/**
 * The numbers on the overview. Providers come from the same list the
 * Providers tab renders so both agree; the agent and activity counts are
 * `totalCount`-only connection queries, which the server answers with a
 * COUNT and no rows.
 */
export function useAIControlStats() {
  const providersQuery = useQuery(queries.aiProvider.list());
  const countsQuery = useQuery({
    queryKey: COUNTS_QUERY_KEY,
    queryFn: ({ signal }) => fetchAgentActivityCounts({ signal }),
    refetchInterval: 60_000,
  });

  const usageQuery = useQuery({
    ...queries.aiProvider.usage(USAGE_WINDOW_DAYS),
    refetchInterval: 60_000,
  });

  const providers = providersQuery.data ?? [];

  return {
    isLoading: providersQuery.isLoading || countsQuery.isLoading,
    providersTotal: providers.length,
    providersEnabled: providers.filter((provider) => provider.enabled).length,
    counts: countsQuery.data,
    usage: usageQuery.data,
    usageLoading: usageQuery.isLoading,
    usageWindowDays: USAGE_WINDOW_DAYS,
  };
}

/** The overview reads a week: long enough to smooth a quiet weekend. */
export const USAGE_WINDOW_DAYS = 7;

export const aiControlStatsQueryKey = COUNTS_QUERY_KEY;
