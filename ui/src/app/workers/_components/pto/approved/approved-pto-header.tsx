import { Button } from "@/components/ui/button";
import { PTOFilterSchema } from "@/lib/schemas/worker-schema";
import { PTOType } from "@/types/worker";
import { Calendar, ChartColumn } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useCallback } from "react";
import { PTOFilterPopover } from "../pto-filter-popover";
import { usePTOFilters } from "../use-pto-filters";
import { ptoSearchParamsParser, viewTypeChoices } from "../use-pto-state";

export function ApprovedPTOHeader() {
  const [searchParams, setSearchParams] = useQueryStates(ptoSearchParamsParser);
  const { defaultValues } = usePTOFilters();

  const handleViewTypeChange = useCallback(
    (viewType: (typeof viewTypeChoices)[number]) => {
      setSearchParams({ viewType, ...defaultValues });
    },
    [defaultValues, setSearchParams],
  );

  const handleFilterSubmit = (data: PTOFilterSchema) => {
    setSearchParams({
      viewType: "chart",
      ptoOverviewFilters: {
        startDate: data.startDate,
        endDate: data.endDate,
        type: data.type as PTOType | undefined,
        workerId: data.workerId,
        fleetCodeId: data.fleetCodeId,
      },
    });
  };

  const resetFilters = () => {
    setSearchParams({
      viewType: "chart",
      ptoOverviewFilters: {
        startDate: defaultValues.startDate,
        endDate: defaultValues.endDate,
        type: undefined,
        workerId: undefined,
        fleetCodeId: undefined,
      },
    });
  };

  return (
    <HeaderOuter>
      <h3 className="text-lg font-medium font-table">Approved PTO Overview</h3>
      <HeaderInner>
        <ViewTypeButtons>
          <Button
            variant={searchParams.viewType === "chart" ? "default" : "ghost"}
            onClick={() => handleViewTypeChange("chart")}
          >
            <ChartColumn className="size-3.5" />
            <span>Chart</span>
          </Button>
          <Button
            variant={searchParams.viewType === "calendar" ? "default" : "ghost"}
            onClick={() => handleViewTypeChange("calendar")}
            disabled
          >
            <Calendar className="size-3.5" />
            <span>Calendar</span>
          </Button>
        </ViewTypeButtons>
        <PTOFilterPopover
          defaultValues={defaultValues}
          onSubmit={handleFilterSubmit}
          onReset={resetFilters}
        />
      </HeaderInner>
    </HeaderOuter>
  );
}

function HeaderOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center justify-between">{children}</div>;
}

function HeaderInner({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center gap-2">{children}</div>;
}

function ViewTypeButtons({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-row gap-0.5 items-center p-0.5 bg-sidebar border border-border rounded-md">
      {children}
    </div>
  );
}
