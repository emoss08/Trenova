import {
  CREDENTIAL_EXPIRY_FORECAST_KEY,
  WORKER_CREDENTIAL_SUMMARY_KEY,
  WORKER_CREDENTIALS_KEY,
} from "@/lib/graphql/worker-credential";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

/**
 * Every credential write moves the worker's summary, the credential list, the
 * org-wide forecast, and — because the profile columns are mirrored — the
 * worker record itself.
 */
export function useCredentialInvalidation(workerId: string) {
  const queryClient = useQueryClient();
  return useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [WORKER_CREDENTIAL_SUMMARY_KEY, workerId] }),
      queryClient.invalidateQueries({ queryKey: [WORKER_CREDENTIALS_KEY, workerId] }),
      queryClient.invalidateQueries({ queryKey: [CREDENTIAL_EXPIRY_FORECAST_KEY] }),
      queryClient.invalidateQueries({ queryKey: ["worker"] }),
      queryClient.invalidateQueries({ queryKey: ["worker-list"] }),
    ]);
  }, [queryClient, workerId]);
}
