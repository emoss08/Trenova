import { QueryLazyComponent } from "@/components/error-boundary";
import { queries } from "@/lib/queries";
import { useQueryStates } from "nuqs";
import React, { Activity, lazy } from "react";
import { usePTOFilters } from "../use-pto-filters";
import { ptoSearchParamsParser } from "../use-pto-state";
import { ApprovedPTOHeader } from "./approved-pto-header";
import { ApprovedChartBoundary } from "./chart/approved-chart-state";

const PTOCalendar = lazy(() => import("./approved-pto-calendar"));
const PTOChart = lazy(() => import("./chart/approved-pto-chart"));

export function ApprovedPTOOverview() {
  const { defaultValues } = usePTOFilters();
  const [searchParams] = useQueryStates(ptoSearchParamsParser);

  return (
    <OverviewOuter>
      <ApprovedPTOHeader />
      <OverviewInner>
        <Activity
          mode={searchParams.viewType === "chart" ? "visible" : "hidden"}
        >
          <ApprovedChartBoundary>
            <PTOChart
              startDate={
                searchParams.ptoOverviewFilters?.startDate ??
                defaultValues.startDate
              }
              endDate={
                searchParams.ptoOverviewFilters?.endDate ??
                defaultValues.endDate
              }
              type={searchParams.ptoOverviewFilters?.type ?? undefined}
              workerId={searchParams.ptoOverviewFilters?.workerId ?? undefined}
            />
          </ApprovedChartBoundary>
        </Activity>
        <Activity
          mode={searchParams.viewType === "calendar" ? "visible" : "hidden"}
        >
          <QueryLazyComponent queryKey={queries.worker.getPTOCalendarData._def}>
            <PTOCalendar
              startDate={
                searchParams.ptoOverviewFilters?.startDate ??
                defaultValues.startDate
              }
              endDate={
                searchParams.ptoOverviewFilters?.endDate ??
                defaultValues.endDate
              }
              type={searchParams.ptoOverviewFilters?.type ?? undefined}
            />
          </QueryLazyComponent>
        </Activity>
      </OverviewInner>
    </OverviewOuter>
  );
}

function OverviewOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-1 flex-3">{children}</div>;
}

function OverviewInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="border border-border rounded-md flex-1 p-3">{children}</div>
  );
}
