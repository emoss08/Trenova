import { referenceRefreshRunning } from "@/lib/accounting-sync";
import { accountingSync } from "@/lib/queries/accounting-sync";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";

const REFRESH_POLL_MS = 5000;

export function useAccountingMappingSummary(system: AccountingSystem, enabled: boolean) {
  const queryClient = useQueryClient();
  const summary = useQuery({
    ...accountingSync.mappingSummary(system),
    enabled,
    refetchInterval: (query) =>
      referenceRefreshRunning(query.state.data?.connection) ? REFRESH_POLL_MS : false,
  });

  const running = referenceRefreshRunning(summary.data?.connection);
  const wasRunning = useRef(running);
  useEffect(() => {
    if (wasRunning.current && !running) {
      void queryClient.invalidateQueries({ queryKey: accountingSync._def });
    }
    wasRunning.current = running;
  }, [queryClient, running]);

  return summary;
}
