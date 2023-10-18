import { createGlobalStore } from "./../lib/useGlobalStore";
import {
  ColumnFiltersState,
  PaginationState,
  RowSelectionState,
  SortingState,
  VisibilityState,
} from "@tanstack/react-table";

export type TableStoreProps = {
  pagination: PaginationState;
  columnVisibility: VisibilityState;
  rowSelection: RowSelectionState;
  currentRecord: Record<string, any> | null;
  columnFilters: ColumnFiltersState;
  sorting: SortingState;
  sheetOpen: boolean;
};

export const useTableStore = createGlobalStore<TableStoreProps>({
  pagination: {
    pageIndex: 0,
    pageSize: 10,
  },
  columnVisibility: {},
  currentRecord: null,
  rowSelection: {},
  columnFilters: [],
  sorting: [],
  sheetOpen: false,
});
