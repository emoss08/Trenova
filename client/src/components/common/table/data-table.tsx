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
  getExpandedRowModel,
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
import { cn } from "@/lib/utils";
import { useTableStore as store } from "@/stores/TableStore";
import { ApiResponse } from "@/types/server";
import {
  DataTableFacetedFilterListProps,
  DataTableProps,
  FilterConfig,
} from "@/types/tables";
import { AlertTriangle, Plus, X } from "lucide-react";
import React, { Fragment } from "react";
import { useQuery } from "@tanstack/react-query";
import { DataTableFacetedFilter } from "./data-table-faceted-filter";
import { Button } from "@/components/ui/button";
import { DataTableViewOptions } from "./data-table-view-options";
import { Input } from "@/components/common/fields/input";
import { Skeleton } from "@/components/ui/skeleton";
import { DataTablePagination } from "./data-table-pagination";
import {
  DataTableImportExportOption,
  TableExportModal,
} from "./data-table-export-modal";
import { API_URL } from "@/lib/constants";
import axios from "@/lib/axiosConfig";

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
    <div className="flex flex-col sm:flex-row justify-between">
      <div className="flex-1 flex flex-col sm:flex-row space-y-2 sm:space-y-0 sm:space-x-2 mr-2">
        <Input
          placeholder="Filter..."
          value={
            (table.getColumn(filterColumn)?.getFilterValue() as string) ?? ""
          }
          onChange={(event) =>
            table.getColumn(filterColumn)?.setFilterValue(event.target.value)
          }
          className="w-full lg:w-[250px] h-8"
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
      <div className="flex flex-col sm:flex-row space-y-2 sm:space-y-0 sm:space-x-2 mt-2 sm:mt-0">
        <DataTableViewOptions table={table} />
        <DataTableImportExportOption />
        <Button
          variant={buttonVariant}
          onClick={() => store.set("sheetOpen", true)}
          className="h-8"
        >
          <Plus className="mr-2 h-4 w-4" /> {buttonLabel}
        </Button>
      </div>
    </div>
  );
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
}: DataTableProps<K>) {
  const [{ pageIndex, pageSize }, setPagination] = store.use("pagination");
  const [rowSelection, setRowSelection] = store.use("rowSelection");
  const [currentRecord, setCurrentRecord] = store.use("currentRecord");
  const [columnVisibility, setColumnVisibility] = store.use("columnVisibility");
  const [columnFilters, setColumnFilters] = store.use("columnFilters");
  const [sorting, setSorting] = store.use("sorting");
  const [drawerOpen, setDrawerOpen] = store.use("sheetOpen");
  const [editDrawerOpen, setEditDrawerOpen] = store.use("editSheetOpen");

  const dataQuery = useQuery<ApiResponse<K>, Error>({
    queryKey: [queryKey, link, pageIndex, pageSize, extraSearchParams],
    queryFn: async () => {
      const fetchURL = new URL(`${API_URL}${link}`);
      fetchURL.searchParams.set("limit", pageSize.toString());
      fetchURL.searchParams.set("offset", (pageIndex * pageSize).toString());
      if (extraSearchParams) {
        Object.entries(extraSearchParams).forEach(([key, value]) =>
          fetchURL.searchParams.set(key, value),
        );
      }

      const response = await axios.get(fetchURL.href);
      if (response.status !== 200) {
        throw new Error("Failed to fetch data from server");
      }
      return response.data;
    },
  });

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
    getRowCanExpand,
    pageCount: dataQuery.data ? Math.ceil(dataQuery.data.count / pageSize) : -1,
    state: {
      pagination: pagination,
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
          />
          <div className="rounded-md border">
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
                    <Fragment key={row.id}>
                      <TableRow data-state={row.getIsSelected() && "selected"}>
                        {row.getVisibleCells().map((cell) => (
                          <TableCell
                            key={cell.id}
                            className={cn("cursor-pointer")}
                            onDoubleClick={() => {
                              setCurrentRecord(row.original);
                              setEditDrawerOpen(true);
                            }}
                          >
                            {flexRender(
                              cell.column.columnDef.cell,
                              cell.getContext(),
                            )}
                          </TableCell>
                        ))}
                      </TableRow>
                      {/* Expanded row */}
                      {row.getIsExpanded() && (
                        <tr>
                          <td colSpan={row.getVisibleCells().length}>
                            {renderSubComponent({ row })}
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  ))
                ) : (
                  <TableRow>
                    <TableCell
                      colSpan={columns.length}
                      className="h-24 text-center"
                    >
                      No data available to display.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
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
