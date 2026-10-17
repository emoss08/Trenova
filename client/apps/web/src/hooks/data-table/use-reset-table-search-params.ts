import { useQueryStates } from "nuqs";
import { useCallback } from "react";
import { searchParamsParser } from "./use-data-table-state";

/**
 * Clears what a data table keeps in the address: its search, filters, sort, page and open panel.
 * Every table on a page reads the same parameters, so a page that swaps one table for another
 * (a workspace tab, a view switch) clears them on the way, or the new table would query with
 * filters on fields it does not have.
 */
export function useResetTableSearchParams(): () => void {
  const [, setTableParams] = useQueryStates(searchParamsParser);

  return useCallback(() => {
    void setTableParams({
      entityId: null,
      modalType: null,
      panelType: null,
      panelEntityId: null,
      pageIndex: null,
      pageSize: null,
      query: null,
      fieldFilters: null,
      filterGroups: null,
      sort: null,
    });
  }, [setTableParams]);
}
