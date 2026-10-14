import { edi } from "@/lib/queries/edi";
import { useQuery } from "@tanstack/react-query";

export const EDI_SUMMARY_REFETCH_INTERVAL = 30_000;

export function useEDISummary(sinceHours?: number) {
  return useQuery({
    ...edi.summary(sinceHours),
    refetchInterval: EDI_SUMMARY_REFETCH_INTERVAL,
  });
}

export function useEDIPartnerScorecards(sinceHours?: number) {
  return useQuery({
    ...edi.partnerScorecards(sinceHours),
    refetchInterval: EDI_SUMMARY_REFETCH_INTERVAL,
  });
}

export function useEDIVolumeSeries(sinceHours?: number) {
  return useQuery({
    ...edi.volumeSeries(sinceHours),
    refetchInterval: EDI_SUMMARY_REFETCH_INTERVAL,
  });
}
