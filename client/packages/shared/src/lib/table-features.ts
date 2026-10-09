import {
  cellSelectionFeature,
  columnOrderingFeature,
  columnPinningFeature,
  columnResizingFeature,
  columnSizingFeature,
  columnVisibilityFeature,
  createSortedRowModel,
  rowPaginationFeature,
  rowPinningFeature,
  rowSelectionFeature,
  rowSortingFeature,
  sortFns,
  tableFeatures,
  type ColumnOrderState,
  type ColumnPinningState,
  type ColumnSizingState,
  type PaginationState,
  type RowPinningState,
  type RowSelectionState,
  type TableState,
} from "@tanstack/react-table";
import { cellEditingFeature } from "./cell-editing-feature";

const columnFeatures = {
  cellEditingFeature,
  cellSelectionFeature,
  columnOrderingFeature,
  columnPinningFeature,
  columnSizingFeature,
  columnResizingFeature,
  columnVisibilityFeature,
  rowPaginationFeature,
  rowPinningFeature,
  rowSelectionFeature,
  rowSortingFeature,
} as const;

/**
 * The server-driven data table. The server sorts and pages, so no client-side
 * row model or sort-function registry is shipped with it.
 */
export const dataTableFeatures = tableFeatures(columnFeatures);

export type DataTableFeatures = typeof dataTableFeatures;

/** A table that holds every row in the browser and sorts them itself. */
export const clientSortedTableFeatures = tableFeatures({
  ...columnFeatures,
  sortedRowModel: createSortedRowModel(),
  sortFns,
});

export type ClientSortedTableFeatures = typeof clientSortedTableFeatures;

/**
 * The slice of table state the data table's shell draws from. Everything else
 * (selection, the cell being edited, a column mid-resize) is read by the part
 * that shows it, so a checkbox, an editor or a resize drag never redraws the
 * toolbar, the headers and the pager around it.
 */
export type DataTableViewState = {
  columnVisibility: TableState<DataTableFeatures>["columnVisibility"];
  columnOrder: ColumnOrderState;
  columnPinning: ColumnPinningState;
  pagination: PaginationState;
  /** The rows a person keeps at the top of the table, whatever page or filter is showing. */
  rowPinning: RowPinningState;
  /** The widths once a resize settles; absent while a column is being dragged. */
  columnSizing: ColumnSizingState | undefined;
};

export function hasSelectedRows(selection: RowSelectionState): boolean {
  for (const id in selection) {
    if (selection[id]) return true;
  }
  return false;
}

export function countSelectedRows(selection: RowSelectionState): number {
  let count = 0;
  for (const id in selection) {
    if (selection[id]) count += 1;
  }
  return count;
}

export function selectDataTableViewState(
  state: TableState<DataTableFeatures>,
): DataTableViewState {
  return {
    columnVisibility: state.columnVisibility,
    columnOrder: state.columnOrder,
    columnPinning: state.columnPinning,
    pagination: state.pagination,
    rowPinning: state.rowPinning,
    columnSizing: state.columnResizing.isResizingColumn === false ? state.columnSizing : undefined,
  };
}
