/* eslint-disable react-refresh/only-export-components */
import { ControlsProvider } from "@/contexts/control-context";
import type { PanelMode } from "@/types/data-table";
import type {
  ColumnDef,
  PaginationState,
  Row,
  RowSelectionState,
  Table,
} from "@tanstack/react-table";
import { createContext, useContext, useMemo } from "react";

interface DataTableStateContextType {
  pagination: PaginationState;
}

interface DataTablePermissionsContextType {
  canCreate: boolean;
  canUpdate: boolean;
  canExport: boolean;
}

interface DataTablePanelContextType<TData = unknown> {
  isPanelOpen: boolean;
  panelMode: PanelMode;
  panelRow: TData | null;
  rowSelection: RowSelectionState;
  openPanelCreate: () => void;
  openPanelEdit: (row: Row<TData>) => void;
  closePanel: () => void;
  hasPanel: boolean;
}

interface DataTableBaseContextType<TData = unknown, TValue = unknown> {
  table: Table<TData>;
  columns: ColumnDef<TData, TValue>[];
  isLoading: boolean;
}

interface DataTableContextType<TData = unknown, TValue = unknown>
  extends DataTableStateContextType,
    DataTableBaseContextType<TData, TValue>,
    DataTablePanelContextType<TData>,
    DataTablePermissionsContextType {}

const DataTableContext = createContext<DataTableContextType<any, any> | null>(
  null,
);

const noopFn = () => {};

export function DataTableProvider<TData, TValue>({
  children,
  ...props
}: Partial<DataTableStateContextType> &
  Partial<DataTablePanelContextType<TData>> &
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
      rowSelection: props.rowSelection ?? {},
      isPanelOpen: props.isPanelOpen ?? false,
      panelMode: props.panelMode ?? "create",
      panelRow: props.panelRow ?? null,
      openPanelCreate: props.openPanelCreate ?? noopFn,
      openPanelEdit: props.openPanelEdit ?? noopFn,
      closePanel: props.closePanel ?? noopFn,
      hasPanel: props.hasPanel ?? false,
      canCreate: props.canCreate ?? true,
      canUpdate: props.canUpdate ?? true,
      canExport: props.canExport ?? true,
    }),
    [
      props.table,
      props.columns,
      props.isLoading,
      props.pagination,
      props.rowSelection,
      props.isPanelOpen,
      props.panelMode,
      props.panelRow,
      props.openPanelCreate,
      props.openPanelEdit,
      props.closePanel,
      props.hasPanel,
      props.canCreate,
      props.canUpdate,
      props.canExport,
    ],
  );

  return (
    <DataTableContext.Provider value={value}>
      <ControlsProvider>{children}</ControlsProvider>
    </DataTableContext.Provider>
  );
}

export function useDataTable<TData, TValue>() {
  const context = useContext(DataTableContext);

  if (!context) {
    throw new Error("useDataTable must be used within a DataTableProvider");
  }

  return context as DataTableContextType<TData, TValue>;
}
