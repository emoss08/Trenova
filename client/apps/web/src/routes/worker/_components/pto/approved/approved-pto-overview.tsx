import { useQueryStates } from "nuqs";
import React, { lazy, Suspense, useCallback } from "react";
import { usePTOFilters } from "../use-pto-filters";
import {
  ptoOverviewFiltersSearchParamsParser,
  ptoViewTypeSearchParamsParser,
} from "../use-pto-state";
import { ApprovedPTOHeader } from "./approved-pto-header";
import { ApprovedChartBoundary } from "./chart/approved-chart-state";
import { ApprovedPTOKPICards } from "./chart/approved-pto-kpi-cards";
import { useApprovedPTOAnalytics } from "./chart/use-approved-pto-analytics";

const ApprovedPTOChart = lazy(() => import("./chart/approved-pto-chart"));
const PTOMonthCalendar = lazy(() =>
  import("../calendar/pto-month-calendar").then((module) => ({
    default: module.PTOMonthCalendar,
  })),
);

export function ApprovedPTOOverview() {
  const { defaultValues } = usePTOFilters();
  const [searchParams, setSearchParams] = useQueryStates(ptoOverviewFiltersSearchParamsParser);
  const [{ viewType }] = useQueryStates(ptoViewTypeSearchParamsParser);
  const filters = searchParams.ptoOverviewFilters ?? defaultValues;
  const onMonthChange = useCallback(
    (startDate: number, endDate: number) => {
      void setSearchParams({
        ptoOverviewFilters: { ...filters, startDate, endDate },
      });
    },
    [filters, setSearchParams],
  );
  const analytics = useApprovedPTOAnalytics({
    startDate: filters.startDate,
    endDate: filters.endDate,
    type: filters.type ?? undefined,
    workerId: filters.workerId ?? undefined,
    fleetCodeId: filters.fleetCodeId ?? undefined,
  });

  return (
    <OverviewOuter>
      <ApprovedPTOHeader />
      <OverviewInner>
        {viewType === "calendar" ? (
          <Suspense fallback={null}>
            <PTOMonthCalendar filters={filters} onMonthChange={onMonthChange} />
          </Suspense>
        ) : (
          <div className="flex min-h-0 flex-1 flex-col">
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
          </div>
        )}
      </OverviewInner>
    </OverviewOuter>
  );
}

function OverviewOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex min-h-0 flex-3 flex-col gap-1">{children}</div>;
}

function OverviewInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="border-border flex min-h-0 flex-1 flex-col overflow-hidden rounded-md border p-3">
      {children}
    </div>
  );
}
