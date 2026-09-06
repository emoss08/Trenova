import {
  WORKER_DISCIPLINARY_ACTIONS_KEY,
  WORKER_DISCIPLINARY_LADDER_KEY,
  WORKER_RECOGNITIONS_KEY,
  WORKER_SAFETY_EVENTS_KEY,
  WORKER_SAFETY_SCORECARD_KEY,
} from "@/lib/graphql/worker-safety";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

const KEYS = [
  WORKER_SAFETY_EVENTS_KEY,
  WORKER_SAFETY_SCORECARD_KEY,
  WORKER_DISCIPLINARY_ACTIONS_KEY,
  WORKER_DISCIPLINARY_LADDER_KEY,
  WORKER_RECOGNITIONS_KEY,
];

export function useSafetyInvalidation(workerId: string) {
  const queryClient = useQueryClient();
  return useCallback(
    () =>
      Promise.all([
        ...KEYS.map((key) => queryClient.invalidateQueries({ queryKey: [key, workerId] })),
        queryClient.invalidateQueries({ queryKey: ["worker"] }),
        queryClient.invalidateQueries({ queryKey: ["worker-employment-events", workerId] }),
      ]),
    [queryClient, workerId],
  );
}
