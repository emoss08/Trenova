import { fetchData } from "@/hooks/data-table/use-data-table-query";
import type { Shipment } from "@/types/shipment";
import { useQueries } from "@tanstack/react-query";
import { useMemo } from "react";
import { getMandatoryFieldFilters, SAVED_VIEWS, type SavedViewId } from "./saved-views";

const SHIPMENTS_LINK = "/shipments/";
const COUNT_PAGE_SIZE = 1; // backend min limit; we only read `count` from the response

export function useSavedViewCounts(): Record<SavedViewId, number | undefined> {
  const queries = useMemo(
    () =>
      SAVED_VIEWS.map((view) => {
        const fieldFilters = getMandatoryFieldFilters(view.id, []);
        return {
          queryKey: ["shipment-list", "view-count", view.id, fieldFilters],
          queryFn: () =>
            fetchData<Shipment & Record<string, unknown>>(
              SHIPMENTS_LINK,
              0,
              COUNT_PAGE_SIZE,
              { fieldFilters },
            ),
          staleTime: 30_000,
        };
      }),
    [],
  );

  // useQueries returns a new array reference each render, so use the
  // `combine` option to derive a stable result map.
  return useQueries({
    queries,
    combine: (results) => {
      const map = {} as Record<SavedViewId, number | undefined>;
      SAVED_VIEWS.forEach((v, i) => {
        map[v.id] = results[i]?.data?.count;
      });
      return map;
    },
  });
}
