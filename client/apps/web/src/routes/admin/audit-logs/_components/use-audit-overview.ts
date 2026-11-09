import { audit } from "@/lib/queries/audit";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

/**
 * The overview reads rollups the worker refreshes every minute, so reading it more often shows
 * nothing new; every write in the organization would otherwise refetch it.
 */
export const OVERVIEW_REFRESH_MS = 60_000;

export function useAuditOverview(days: number) {
  return useQuery({
    ...audit.overview(days),
    staleTime: OVERVIEW_REFRESH_MS,
    refetchInterval: OVERVIEW_REFRESH_MS,
    placeholderData: keepPreviousData,
  });
}
