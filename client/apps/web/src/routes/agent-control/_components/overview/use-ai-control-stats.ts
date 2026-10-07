import { queries } from "@/lib/queries";
import { fetchAgentActivityCounts } from "@/lib/graphql/agent-activity";
import { type QueryClient, useQuery } from "@tanstack/react-query";

const COUNTS_QUERY_KEY = ["ai-control", "activity-counts"] as const;

/**
 * The numbers on the rail beside every tab. Providers come from the same list the
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


/**
 * Refreshes what counts the organization's agent work: the rail's figures and the facts
 * each tab's sentence is made of. Call it after anything that decides, adds or removes work.
 */
export function invalidateAIControlCounts(queryClient: QueryClient): Promise<unknown> {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: COUNTS_QUERY_KEY }),
    queryClient.invalidateQueries({ queryKey: queries.aiControl._def }),
  ]);
}
