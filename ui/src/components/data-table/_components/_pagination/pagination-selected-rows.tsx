/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import type { Table } from "@tanstack/react-table";

export function PaginationSelectedRows<TData>({
  table,
}: {
  table: Table<TData>;
}) {
  const { pageIndex, pageSize } = table.getState().pagination;

  const selectedRows = table.getFilteredSelectedRowModel().rows.length;
  const totalRows = table.getFilteredRowModel().rows.length;
  const totalCount = table.getRowCount();
  const currentPage = pageIndex + 1;

  const firstRow = Math.min((currentPage - 1) * pageSize + 1, totalCount);
  const lastRow = Math.min(currentPage * pageSize, totalCount);

  return (
    <div className="flex-1 whitespace-nowrap text-sm text-muted-foreground">
      {selectedRows > 0 ? (
        <span>
          {selectedRows} of {totalRows} row(s) selected
        </span>
      ) : (
        <span>
          Showing {firstRow} to {lastRow} of {totalCount} entries
        </span>
      )}
    </div>
  );
}
