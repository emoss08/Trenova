import { WORKER_DRUG_ALCOHOL_KEY } from "@/lib/graphql/worker-drug-alcohol";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

/**
 * Every write in the testing programme can move the worker's standing, and the
 * standing is what the roster, the overview tab and the dispatch check read.
 * Invalidating the worker list alongside the file keeps those from disagreeing
 * with the panel the change was made on.
 */
export function useTestingInvalidation(workerId: string) {
  const queryClient = useQueryClient();
  return useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: [WORKER_DRUG_ALCOHOL_KEY, workerId] }),
        queryClient.invalidateQueries({ queryKey: ["worker-overview", workerId] }),
        queryClient.invalidateQueries({ queryKey: ["worker-list"] }),
        queryClient.invalidateQueries({ queryKey: ["worker"] }),
      ]),
    [queryClient, workerId],
  );
}
