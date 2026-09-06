import {
  fetchPtoBalanceSummary,
  fetchPtoLiabilityReport,
  PTO_BALANCE_SUMMARY_KEY,
  PTO_LIABILITY_REPORT_KEY,
} from "@/lib/graphql/pto-policy";

const SUMMARY_STALE_TIME_MS = 60 * 1000;

export function ptoBalanceSummaryQuery() {
  return {
    queryKey: [PTO_BALANCE_SUMMARY_KEY] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchPtoBalanceSummary({ signal }),
    staleTime: SUMMARY_STALE_TIME_MS,
  };
}

export function ptoLiabilityReportQuery() {
  return {
    queryKey: [PTO_LIABILITY_REPORT_KEY] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchPtoLiabilityReport(undefined, { signal }),
    staleTime: SUMMARY_STALE_TIME_MS,
  };
}
