/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { QueryLazyComponent } from "@/components/error-boundary";
import { Button } from "@/components/ui/button";
import { queries } from "@/lib/queries";
import { PTOFilterSchema } from "@/lib/schemas/worker-schema";
import { PTOType } from "@/types/worker";
import { Calendar, ChartColumn } from "lucide-react";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { lazy, useCallback, useState } from "react";
import { PTOFilterPopover } from "./pto-filter-popover";
import { usePTOFilters } from "./use-pto-filters";

const PTOChart = lazy(() => import("./approved-pto-chart"));
const PTOCalendar = lazy(() => import("./approved-pto-calendar"));

const viewTypeChoices = ["chart", "calendar"] as const;

export function ApprovedPTOOverview() {
  const [viewType, setViewType] = useQueryState(
    "viewType",
    parseAsStringLiteral(viewTypeChoices)
      .withOptions({
        shallow: true,
      })
      .withDefault("chart"),
  );

  // Use the hook to get default values only
  const { defaultValues } = usePTOFilters();

  // Local filter state for this component only
  const [filters, setFilters] = useState({
    startDate: defaultValues.startDate,
    endDate: defaultValues.endDate,
    type: undefined as PTOType | undefined,
    workerId: undefined as string | undefined,
  });

  const handleFilterSubmit = useCallback((data: PTOFilterSchema) => {
    setFilters({
      startDate: data.startDate,
      endDate: data.endDate,
      type: data.type as PTOType | undefined,
      workerId: data.workerId,
    });
  }, []);

  const resetFilters = useCallback(() => {
    setFilters({
      startDate: defaultValues.startDate,
      endDate: defaultValues.endDate,
      type: undefined,
      workerId: undefined,
    });
  }, [defaultValues]);

  return (
    <div className="flex flex-col gap-1 flex-3">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium font-table">
          Approved PTO Overview
        </h3>
        <div className="flex items-center gap-2">
          <div className="flex flex-row gap-0.5 items-center p-0.5 bg-sidebar border border-border rounded-md">
            <Button
              variant={viewType === "chart" ? "default" : "ghost"}
              onClick={() => setViewType("chart")}
            >
              <ChartColumn className="size-3.5" />
              <span>Chart</span>
            </Button>
            <Button
              variant={viewType === "calendar" ? "default" : "ghost"}
              onClick={() => setViewType("calendar")}
            >
              <Calendar className="size-3.5" />
              <span>Calendar</span>
            </Button>
          </div>
          <PTOFilterPopover
            defaultValues={defaultValues}
            onSubmit={handleFilterSubmit}
            onReset={resetFilters}
          />
        </div>
      </div>
      <div className="border border-border rounded-md flex-1 p-3">
        {viewType === "chart" ? (
          <QueryLazyComponent queryKey={queries.worker.getPTOChartData._def}>
            <PTOChart
              startDate={filters.startDate}
              endDate={filters.endDate}
              type={filters.type}
              workerId={filters.workerId}
            />
          </QueryLazyComponent>
        ) : (
          <QueryLazyComponent queryKey={queries.worker.getPTOCalendarData._def}>
            <PTOCalendar
              startDate={filters.startDate}
              endDate={filters.endDate}
              type={filters.type}
            />
          </QueryLazyComponent>
        )}
      </div>
    </div>
  );
}
