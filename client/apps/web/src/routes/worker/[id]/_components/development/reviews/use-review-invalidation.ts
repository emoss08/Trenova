import { WORKER_REVIEWS_KEY } from "@/lib/graphql/performance-review";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

export function useReviewInvalidation(workerId: string) {
  const queryClient = useQueryClient();
  return useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: [WORKER_REVIEWS_KEY, workerId] }),
        queryClient.invalidateQueries({ queryKey: ["worker"] }),
      ]),
    [queryClient, workerId],
  );
}
