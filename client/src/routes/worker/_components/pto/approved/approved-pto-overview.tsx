import { useQueryStates } from "nuqs";
import React, { lazy } from "react";
import { usePTOFilters } from "../use-pto-filters";
import { ptoSearchParamsParser } from "../use-pto-state";
import { ApprovedPTOHeader } from "./approved-pto-header";
import { ApprovedChartBoundary } from "./chart/approved-chart-state";
import { ApprovedPTOKPICards } from "./chart/approved-pto-kpi-cards";
import { useApprovedPTOAnalytics } from "./chart/use-approved-pto-analytics";

const ApprovedPTOChart = lazy(() => import("./chart/approved-pto-chart"));

export function ApprovedPTOOverview() {
  const { defaultValues } = usePTOFilters();
  const [searchParams] = useQueryStates(ptoSearchParamsParser);
  const filters = searchParams.ptoOverviewFilters ?? defaultValues;
  const analytics = useApprovedPTOAnalytics({
    startDate: filters.startDate,
    endDate: filters.endDate,
    type: filters.type ?? undefined,
    workerId: filters.workerId ?? undefined,
  });

  throw new Error("Test error");

  return (
    <OverviewOuter>
      <ApprovedPTOHeader />
      <OverviewInner>
        <ApprovedPTOKPICards
          metrics={analytics.metrics}
          requestedCount={analytics.requestedCount}
          chartLoading={analytics.chartLoading}
          requestedLoading={analytics.requestedLoading}
          requestedError={analytics.requestedError}
        />
        <ApprovedChartBoundary>
          <ApprovedPTOChart
            data={analytics.chartData}
            isLoading={analytics.chartLoading}
            isError={analytics.chartError}
            errorMessage={analytics.chartErrorMessage}
          />
        </ApprovedChartBoundary>
      </OverviewInner>
    </OverviewOuter>
  );
}

function OverviewOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-3 flex-col gap-1">{children}</div>;
}

function OverviewInner({ children }: { children: React.ReactNode }) {
  return <div className="flex-1 rounded-md border border-border p-3">{children}</div>;
}
