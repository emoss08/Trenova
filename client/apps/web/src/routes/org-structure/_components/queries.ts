import {
  APPROVAL_DELEGATIONS_KEY,
  fetchApprovalDelegations,
  fetchHeadcount,
  fetchJobPositions,
  HEADCOUNT_KEY,
  JOB_POSITIONS_KEY,
} from "@/lib/graphql/org-structure";

export function headcountQuery() {
  return {
    queryKey: [HEADCOUNT_KEY] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchHeadcount({ signal }),
  };
}

export function jobPositionsQuery() {
  return {
    queryKey: [JOB_POSITIONS_KEY] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchJobPositions(undefined, { signal }),
  };
}

/**
 * The two sides of cover. Naming one side and leaving the other open is what
 * keeps them apart: the server refuses a pair that names neither side as the
 * signed-in user, so neither view can read somebody else's arrangements.
 */
export function delegationsGivenQuery(userId: string) {
  return {
    queryKey: [APPROVAL_DELEGATIONS_KEY, "given", userId] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchApprovalDelegations({ delegatorId: userId }, { signal }),
  };
}

export function delegationsReceivedQuery(userId: string) {
  return {
    queryKey: [APPROVAL_DELEGATIONS_KEY, "received", userId] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchApprovalDelegations({ delegateId: userId }, { signal }),
  };
}
