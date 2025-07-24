/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

/* eslint-disable react-refresh/only-export-components */
import { ControlsProvider } from "@/app/providers/controls";
import type {
  ColumnDef,
  PaginationState,
  RowSelectionState,
  Table,
  VisibilityState,
} from "@tanstack/react-table";
import { createContext, useContext, useMemo } from "react";

interface DataTableStateContextType {
  rowSelection: RowSelectionState;
  pagination: PaginationState;
  columnVisibility: VisibilityState;
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
      columnVisibility: props.columnVisibility ?? {},
    }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [
      props.table,
      props.pagination,
      props.isLoading,
      props.columns,
      props.rowSelection,
      props.columnVisibility,
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
