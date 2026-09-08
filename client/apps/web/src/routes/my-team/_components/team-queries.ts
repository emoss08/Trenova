import {
  APPROVAL_DELEGATIONS_KEY,
  fetchApprovalDelegations,
  fetchMyTeam,
  MY_TEAM_KEY,
} from "@/lib/graphql/org-structure";

export function myTeamQuery(includeInactive: boolean) {
  return {
    queryKey: [MY_TEAM_KEY, includeInactive] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchMyTeam(includeInactive, { signal }),
  };
}

/**
 * The delegations in force for the signed-in user. Naming the delegate side
 * keeps the read to their own arrangements, which is all this page may see.
 */
export function coverQuery(userId: string) {
  return {
    queryKey: [APPROVAL_DELEGATIONS_KEY, "received", "active", userId] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchApprovalDelegations({ delegateId: userId, activeOnly: true }, { signal }),
  };
}
