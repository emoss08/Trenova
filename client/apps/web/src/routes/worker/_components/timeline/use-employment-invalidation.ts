import { WORKER_EMPLOYMENT_EVENTS_KEY } from "@/lib/graphql/worker-employment";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

/**
 * Employment events move the worker row (status, fleet, dispatchability) and,
 * on termination or rehire, the PTO book — so every write refreshes those too.
 */
export function useEmploymentInvalidation(workerId: string) {
  const queryClient = useQueryClient();
  return useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [WORKER_EMPLOYMENT_EVENTS_KEY, workerId] }),
      queryClient.invalidateQueries({ queryKey: ["worker"] }),
      queryClient.invalidateQueries({ queryKey: ["worker-list"] }),
      queryClient.invalidateQueries({ queryKey: ["worker-pto-balances", workerId] }),
      queryClient.invalidateQueries({ queryKey: ["worker-pto-assignments", workerId] }),
      queryClient.invalidateQueries({ queryKey: ["pto-history", workerId] }),
      queryClient.invalidateQueries({ queryKey: ["worker-pto-list"] }),
    ]);
  }, [queryClient, workerId]);
}
