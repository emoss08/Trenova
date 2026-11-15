import { getEndOfMonth, getStartOfMonth } from "@trenova/shared/lib/date";
import { ptoFilterSchema } from "@trenova/shared/types/worker";
import { searchJson, searchLiteral } from "@trenova/shared/lib/router/search-codec";

export const viewTypeChoices = ["chart", "calendar"] as const;
export type PTOViewType = (typeof viewTypeChoices)[number];

const ptoFilter = searchJson((value) => ptoFilterSchema.safeParse(value).data ?? null).withDefault({
  startDate: getStartOfMonth(),
  endDate: getEndOfMonth(),
  type: undefined,
  workerId: undefined,
  fleetCodeId: undefined,
});

export const ptoSearchParamsParser = {
  viewType: searchLiteral(viewTypeChoices).withDefault("chart"),
  ptoOverviewFilters: ptoFilter,
  requestPTOFilters: ptoFilter,
};

export const ptoViewTypeSearchParamsParser = {
  viewType: ptoSearchParamsParser.viewType,
};

export const ptoOverviewFiltersSearchParamsParser = {
  ptoOverviewFilters: ptoSearchParamsParser.ptoOverviewFilters,
};

export const requestPTOFiltersSearchParamsParser = {
  requestPTOFilters: ptoSearchParamsParser.requestPTOFilters,
};
