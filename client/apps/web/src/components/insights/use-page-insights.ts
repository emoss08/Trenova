import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { sortInsights } from "@/routes/home/_components/widgets/insight-presentation";
import type { Insight, InsightSurface } from "@/types/insight";
import { useQuery } from "@tanstack/react-query";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const PAGE_INSIGHTS_LIMIT = 5;

/**
 * The findings that belong on one working page.
 *
 * The permission is checked here rather than left to the request so a reader
 * without it sees no card at all, not a card that says "forbidden". The server
 * still filters by what the reader may see; this only decides whether to ask.
 */
export function usePageInsights(
  surface: InsightSurface,
  limit = PAGE_INSIGHTS_LIMIT,
): { insights: Insight[]; isLoading: boolean; allowed: boolean } {
  const { allowed } = usePermission(Resource.Insight, Operation.Read);

  const query = useQuery({
    ...queries.insight.active(limit, surface),
    enabled: allowed,
  });

  return {
    insights: sortInsights(query.data?.results ?? []),
    isLoading: allowed && query.isLoading,
    allowed,
  };
}
