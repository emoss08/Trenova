/* eslint-disable react-refresh/only-export-components */
import { ControlsProvider } from "@/contexts/control-context";
import type { PanelMode, ColumnDef, Row, Table } from "@trenova/shared/types/data-table";
import type { PaginationState, RowData } from "@tanstack/react-table";
import { createContext, useContext, useMemo } from "react";

interface DataTableStateContextType {
  pagination: PaginationState;
}

interface DataTablePermissionsContextType {
  canCreate: boolean;
  canUpdate: boolean;
  canExport: boolean;
}

interface DataTablePanelContextType<TData extends RowData = RowData> {
  isPanelOpen: boolean;
  panelMode: PanelMode;
  panelRow: TData | null;
  getSelectedRows: () => TData[];
  openPanelCreate: () => void;
  openPanelEdit: (row: Row<TData>) => void;
  closePanel: () => void;
  hasPanel: boolean;
  canOpenPanel: boolean;
}

interface DataTableLayoutContextType {
  /** Widens or narrows columns to their widest content on the page; every resizable column when none are named. */
  fitColumns: (columnIds?: readonly string[]) => void;
  /** Copies the top selected cell of each column into the selected cells below it. */
  fillDown: () => void;
  /** Pastes the clipboard onto the selected cells, from the selection's top-left. */
  pasteFromClipboard: () => void;
}

interface DataTableBaseContextType<TData extends RowData = RowData, TValue = unknown> {
  table: Table<TData>;
  columns: ColumnDef<TData, TValue>[];
  isLoading: boolean;
}

interface DataTableContextType<TData extends RowData = RowData, TValue = unknown>
  extends
    DataTableStateContextType,
    DataTableBaseContextType<TData, TValue>,
    DataTablePanelContextType<TData>,
    DataTableLayoutContextType,
    DataTablePermissionsContextType {}

const DataTableContext = createContext<DataTableContextType<any, any> | null>(null);

const noopFn = () => {};
const emptyRows = () => [];

export function DataTableProvider<TData extends RowData, TValue>({
  children,
  ...props
}: Partial<DataTableStateContextType> &
  Partial<DataTablePanelContextType<TData>> &
  Partial<DataTableLayoutContextType> &
  Partial<DataTablePermissionsContextType> &
  DataTableBaseContextType<TData, TValue> & {
    children: React.ReactNode;
  }) {
  const value = useMemo(
    () => ({
      table: props.table,
      columns: props.columns,
      isLoading: props.isLoading,
      pagination: props.pagination ?? { pageIndex: 0, pageSize: 10 },
      getSelectedRows: props.getSelectedRows ?? emptyRows,
      isPanelOpen: props.isPanelOpen ?? false,
      panelMode: props.panelMode ?? "create",
      panelRow: props.panelRow ?? null,
      openPanelCreate: props.openPanelCreate ?? noopFn,
      openPanelEdit: props.openPanelEdit ?? noopFn,
      closePanel: props.closePanel ?? noopFn,
      hasPanel: props.hasPanel ?? false,
      canOpenPanel: props.canOpenPanel ?? false,
      fitColumns: props.fitColumns ?? noopFn,
      fillDown: props.fillDown ?? noopFn,
      pasteFromClipboard: props.pasteFromClipboard ?? noopFn,
      canCreate: props.canCreate ?? true,
      canUpdate: props.canUpdate ?? true,
      canExport: props.canExport ?? true,
    }),
    [
      props.table,
      props.columns,
      props.isLoading,
      props.pagination,
      props.getSelectedRows,
      props.isPanelOpen,
      props.panelMode,
      props.panelRow,
      props.openPanelCreate,
      props.openPanelEdit,
      props.closePanel,
      props.hasPanel,
      props.canOpenPanel,
      props.fitColumns,
      props.fillDown,
      props.pasteFromClipboard,
      props.canCreate,
      props.canUpdate,
      props.canExport,
    ],
  );

  return (
    <DataTableContext.Provider value={value as DataTableContextType<any, any>}>
      <ControlsProvider>{children}</ControlsProvider>
    </DataTableContext.Provider>
  );
}

export function useDataTable<TData extends RowData, TValue>() {
  const context = useContext(DataTableContext);

  if (!context) {
    throw new Error("useDataTable must be used within a DataTableProvider");
  }

  return context as DataTableContextType<TData, TValue>;
}

export function useOptionalDataTable<TData extends RowData, TValue>() {
  return useContext(DataTableContext) as DataTableContextType<TData, TValue> | null;
}
