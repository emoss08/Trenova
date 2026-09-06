import { queries } from "@/lib/queries";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { PTOChartDataPoint, PTOType, Worker } from "@trenova/shared/types/worker";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { buildApprovedPTOMetrics, type ApprovedPTOMetrics } from "./approved-pto-metrics";

export type ApprovedPTOAnalyticsParams = {
  startDate: number;
  endDate: number;
  type?: string;
  workerId?: Worker["id"];
  fleetCodeId?: string;
};

const ANALYTICS_STALE_TIME_MS = 5 * 60 * 1000;
const ANALYTICS_GC_TIME_MS = 10 * 60 * 1000;

/** The month's approved days, bucketed for the chart and the KPI cards. */
export function approvedPtoChartQuery(
  { startDate, endDate, type, workerId }: ApprovedPTOAnalyticsParams,
  timezone: string | undefined,
) {
  return {
    ...queries.worker.ptoChartData({
      startDateFrom: startDate,
      startDateTo: endDate,
      type: type as PTOType,
      workerId,
      timezone,
    }),
    staleTime: ANALYTICS_STALE_TIME_MS,
    gcTime: ANALYTICS_GC_TIME_MS,
  };
}

/**
 * Only the count of open requests is needed here, so the page size is one; the
 * requested overview beside it pages through the same filter on its own key.
 */
export function requestedPtoCountQuery(
  { startDate, endDate, type, workerId, fleetCodeId }: ApprovedPTOAnalyticsParams,
  timezone: string | undefined,
) {
  return {
    ...queries.worker.listUpcomingPTO({
      filter: {
        limit: 1,
        after: null,
      },
      type: type as PTOType,
      status: "Requested",
      startDate,
      endDate,
      workerId,
      fleetCodeId,
      timezone,
    }),
    staleTime: ANALYTICS_STALE_TIME_MS,
    gcTime: ANALYTICS_GC_TIME_MS,
  };
}

export type ApprovedPTOAnalyticsState = {
  chartData: PTOChartDataPoint[];
  chartLoading: boolean;
  chartError: boolean;
  chartErrorMessage?: string;
  requestedCount: number;
  requestedLoading: boolean;
  requestedError: boolean;
  metrics: ApprovedPTOMetrics;
};

export function useApprovedPTOAnalytics(
  params: ApprovedPTOAnalyticsParams,
): ApprovedPTOAnalyticsState {
  const timezone = useAuthStore((state) => state.user?.timezone);

  const chartQuery = useQuery(approvedPtoChartQuery(params, timezone));

  const requestedQuery = useQuery(requestedPtoCountQuery(params, timezone));

  const chartData = useMemo(() => chartQuery.data ?? [], [chartQuery.data]);

  const metrics = useMemo(() => buildApprovedPTOMetrics(chartData), [chartData]);

  return {
    chartData,
    chartLoading: chartQuery.isLoading,
    chartError: chartQuery.isError,
    chartErrorMessage: chartQuery.error?.message,
    requestedCount: requestedQuery.data?.count ?? 0,
    requestedLoading: requestedQuery.isLoading,
    requestedError: requestedQuery.isError,
    metrics,
  };
}
