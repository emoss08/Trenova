import {
  fetchProfileChangeRequests,
  PROFILE_CHANGE_REQUESTS_KEY,
  type ProfileChangeRequestRow,
} from "@/lib/graphql/self-service";

/** How many of a worker's latest requests the page reads: every waiting one and recent decisions. */
export const WORKER_PROFILE_CHANGE_LIMIT = 20;

/**
 * A worker's latest profile change requests, the one read the Profile alert, the review pane and
 * Files › Driver portal share.
 */
export function workerProfileChangesQuery(workerId: string) {
  return {
    queryKey: [PROFILE_CHANGE_REQUESTS_KEY, workerId] as const,
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchProfileChangeRequests({ workerId, limit: WORKER_PROFILE_CHANGE_LIMIT }, { signal }),
  };
}

/** The requests still waiting for a decision, oldest first, so the alert opens the one waited longest. */
export function pendingProfileChanges(
  rows: readonly ProfileChangeRequestRow[] | undefined,
): ProfileChangeRequestRow[] {
  return (rows ?? [])
    .filter((row) => row.status === "Pending")
    .sort((a, b) => a.submittedAt - b.submittedAt);
}
