/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { Skeleton } from "@/components/ui/skeleton";
import { Table } from "@/components/ui/table";
import { useUserPermissions } from "@/context/user-permissions";
import axios from "@/lib/axiosConfig";
import { API_URL } from "@/lib/constants";
import { useTableStore as store } from "@/stores/TableStore";
import type { QueryKeys, ValuesOf } from "@/types";
import { ApiResponse } from "@/types/server";
import { DataTableProps } from "@/types/tables";
import { useQuery } from "@tanstack/react-query";
import {
  ColumnDef,
  ColumnFilter,
  ColumnFiltersState,
  ColumnSort,
  OnChangeFn,
  PaginationState,
  RowSelectionState,
  SortingState,
  VisibilityState,
  getCoreRowModel,
  getExpandedRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import React, { SetStateAction } from "react";
import { DataTableBody } from "./data-table-body";
import { ErrorLoadingData } from "./data-table-components";
import { TableExportModal } from "./data-table-export-modal";
import { DataTableHeader, DataTableTopBar } from "./data-table-header";
import { DataTablePagination } from "./data-table-pagination";

// Define the structure of the state managed by the hook
interface DataTableState<TData extends Record<string, any>> {
  pagination: { pageIndex: number; pageSize: number };
  setPagination: OnChangeFn<PaginationState>;
  rowSelection: Record<string, boolean>;
  setRowSelection: OnChangeFn<RowSelectionState>;
  currentRecord?: TData;
  setCurrentRecord: (currentRecord: TData | null) => void;
  columnVisibility: Record<string, boolean>;
  setColumnVisibility: OnChangeFn<VisibilityState>;
  columnFilters: ColumnFilter[];
  setColumnFilters: (value: SetStateAction<ColumnFiltersState>) => void;
  sorting: ColumnSort[];
  setSorting: (value: SetStateAction<SortingState>) => void;
  drawerOpen: boolean;
  setDrawerOpen: (drawerOpen: boolean) => void;
  editDrawerOpen: boolean;
  setEditDrawerOpen: (editDrawerOpen: boolean) => void;
}

// Custom hook for managing DataTable state
function useDataTableState<
  TData extends Record<string, any>,
>(): DataTableState<TData> {
  const [{ pageIndex, pageSize }, setPagination] = store.use("pagination");
  const [rowSelection, setRowSelection] = store.use("rowSelection");
  const [currentRecord, setCurrentRecord] = store.use("currentRecord");
  const [columnVisibility, setColumnVisibility] = store.use("columnVisibility");
  const [columnFilters, setColumnFilters] = store.use("columnFilters");
  const [sorting, setSorting] = store.use("sorting");
  const [drawerOpen, setDrawerOpen] = store.use("sheetOpen");
  const [editDrawerOpen, setEditDrawerOpen] = store.use("editSheetOpen");

  return {
    pagination: { pageIndex, pageSize },
    setPagination,
    rowSelection,
    setRowSelection,
    currentRecord,
    setCurrentRecord,
    columnVisibility,
    setColumnVisibility,
    columnFilters,
    setColumnFilters,
    sorting,
    setSorting,
    drawerOpen,
    setDrawerOpen,
    editDrawerOpen,
    setEditDrawerOpen,
  };
}

// Custom hook for data fetching
function useDataTableQuery<K>(
  queryKey: ValuesOf<QueryKeys>,
  link: string,
  pageIndex: number,
  pageSize: number,
  extraSearchParams?: Record<string, any>,
) {
  return useQuery<ApiResponse<K>, Error>({
    queryKey: [queryKey, link, pageIndex, pageSize, extraSearchParams],
    queryFn: () => fetchData<K>(link, pageIndex, pageSize, extraSearchParams),
  });
}

// Separate function for the fetch logic
async function fetchData<K>(
  link: string,
  pageIndex: number,
  pageSize: number,
  extraSearchParams?: Record<string, any>,
): Promise<ApiResponse<K>> {
  const fetchURL = new URL(`${API_URL}${link}`);
  fetchURL.searchParams.set("limit", pageSize.toString());
  fetchURL.searchParams.set("offset", (pageIndex * pageSize).toString());
  if (extraSearchParams) {
    Object.entries(extraSearchParams).forEach(([key, value]) =>
      fetchURL.searchParams.set(key, value),
    );
  }

  const response = await axios.get<ApiResponse<K>>(fetchURL.href);
  if (response.status !== 200) {
    throw new Error("Failed to fetch data from server");
  }
  return response.data;
}

export function DataTable<TData extends Record<string, any>>({
  columns,
  link,
  extraSearchParams,
  queryKey,
  name,
  filterColumn,
  tableFacetedFilters,
  TableSheet,
  TableEditSheet,
  exportModelName,
  renderSubComponent,
  getRowCanExpand,
  addPermissionName,
}: DataTableProps<TData>) {
  const {
    pagination,
    setPagination,
    rowSelection,
    setRowSelection,
    currentRecord,
    setCurrentRecord,
    columnVisibility,
    setColumnVisibility,
    columnFilters,
    setColumnFilters,
    sorting,
    setSorting,
    drawerOpen,
    setDrawerOpen,
    editDrawerOpen,
    setEditDrawerOpen,
  } = useDataTableState<TData>();

  const { userHasPermission } = useUserPermissions();

  const dataQuery = useDataTableQuery(
    queryKey,
    link,
    pagination.pageIndex,
    pagination.pageSize,
    extraSearchParams,
  );

  const placeholderData: unknown[] = React.useMemo(
    () =>
      dataQuery.isLoading
        ? Array.from({ length: pagination.pageSize }, () => ({}) as TData)
        : dataQuery.data?.results || [],
    [dataQuery.isLoading, dataQuery.data, pagination.pageSize],
  );

  const displayColumns: ColumnDef<TData>[] = React.useMemo(
    () =>
      dataQuery.isLoading
        ? columns.map((column) => ({
            ...column,
            cell: () => <Skeleton className="h-5 w-full" />,
          }))
        : columns,
    [dataQuery.isLoading, columns],
  );

  const paginationState = React.useMemo(
    () => ({
      pageIndex: pagination.pageIndex,
      pageSize: pagination.pageSize,
    }),
    [pagination.pageIndex, pagination.pageSize],
  );

  const table = useReactTable({
    data: placeholderData as TData[],
    columns: displayColumns,
    getRowCanExpand: getRowCanExpand,
    pageCount: dataQuery.data
      ? Math.ceil(dataQuery.data.count / pagination.pageSize)
      : -1,
    state: {
      pagination: paginationState,
      sorting,
      columnVisibility,
      rowSelection,
      columnFilters,
    },
    manualPagination: true,
    enableRowSelection: true,
    onRowSelectionChange: setRowSelection,
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    getExpandedRowModel: getExpandedRowModel(),
  });

  if (dataQuery.isError) {
    return <ErrorLoadingData />;
  }

  const selectedRowCount = Object.keys(rowSelection).length;

  return (
    <>
      <div className="my-2">
        <div className="space-y-4">
          <DataTableTopBar
            table={table}
            name={name}
            filterColumn={filterColumn}
            selectedRowCount={selectedRowCount}
            tableFacetedFilters={tableFacetedFilters}
            addPermissionName={addPermissionName}
            userHasPermission={userHasPermission}
            store={store}
          />
          <div className="border-border rounded-md border">
            <Table>
              <DataTableHeader table={table} />
              <DataTableBody
                columns={columns}
                setCurrentRecord={setCurrentRecord}
                setEditDrawerOpen={setEditDrawerOpen}
                table={table}
                renderSubComponent={renderSubComponent}
              />
            </Table>
          </div>
          <DataTablePagination table={table} pagination={pagination} />
        </div>
      </div>
      <TableExportModal store={store} name={name} modelName={exportModelName} />
      {TableSheet && (
        <TableSheet open={drawerOpen} onOpenChange={setDrawerOpen} />
      )}
      {TableEditSheet && (
        <TableEditSheet
          open={editDrawerOpen}
          onOpenChange={setEditDrawerOpen}
          currentRecord={currentRecord}
        />
      )}
    </>
  );
}
