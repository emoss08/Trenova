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
  flexRender,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  Table as TableType,
  useReactTable,
} from "@tanstack/react-table";

import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import axios from "@/lib/AxiosConfig";
import { API_URL } from "@/lib/constants";
import { useTableStore } from "@/stores/TableStore";
import { ApiResponse } from "@/types/server";
import {
  DataTableFacetedFilterListProps,
  DataTableProps,
  FilterConfig,
} from "@/types/tables";
import { DownloadIcon } from "@radix-ui/react-icons";
import { Plus, X } from "lucide-react";
import React from "react";
import { useQuery } from "react-query";
import { Button } from "./button";
import { DataTableFacetedFilter } from "./data-table-faceted-filter";
import { DataTablePagination } from "./data-table-pagination";
import { DataTableViewOptions } from "./data-table-view-options";
import { Input } from "./input";
import { Skeleton } from "./skeleton";

function DataTableFacetedFilterList<TData>({
  table,
  filters,
}: DataTableFacetedFilterListProps<TData>) {
  return (
    <>
      {filters.map((filter) => {
        const column = table.getColumn(filter.columnName as string);
        return (
          column && (
            <DataTableFacetedFilter
              key={filter.columnName as string}
              column={column}
              title={filter.title}
              options={filter.options}
            />
          )
        );
      })}
    </>
  );
}

function DataTableTopBar<K>({
  table,
  name,
  selectedRowCount,
  filterColumn,
  tableFacetedFilters,
}: {
  table: TableType<K>;
  name: string;
  selectedRowCount: number;
  filterColumn: string;
  tableFacetedFilters?: FilterConfig<K>[];
}) {
  const buttonConfig: {
    label: string;
    variant:
      | "default"
      | "destructive"
      | "outline"
      | "secondary"
      | "ghost"
      | "link"
      | "blue";
  } =
    selectedRowCount > 0
      ? {
          label: `Inactivate ${selectedRowCount} records`,
          variant: "destructive",
        }
      : {
          label: `Add New ${name}`,
          variant: "default",
        };

  const { label: buttonLabel, variant: buttonVariant } = buttonConfig;
  const isFiltered = table.getState().columnFilters.length > 0;

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-1 items-center space-x-2">
        <Input
          placeholder="Filter..."
          value={
            (table.getColumn(filterColumn)?.getFilterValue() as string) ?? ""
          }
          onChange={(event) =>
            table.getColumn(filterColumn)?.setFilterValue(event.target.value)
          }
          className="w-[150px] lg:w-[250px]"
        />
        {tableFacetedFilters && (
          <DataTableFacetedFilterList
            table={table}
            filters={tableFacetedFilters}
          />
        )}
        {isFiltered && (
          <Button
            variant="ghost"
            onClick={() => table.resetColumnFilters()}
            className="h-8 px-2 lg:px-3"
          >
            Reset
            <X className="ml-2 h-4 w-4" />
          </Button>
        )}
      </div>
      <DataTableViewOptions table={table} />
      <Button variant="default" className="hidden h-8 lg:flex">
        <DownloadIcon className="mr-2 h-4 w-4" /> Export
      </Button>
      <Button
        variant={buttonVariant}
        onClick={() => useTableStore.set("sheetOpen", true)}
        className="ml-2 hidden h-8 lg:flex"
      >
        <Plus className="mr-2 h-4 w-4" /> {buttonLabel}
      </Button>
    </div>
  );
}

export function DataTable<K>({
  columns,
  link,
  name,
  filterColumn,
  tableFacetedFilters,
  TableSheet,
}: DataTableProps<K>) {
  const [{ pageIndex, pageSize }, setPagination] =
    useTableStore.use("pagination");
  const [rowSelection, setRowSelection] = useTableStore.use("rowSelection");

  const [columnVisibility, setColumnVisibility] =
    useTableStore.use("columnVisibility");
  const [columnFilters, setColumnFilters] = useTableStore.use("columnFilters");
  const [sorting, setSorting] = useTableStore.use("sorting");
  const [drawerOpen, setDrawerOpen] = useTableStore.use("sheetOpen");

  const dataQuery = useQuery<ApiResponse<K>, Error>(
    [link, pageIndex, pageSize],
    async () => {
      const fetchURL = new URL(`${API_URL}${link}/`);
      fetchURL.searchParams.set("limit", pageSize.toString());
      fetchURL.searchParams.set("offset", (pageIndex * pageSize).toString());

      const response = await axios.get(fetchURL.href);
      if (response.status !== 200) {
        throw new Error("Failed to fetch data from server");
      }
      return response.data;
    },

    { keepPreviousData: true, staleTime: Infinity },
  );

  const placeholderData: K[] = React.useMemo(
    () =>
      dataQuery.isLoading
        ? Array.from({ length: pageSize }, () => ({}) as K)
        : dataQuery.data?.results || [],
    [dataQuery.isLoading, dataQuery.data, pageSize],
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

  const pagination = React.useMemo(
    () => ({
      pageIndex,
      pageSize,
    }),
    [pageIndex, pageSize],
  );

  const table = useReactTable({
    data: placeholderData,
    columns: displayColumns,
    pageCount: dataQuery.data ? Math.ceil(dataQuery.data.count / pageSize) : -1,
    state: {
      pagination: pagination,
      sorting,
      columnVisibility,
      rowSelection,
      columnFilters,
    },
    debugTable: true,
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
  });

  const selectedRowCount = Object.keys(rowSelection).length;

  return (
    <>
      <div className="space-y-4">
        <DataTableTopBar
          table={table}
          name={name}
          filterColumn={filterColumn}
          selectedRowCount={selectedRowCount}
          tableFacetedFilters={tableFacetedFilters}
        />
        <div className="rounded-md">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => {
                    return (
                      <TableHead key={header.id}>
                        {header.isPlaceholder
                          ? null
                          : flexRender(
                              header.column.columnDef.header,
                              header.getContext(),
                            )}
                      </TableHead>
                    );
                  })}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows?.length ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow
                    key={row.id}
                    data-state={row.getIsSelected() && "selected"}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext(),
                        )}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell
                    colSpan={columns.length}
                    className="h-24 text-center"
                  >
                    No results.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
        <DataTablePagination table={table} pagination={pagination} />
      </div>
      {TableSheet && (
        <TableSheet open={drawerOpen} onOpenChange={setDrawerOpen} />
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
