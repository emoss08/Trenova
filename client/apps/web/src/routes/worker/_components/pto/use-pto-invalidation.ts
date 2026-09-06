import { queries } from "@/lib/queries";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

export const PTO_TABLE_QUERY_KEY = "worker-pto-list";
export const WORKER_TABLE_QUERY_KEY = "worker-list";

export function ptoQueryKeys(): readonly (readonly string[])[] {
  return [
    [PTO_TABLE_QUERY_KEY],
    [...queries.worker.listUpcomingPTO._def],
    [...queries.worker.ptoChartData._def],
    [WORKER_TABLE_QUERY_KEY],
  ];
}

export async function invalidatePTOQueries(queryClient: QueryClient): Promise<void> {
  await Promise.all(
    ptoQueryKeys().map((queryKey) =>
      queryClient.invalidateQueries({ queryKey: [...queryKey], refetchType: "all" }),
    ),
  );
}

export function usePTOInvalidation() {
  const queryClient = useQueryClient();
  return useCallback(() => invalidatePTOQueries(queryClient), [queryClient]);
}
