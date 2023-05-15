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

import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import {
  flexRender,
  getCoreRowModel,
  PaginationState,
  useReactTable
} from "@tanstack/react-table";
import { DataTablePagination } from "@/components/ui/pagination";
import React from "react";
import axios from "@/lib/axiosConfig";
import { columns } from "@/components/user-management/users/columns";
import { useQuery } from "react-query";

export function UserDataTable({}) {
  const [page, setPage] = React.useState<number>(0);
  const [count, setCount] = React.useState<number>(10);
  const [currentUrl, setCurrentUrl] = React.useState<string>("");

  const fetchUsers = (url: string, pageSize: number) => {
    if (!url) {
      url = `http://localhost:8000/api/users/?limit=${pageSize}`;
    }
    return axios.get(url).then((res) => res.data);
  };

  const dataQuery = useQuery(
    ["users", currentUrl],
    () => fetchUsers(currentUrl, count),
    { keepPreviousData: true }
  );

  const [{ pageIndex, pageSize }, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10
  });

  const handlePageIndexChange = (pageIndex: number) => {
    setPage(pageIndex);
  };

  React.useEffect(() => {
    setCurrentUrl(`http://localhost:8000/api/users/?limit=${count}&offset=${page * count}`);
  }, [page, count]);

  const handlePageSizeChange = (newPageSize: number) => {
    setCount(newPageSize);
    setPage(0);
  };

  const defaultData = React.useMemo(() => [], []);

  const pagination = React.useMemo(
    () => ({
      pageIndex,
      pageSize
    }),
    [pageIndex, pageSize]
  );

  const table = useReactTable({
    data: dataQuery.data?.results ?? defaultData,
    columns: columns,
    pageCount: dataQuery.data?.count ?? 0,
    state: {
      pagination
    },
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true
  });
  return (
    <>
      <div className="rounded-md border mb-2">
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
                          header.getContext()
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
      <DataTablePagination
        table={table}
        pageCount={Math.ceil(dataQuery.data?.count / pageSize)}
        onPageIndexChange={handlePageIndexChange}
        onPageSizeChange={handlePageSizeChange}
        dataQuery={dataQuery}
      />
    </>
  );
}