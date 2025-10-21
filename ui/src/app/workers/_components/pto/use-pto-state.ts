import { dateToUnixTimestamp } from "@/lib/date";
import { ptoFilterSchema } from "@/lib/schemas/worker-schema";
import { parseAsJson, parseAsStringLiteral } from "nuqs";

export const viewTypeChoices = ["chart", "calendar"] as const;

export const ptoSearchParamsParser = {
  viewType: parseAsStringLiteral(viewTypeChoices)
    .withOptions({
      shallow: true,
    })
    .withDefault("chart"),
  ptoOverviewFilters: parseAsJson(ptoFilterSchema)
    .withOptions({
      shallow: true,
    })
    .withDefault({
      startDate: dateToUnixTimestamp(
        new Date(new Date().getFullYear(), new Date().getMonth(), 1),
      ),
      endDate: dateToUnixTimestamp(
        new Date(new Date().getFullYear(), new Date().getMonth() + 1, 0),
      ),
      type: undefined,
      workerId: undefined,
      fleetCodeId: undefined,
    }),
  requestPTOFilters: parseAsJson(ptoFilterSchema)
    .withOptions({
      shallow: true,
    })
    .withDefault({
      startDate: dateToUnixTimestamp(
        new Date(new Date().getFullYear(), new Date().getMonth(), 1),
      ),
      endDate: dateToUnixTimestamp(
        new Date(new Date().getFullYear(), new Date().getMonth() + 1, 0),
      ),
      type: undefined,
      workerId: undefined,
      fleetCodeId: undefined,
    }),
};
