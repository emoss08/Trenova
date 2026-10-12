import type { RowData } from "@tanstack/react-table";
import type { RowAction } from "@trenova/shared/types/data-table";
import { createContext, useContext } from "react";

export type DataTableRowState = {
  isExpanded: boolean;
  isCursor: boolean;
  isSelected: boolean;
};

const NO_ROW_STATE: DataTableRowState = { isExpanded: false, isCursor: false, isSelected: false };

/**
 * The state of the row a cell sits in. A cell that draws differently when its
 * row is open or under the cursor reads it here rather than closing over the
 * table's expanded or cursor id, which would rebuild every column (and redraw
 * every cell) each time that id moved.
 */
export const DataTableRowStateContext = createContext<DataTableRowState>(NO_ROW_STATE);

export function useDataTableRowState(): DataTableRowState {
  return useContext(DataTableRowStateContext);
}

const NO_ROW_ACTIONS: RowAction<any>[] = [];

/**
 * The table's row actions, already guarded so a running action locks its row.
 * They travel by context rather than as a column or row prop: the set changes
 * whenever an action starts or settles, and as a prop that would redraw every
 * row, or rebuild the columns and remount every cell.
 */
export const DataTableRowActionsContext = createContext<RowAction<any>[]>(NO_ROW_ACTIONS);

export function useDataTableRowActions<TData extends RowData>(): RowAction<TData>[] {
  return useContext(DataTableRowActionsContext) as RowAction<TData>[];
}
