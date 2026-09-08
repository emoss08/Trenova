import {
  BENEFIT_COSTS_KEY,
  BENEFIT_ENROLLMENT_LIST_KEY,
  BENEFIT_PLANS_KEY,
  fetchBenefitCosts,
  fetchBenefitEnrollments,
  fetchBenefitPlans,
} from "@/lib/graphql/benefits";

/** Every plan on file, all years: the year picker is built from it. */
export function benefitPlansQuery() {
  return {
    queryKey: [BENEFIT_PLANS_KEY] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchBenefitPlans(undefined, { signal }),
  };
}

export function benefitCostsQuery(planYear: number | null) {
  return {
    queryKey: [BENEFIT_COSTS_KEY, planYear] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchBenefitCosts(planYear ?? undefined, { signal }),
  };
}

/** Cover that is arranged or running: what is starting and what is ending. */
export function openEnrollmentsQuery() {
  return {
    queryKey: [BENEFIT_ENROLLMENT_LIST_KEY, "open"] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchBenefitEnrollments({ openOnly: true }, { signal }),
  };
}

const RECENT_DECLINES = 25;

export function declinedEnrollmentsQuery() {
  return {
    queryKey: [BENEFIT_ENROLLMENT_LIST_KEY, "declined"] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchBenefitEnrollments({ statuses: ["Waived"], limit: RECENT_DECLINES }, { signal }),
  };
}

/** Everyone who has ever been put on one plan, whatever became of it. */
export function planEnrollmentsQuery(planId: string) {
  return {
    queryKey: [BENEFIT_ENROLLMENT_LIST_KEY, "plan", planId] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchBenefitEnrollments({ planId }, { signal }),
  };
}
