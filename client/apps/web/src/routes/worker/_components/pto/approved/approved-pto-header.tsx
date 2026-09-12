import { useT } from "@trenova/shared/i18n/use-t";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import type { PTOFilter, PTOType } from "@trenova/shared/types/worker";
import { BarChart3Icon, CalendarDaysIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { PTOFilterPopover } from "../pto-filter-popover";
import { HeaderContent } from "../pto-header-components";
import { usePTOFilters } from "../use-pto-filters";
import {
  ptoOverviewFiltersSearchParamsParser,
  ptoViewTypeSearchParamsParser,
  type PTOViewType,
} from "../use-pto-state";

const VIEW_ITEMS = [
  { value: "chart" as const, label: "Chart", icon: BarChart3Icon },
  { value: "calendar" as const, label: "Calendar", icon: CalendarDaysIcon },
];

export function ApprovedPTOHeader() {
  const t = useT();

  const [, setSearchParams] = useQueryStates(ptoOverviewFiltersSearchParamsParser);
  const [{ viewType }, setViewType] = useQueryStates(ptoViewTypeSearchParamsParser);
  const { defaultValues } = usePTOFilters();

  const handleFilterSubmit = (data: PTOFilter) => {
    void setSearchParams({
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
    void setSearchParams({
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
    <HeaderContent title={t("Approved PTO Overview")}>
      <SegmentedControl<PTOViewType>
        items={VIEW_ITEMS}
        value={viewType}
        className="h-full"
        onValueChange={(value) => void setViewType({ viewType: value })}
        aria-label={t("PTO view")}
      />
      <PTOFilterPopover
        defaultValues={defaultValues}
        onSubmit={handleFilterSubmit}
        onReset={resetFilters}
      />
    </HeaderContent>
  );
}
