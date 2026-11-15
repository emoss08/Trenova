import { useSearchState } from "@trenova/shared/hooks/use-search-state";
import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import type { ShipmentBoardScopeInput } from "@trenova/graphql/generated/graphql";
import { useMemo } from "react";
import { useShipmentBoardUrl } from "./url-state";

/**
 * The set of shipments the table is showing, stated the way the board's
 * aggregate queries take it, so group totals, facet counts and quick-filter
 * counts always describe the same rows the table pages through.
 */
export function useBoardScope(): ShipmentBoardScopeInput {
  const [{ query, fieldFilters, filterGroups }] = useSearchState(searchParamsParser);
  const [{ qf }] = useShipmentBoardUrl();
  const timezone = useUserTimezone();
  return useMemo(
    () => ({
      query: query || undefined,
      fieldFilters,
      filterGroups,
      quickFilters: qf,
      timezone,
    }),
    [query, fieldFilters, filterGroups, qf, timezone],
  );
}
