import {
  buildFilterItemsFromUrlState,
  filterItemsToUrlFilterState,
  serializeUrlFilterState,
  type UrlFilterState,
} from "@/lib/data-table";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import type { ColumnDef, FilterItem } from "@trenova/shared/types/data-table";
import type { RowData } from "@tanstack/react-table";
import { useCallback, useEffect, useRef, useState } from "react";

type FilterSearchParamsUpdate = UrlFilterState & { pageIndex: number };

type UseDataTableFilterSyncParams<TData extends RowData> = {
  columns: ColumnDef<TData>[];
  fieldFilters: UrlFilterState["fieldFilters"];
  filterGroups: UrlFilterState["filterGroups"];
  setSearchParams: (values: FilterSearchParamsUpdate) => unknown;
};

export function useDataTableFilterSync<TData extends RowData>({
  columns,
  fieldFilters,
  filterGroups,
  setSearchParams,
}: UseDataTableFilterSyncParams<TData>) {
  const urlState: UrlFilterState = { fieldFilters, filterGroups };
  const urlSerialized = serializeUrlFilterState(urlState);

  const [filterItems, setFilterItems] = useState<FilterItem[]>(() =>
    buildFilterItemsFromUrlState(urlState, columns),
  );

  const lastSyncedUrlRef = useRef(urlSerialized);
  const urlSerializedRef = useRef(urlSerialized);
  urlSerializedRef.current = urlSerialized;
  const filterItemsRef = useRef(filterItems);
  filterItemsRef.current = filterItems;
  const columnsRef = useRef(columns);
  columnsRef.current = columns;
  const setSearchParamsRef = useRef(setSearchParams);
  setSearchParamsRef.current = setSearchParams;

  const debouncedFilterItems = useDebounce(filterItems, 300);

  useEffect(() => {
    const next = filterItemsToUrlFilterState(debouncedFilterItems);
    const nextSerialized = serializeUrlFilterState(next);
    const currentDraftSerialized = serializeUrlFilterState(
      filterItemsToUrlFilterState(filterItemsRef.current),
    );
    if (nextSerialized !== currentDraftSerialized) return;
    if (nextSerialized === urlSerializedRef.current) {
      lastSyncedUrlRef.current = nextSerialized;
      return;
    }
    lastSyncedUrlRef.current = nextSerialized;
    void setSearchParamsRef.current({ ...next, pageIndex: 1 });
  }, [debouncedFilterItems]);

  useEffect(() => {
    if (urlSerialized === lastSyncedUrlRef.current) return;
    lastSyncedUrlRef.current = urlSerialized;
    const draftSerialized = serializeUrlFilterState(
      filterItemsToUrlFilterState(filterItemsRef.current),
    );
    if (draftSerialized === urlSerialized) return;
    setFilterItems(
      buildFilterItemsFromUrlState({ fieldFilters, filterGroups }, columnsRef.current),
    );
  }, [urlSerialized, fieldFilters, filterGroups]);

  const applyFilterState = useCallback((state: UrlFilterState) => {
    lastSyncedUrlRef.current = serializeUrlFilterState(state);
    setFilterItems(buildFilterItemsFromUrlState(state, columnsRef.current));
  }, []);

  return { filterItems, setFilterItems, applyFilterState };
}
