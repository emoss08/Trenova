import { usePageViewStore, type RegisteredTableView } from "@/stores/page-view-store";
import type { FieldFilter, FilterGroup, SortField } from "@trenova/shared/types/data-table";
import { useEffect } from "react";

type PageViewRegistrationInput = {
  resource: string | undefined;
  query: string;
  fieldFilters: readonly FieldFilter[];
  filterGroups: readonly FilterGroup[];
  sort: readonly SortField[];
  rowSelection: Readonly<Record<string, boolean>>;
  selectionCount: number;
  columnVisibility: Readonly<Record<string, boolean>>;
  columnIds: readonly string[];
  rowCount: number | null;
};

/**
 * Tells the page view what this table is showing, as it changes, and takes
 * it back when the table leaves. One table per page is what the assistant
 * reads; a page with two registers the one mounted last.
 */
export function usePageViewRegistration({
  resource,
  query,
  fieldFilters,
  filterGroups,
  sort,
  rowSelection,
  selectionCount,
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
  const selectedIds = Object.entries(rowSelection)
    .filter(([, selected]) => selected)
    .map(([id]) => id);
  const selectionKey = selectedIds.join(",");
  const visibleColumns = columnIds.filter((id) => columnVisibility[id] ?? true);
  const columnsKey = visibleColumns.join(",");

  useEffect(() => {
    if (!resource) {
      return;
    }
    const table: RegisteredTableView = {
      resource,
      query,
      fieldFilters: JSON.parse(filtersKey) as RegisteredTableView["fieldFilters"],
      filterGroups: JSON.parse(groupsKey) as RegisteredTableView["filterGroups"],
      sort: JSON.parse(sortKey) as RegisteredTableView["sort"],
      selectedIds: selectionKey === "" ? [] : selectionKey.split(","),
      selectionCount,
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
    selectionCount,
    selectionKey,
    setTable,
    sortKey,
  ]);

  useEffect(() => () => setTable(null), [setTable]);
}
