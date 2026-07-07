import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";

export const EDI_SUMMARY_REFETCH_INTERVAL = 30_000;

export function useEDISummary() {
  return useQuery({
    ...queries.edi.summary(),
    refetchInterval: EDI_SUMMARY_REFETCH_INTERVAL,
  });
}
