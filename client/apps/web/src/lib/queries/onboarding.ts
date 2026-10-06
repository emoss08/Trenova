import { onboardingService } from "@/services/onboarding";
import { queryOptions } from "@tanstack/react-query";

const ONBOARDING_STATE_STALE_MS = 60 * 1000;

export const ONBOARDING_QUERY_ROOT = ["onboarding"] as const;

/**
 * Kept out of the merged key factory on purpose: the protected route loader reads it
 * on every navigation, and importing the factory there would pull every service into
 * the entry chunk. The key carries the organization, so a sign-out and sign-in as
 * somebody else in the same tab can never be answered from the previous session.
 */
export function onboardingStateQueryOptions(organizationId: string) {
  return queryOptions({
    queryKey: [...ONBOARDING_QUERY_ROOT, "state", organizationId] as const,
    queryFn: async ({ signal }) => onboardingService.get(signal),
    staleTime: ONBOARDING_STATE_STALE_MS,
    retry: false,
  });
}
