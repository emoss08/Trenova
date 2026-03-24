import type { PTOFilter, PTOType } from "@/types/worker";
import { useQueryStates } from "nuqs";
import { PTOFilterPopover } from "../pto-filter-popover";
import { HeaderContent } from "../pto-header-components";
import { usePTOFilters } from "../use-pto-filters";
import { ptoSearchParamsParser } from "../use-pto-state";

export function ApprovedPTOHeader() {
  const [, setSearchParams] = useQueryStates(ptoSearchParamsParser);
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
    <HeaderContent title="Approved PTO Overview">
      <PTOFilterPopover
        defaultValues={defaultValues}
        onSubmit={handleFilterSubmit}
        onReset={resetFilters}
      />
    </HeaderContent>
  );
}
