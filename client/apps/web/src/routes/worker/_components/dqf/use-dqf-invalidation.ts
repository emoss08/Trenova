import { DRIVER_QUALIFICATION_FILE_KEY } from "@/lib/graphql/worker-dqf";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

/**
 * The qualification file is derived on read from credentials, documents,
 * investigations and the testing record, so any change to an investigation
 * changes the file — and the roster's qualified flag reads the same evidence.
 */
export function useDqfInvalidation(workerId: string) {
  const queryClient = useQueryClient();
  return useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: [DRIVER_QUALIFICATION_FILE_KEY, workerId] }),
        queryClient.invalidateQueries({ queryKey: ["outstanding-employment-verifications"] }),
        queryClient.invalidateQueries({ queryKey: ["worker-overview", workerId] }),
        queryClient.invalidateQueries({ queryKey: ["worker-list"] }),
      ]),
    [queryClient, workerId],
  );
}
