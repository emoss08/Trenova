import {
  ColumnDef,
  flexRender,
  getCoreRowModel,
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
import React from "react";
import { useQuery } from "react-query";
import { Skeleton } from "./skeleton";

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
}

export function DataTable<TData, TValue>({
  columns,
  link,
}: DataTableProps<TData, TValue> & { link: string }) {
  type ApiResponse = {
    count: number;
    next: string | null;
    previous: string | null;
    results: TData[];
  };

  const fetchTableData = async (): Promise<ApiResponse> => {
    const response = await axios.get(link);

    if (response.status !== 200) {
      throw new Error("Failed to fetch data from server");
    }

    return response.data;
  };

  const { data, isLoading, isError, error } = useQuery<ApiResponse>(
    link,
    fetchTableData,
  );

  const tableData = data?.results || [];

  const placeholderData: TData[] = React.useMemo(
    () =>
      isLoading ? Array.from({ length: 10 }, () => ({}) as TData) : tableData,
    [isLoading, tableData],
  );

  const tableColumns = React.useMemo(
    () =>
      isLoading
        ? columns.map((column) => ({
            ...column,
            cell: () => <Skeleton className="h-5 w-full" />,
          }))
        : columns,
    [isLoading, columns],
  );

  const table = useReactTable({
    data: placeholderData,
    columns: tableColumns,
    getCoreRowModel: getCoreRowModel(),
  });

  if (isError) {
    return <div>Error: {(error as Error).message}</div>;
  }

  return (
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
              <TableRow
                key={row.id}
                data-state={row.getIsSelected() && "selected"}
              >
                {row.getVisibleCells().map((cell) => (
                  <TableCell key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={columns.length} className="h-24 text-center">
                No results.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}

export function StatusBadge({ status }: { status: string }) {
  return (
    <Badge variant={status === "A" ? "active" : "inactive"}>
      {status === "A" ? "Active" : "Inactive"}
    </Badge>
  );
}
