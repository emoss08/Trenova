import { usePageViewStore, type RegisteredTableView } from "@/stores/page-view-store";
import { useTableAtom, type TableAtomSource } from "@trenova/shared/hooks/use-table-atom";
import type { FieldFilter, FilterGroup, SortField } from "@trenova/shared/types/data-table";
import type { RowSelectionState } from "@tanstack/react-table";
import { useEffect } from "react";

/** The selected row ids as one string, so a tick redraws only when the set changes. */
function selectedIdsKey(selection: RowSelectionState): string {
  let key = "";
  for (const id in selection) {
    if (selection[id]) key = key === "" ? id : `${key},${id}`;
  }
  return key;
}

type PageViewRegistrationInput = {
  resource: string | undefined;
  query: string;
  fieldFilters: readonly FieldFilter[];
  filterGroups: readonly FilterGroup[];
  sort: readonly SortField[];
  selection: TableAtomSource<RowSelectionState>;
  columnVisibility: Readonly<Record<string, boolean>>;
  columnIds: readonly string[];
  rowCount: number | null;
};

/**
 * Tells the page view what this table is showing, as it changes, and takes
 * it back when the table leaves. One table per page is what the assistant
 * reads; a page with two registers the one mounted last. The selection is
 * followed from the table's own state, so the caller redraws only when the
 * selected set changes, not on every render of the table.
 */
export function usePageViewRegistration({
  resource,
  query,
  fieldFilters,
  filterGroups,
  sort,
  selection,
  columnVisibility,
  columnIds,
  rowCount,
}: PageViewRegistrationInput) {
  const setTable = usePageViewStore((state) => state.setTable);
  // Joined keys keep the effect keyed on content rather than on identity:
  // nuqs hands back fresh arrays on every render.
  const filtersKey = JSON.stringify(fieldFilters);
  const groupsKey = JSON.stringify(filterGroups);
  const sortKey = JSON.stringify(sort);
  const selectionKey = useTableAtom(selection, selectedIdsKey);
  const visibleColumns = columnIds.filter((id) => columnVisibility[id] ?? true);
  const columnsKey = visibleColumns.join(",");

  useEffect(() => {
    if (!resource) {
      return;
    }
    const selectedIds = selectionKey === "" ? [] : selectionKey.split(",");
    const table: RegisteredTableView = {
      resource,
      query,
      fieldFilters: JSON.parse(filtersKey) as RegisteredTableView["fieldFilters"],
      filterGroups: JSON.parse(groupsKey) as RegisteredTableView["filterGroups"],
      sort: JSON.parse(sortKey) as RegisteredTableView["sort"],
      selectedIds,
      selectionCount: selectedIds.length,
      visibleColumns: columnsKey === "" ? [] : columnsKey.split(","),
      rowCount,
    };
    setTable(table);
  }, [
    columnsKey,
    filtersKey,
    groupsKey,
    query,
    resource,
    rowCount,
    selectionKey,
    setTable,
    sortKey,
  ]);

  useEffect(() => () => setTable(null), [setTable]);
}
