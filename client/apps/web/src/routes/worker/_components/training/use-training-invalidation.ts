import { WORKER_TRAINING_KEY, WORKER_TRAINING_SUMMARY_KEY } from "@/lib/graphql/worker-training";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

export function useTrainingInvalidation(workerId: string) {
  const queryClient = useQueryClient();
  return useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: [WORKER_TRAINING_KEY, workerId] }),
        queryClient.invalidateQueries({ queryKey: [WORKER_TRAINING_SUMMARY_KEY, workerId] }),
        queryClient.invalidateQueries({ queryKey: ["worker"] }),
      ]),
    [queryClient, workerId],
  );
}
