import { WORKER_CHECKLISTS_KEY } from "@/lib/graphql/worker-checklist";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

/**
 * Checklist writes can flip the worker's DQF readiness and always move the
 * open-checklist count on the worker row, so the worker queries refresh too.
 */
export function useChecklistInvalidation(workerId: string) {
  const queryClient = useQueryClient();
  return useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [WORKER_CHECKLISTS_KEY, workerId] }),
      queryClient.invalidateQueries({ queryKey: ["worker"] }),
      queryClient.invalidateQueries({ queryKey: ["worker-list"] }),
    ]);
  }, [queryClient, workerId]);
}
