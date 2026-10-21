import {
  buildFilterItemsFromUrlState,
  filterItemsToUrlFilterState,
  type FilterableField,
  serializeUrlFilterState,
  type UrlFilterState,
} from "@/lib/data-table";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import type { FilterItem } from "@trenova/shared/types/data-table";
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";

type FilterSearchParamsUpdate = UrlFilterState & { pageIndex: number };

type UseDataTableFilterSyncParams = {
  fields: readonly FilterableField[];
  fieldFilters: UrlFilterState["fieldFilters"];
  filterGroups: UrlFilterState["filterGroups"];
  setSearchParams: (values: FilterSearchParamsUpdate) => unknown;
};

export function useDataTableFilterSync({
  fields,
  fieldFilters,
  filterGroups,
  setSearchParams,
}: UseDataTableFilterSyncParams) {
  const urlState: UrlFilterState = { fieldFilters, filterGroups };
  const urlSerialized = serializeUrlFilterState(urlState);

  const [filterItems, setFilterItems] = useState<FilterItem[]>(() =>
    buildFilterItemsFromUrlState(urlState, fields),
  );

  const lastSyncedUrlRef = useRef(urlSerialized);
  const urlSerializedRef = useRef(urlSerialized);
  const filterItemsRef = useRef(filterItems);
  const fieldsRef = useRef(fields);
  const setSearchParamsRef = useRef(setSearchParams);

  useLayoutEffect(() => {
    urlSerializedRef.current = urlSerialized;
    filterItemsRef.current = filterItems;
    fieldsRef.current = fields;
    setSearchParamsRef.current = setSearchParams;
  });

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
    setFilterItems(buildFilterItemsFromUrlState({ fieldFilters, filterGroups }, fieldsRef.current));
  }, [urlSerialized, fieldFilters, filterGroups]);

  const applyFilterState = useCallback((state: UrlFilterState) => {
    lastSyncedUrlRef.current = serializeUrlFilterState(state);
    setFilterItems(buildFilterItemsFromUrlState(state, fieldsRef.current));
  }, []);

  return { filterItems, setFilterItems, applyFilterState };
}
