import { Table } from "@tanstack/react-table";
import React from "react";
import { PaginationNavigation } from "./_pagination/pagination-navigation";
import { PaginationPageCount } from "./_pagination/pagination-page-count";
import { PaginationRowSelector } from "./_pagination/pagination-row-selector";
import { PaginationSelectedRows } from "./_pagination/pagination-selected-rows";

interface DataTablePaginationProps<TData> {
  table: Table<TData>;
}

export function PaginationInner<TData>({
  table,
}: DataTablePaginationProps<TData>) {
  const totalCount = table.getRowCount();

  const { pageSize } = table.getState().pagination;
  const totalPages = Math.ceil(totalCount / pageSize);

  return (
    totalPages > 1 && (
      <DataTablePaginationOuter>
        <PaginationSelectedRows table={table} />

        <div className="flex flex-col-reverse items-center gap-4 sm:flex-row sm:gap-6 lg:gap-8">
          {/* Row Selector */}
          <PaginationRowSelector />

          <DataTableNavigationInner>
            {/* Page Count */}
            <PaginationPageCount table={table} />

            {/* Navigation */}
            <PaginationNavigation table={table} />
          </DataTableNavigationInner>
        </div>
      </DataTablePaginationOuter>
    )
  );
}

export function DataTablePaginationOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex w-full flex-col-reverse items-center justify-between gap-4 overflow-visible sm:flex-row sm:gap-8">
      {children}
    </div>
  );
}

export function DataTableNavigationInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex items-center gap-2">{children}</div>;
}
