import type {
  DetentionIntelligence,
  DetentionIntelligenceInput,
} from "@/lib/graphql/detention-intelligence";
import { detention } from "@/lib/queries/detention";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

/**
 * How long a read stays fresh. The window is whole days of the organization's time zone, so a
 * read only changes as stops inside it settle; reopening a scorecard or switching facilities
 * within this reuses the read instead of reading again.
 */
export const INTELLIGENCE_STALE_MS = 5 * 60_000;

/** The read the page and its route prefetch share, so the prefetched result is the one used. */
export function intelligenceInput(days: number): DetentionIntelligenceInput {
  return { days };
}

/** Detention intelligence over the last `days` whole days, ending today in the organization's zone. */
export function useDetentionIntelligence(days: number) {
  return useQuery({
    ...detention.intelligence(intelligenceInput(days)),
    staleTime: INTELLIGENCE_STALE_MS,
    placeholderData: keepPreviousData,
  });
}

/**
 * The same window narrowed to one facility, for its scorecard and its arrival grid. It asks for
 * the days the page's read came back with, so it covers the same window even past midnight, and
 * reads nothing until a facility is chosen.
 */
export function useFacilityIntelligence(page: DetentionIntelligence, locationId: string | null) {
  return useQuery({
    ...detention.intelligence({
      days: page.days,
      through: page.through,
      locationId: locationId ?? undefined,
    }),
    enabled: locationId !== null,
    staleTime: INTELLIGENCE_STALE_MS,
  });
}
