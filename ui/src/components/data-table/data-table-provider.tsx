import { ControlsProvider } from "@/app/providers/controls";
import type {
  ColumnDef,
  PaginationState,
  RowSelectionState,
  Table,
} from "@tanstack/react-table";
import { createContext, useContext, useMemo } from "react";

interface DataTableStateContextType {
  rowSelection: RowSelectionState;
  pagination: PaginationState;
}

interface DataTableBaseContextType<TData = unknown, TValue = unknown> {
  table: Table<TData>;
  columns: ColumnDef<TData, TValue>[];
  isLoading: boolean;
}

interface DataTableContextType<TData = unknown, TValue = unknown>
  extends DataTableStateContextType,
    DataTableBaseContextType<TData, TValue> {}

const DataTableContext = createContext<DataTableContextType<any, any> | null>(
  null,
);

export function DataTableProvider<TData, TValue>({
  children,
  ...props
}: Partial<DataTableStateContextType> &
  DataTableBaseContextType<TData, TValue> & {
    children: React.ReactNode;
  }) {
  const value = useMemo(
    () => ({
      ...props,
      pagination: props.pagination ?? { pageIndex: 0, pageSize: 10 },
      rowSelection: props.rowSelection ?? {},
    }),
    [
      props.table,
      props.pagination,
      props.isLoading,
      props.columns,
      props.rowSelection,
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
