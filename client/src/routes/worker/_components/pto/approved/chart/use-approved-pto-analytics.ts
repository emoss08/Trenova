import { queries } from "@/lib/queries";
import { useAuthStore } from "@/stores/auth-store";
import type { PTOChartDataPoint, PTOType, Worker } from "@/types/worker";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import {
  buildApprovedPTOMetrics,
  type ApprovedPTOMetrics,
} from "./approved-pto-metrics";

type Params = {
  startDate: number;
  endDate: number;
  type?: string;
  workerId?: Worker["id"];
};

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

export function useApprovedPTOAnalytics({
  startDate,
  endDate,
  type,
  workerId,
}: Params): ApprovedPTOAnalyticsState {
  const user = useAuthStore((state) => state.user);

  const chartQuery = useQuery({
    ...queries.worker.ptoChartData({
      startDateFrom: startDate,
      startDateTo: endDate,
      type: type as PTOType,
      workerId,
      timezone: user?.timezone,
    }),
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });

  const requestedQuery = useQuery({
    ...queries.worker.listUpcomingPTO({
      filter: {
        limit: 1,
        offset: 0,
      },
      type: type as PTOType,
      status: "Requested",
      startDate,
      endDate,
      workerId,
      timezone: user?.timezone,
    }),
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });

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
