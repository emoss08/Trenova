import { WORKER_LEAVE_KEY } from "@/lib/graphql/worker-leave";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

/**
 * The entitlement is derived from the cases and their days, so any change to
 * either changes the balance. The employment timeline is invalidated too: a
 * leave of absence shows there as well.
 */
export function useLeaveInvalidation(workerId: string) {
  const queryClient = useQueryClient();
  return useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: [WORKER_LEAVE_KEY, workerId] }),
        queryClient.invalidateQueries({ queryKey: ["worker-employment-events", workerId] }),
        queryClient.invalidateQueries({ queryKey: ["worker-overview", workerId] }),
      ]),
    [queryClient, workerId],
  );
}
