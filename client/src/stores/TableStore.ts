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
  currentRecord: Record<string, unknown> | undefined;
  columnFilters: ColumnFiltersState;
  sorting: SortingState;
  sheetOpen: boolean;
  editSheetOpen: boolean;
};

export const useTableStore = createGlobalStore<TableStoreProps>({
  pagination: {
    pageIndex: 0,
    pageSize: 10,
  },
  columnVisibility: {},
  currentRecord: undefined,
  rowSelection: {},
  columnFilters: [],
  sorting: [],
  sheetOpen: false,
  editSheetOpen: false,
});
