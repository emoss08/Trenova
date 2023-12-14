/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import {
  ColumnDef,
  ColumnFilter,
  ColumnFiltersState,
  ColumnSort,
  getCoreRowModel,
  getExpandedRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  OnChangeFn,
  PaginationState,
  RowSelectionState,
  SortingState,
  useReactTable,
  VisibilityState,
} from "@tanstack/react-table";
import React, { SetStateAction } from "react";

import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Table } from "@/components/ui/table";
import { useUserPermissions } from "@/context/user-permissions";
import axios from "@/lib/axiosConfig";
import { API_URL } from "@/lib/constants";
import { useTableStore as store } from "@/stores/TableStore";
import { EquipmentStatus } from "@/types/equipment";
import { ApiResponse } from "@/types/server";
import { DataTableProps } from "@/types/tables";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import { DataTableBody } from "./data-table-body";
import { TableExportModal } from "./data-table-export-modal";
import { DataTableHeader, DataTableTopBar } from "./data-table-header";
import { DataTablePagination } from "./data-table-pagination";

// Define the structure of the state managed by the hook
interface DataTableState<K> {
  pagination: { pageIndex: number; pageSize: number };
  setPagination: OnChangeFn<PaginationState>;
  rowSelection: Record<string, boolean>;
  setRowSelection: OnChangeFn<RowSelectionState>;
  currentRecord?: K;
  setCurrentRecord: (currentRecord: K | null) => void;
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
function useDataTableState<K>(): DataTableState<K> {
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
  queryKey: string,
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

export function DataTable<K extends Record<string, any>>({
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
}: DataTableProps<K>) {
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
  } = useDataTableState<K>();

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
        ? Array.from({ length: pagination.pageSize }, () => ({}) as K)
        : dataQuery.data?.results || [],
    [dataQuery.isLoading, dataQuery.data, pagination.pageSize],
  );

  const displayColumns: ColumnDef<K>[] = React.useMemo(
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
    data: placeholderData as K[],
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
    return (
      <div className="text-center">
        <AlertTriangle className="mx-auto h-6 w-6 text-accent-foreground" />
        <p className="mt-2 font-semibold text-accent-foreground">
          Well, this is embarrassing...
        </p>
        <p className="mt-2 text-muted-foreground">
          We were unable to load the data for this table. Please try again
          later.
        </p>
      </div>
    );
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
          <div className="rounded-md border">
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

export function StatusBadge({ status }: { status: string }) {
  return (
    <Badge variant={status === "A" ? "default" : "destructive"}>
      {status === "A" ? "Active" : "Inactive"}
    </Badge>
  );
}

export function BoolStatusBadge({ status }: { status: boolean }) {
  return (
    <Badge variant={status ? "default" : "destructive"}>
      {status ? "Yes" : "No"}
    </Badge>
  );
}

/**
 * Status badge that can be used to display the status of equipment. (e.g. Trailer & Tractor statuses)
 * @param status The status of the equipment
 * @returns A badge with the status of the equipment
 */
export function EquipmentStatusBadge({ status }: { status: EquipmentStatus }) {
  const mapToStatus = {
    A: "Available",
    OOS: "Out of Service",
    AM: "At Maintenance",
    S: "Sold",
    L: "Lost",
  };

  return (
    <Badge variant={status === "A" ? "default" : "destructive"}>
      {mapToStatus[status]}
    </Badge>
  );
}
