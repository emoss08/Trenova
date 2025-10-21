/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import type { Table } from "@tanstack/react-table";

export function PaginationPageCount<TData>({ table }: { table: Table<TData> }) {
  const totalCount = table.getRowCount();

  const { pageIndex, pageSize } = table.getState().pagination;
  const currentPage = pageIndex + 1;
  const totalPages = Math.ceil(totalCount / pageSize);

  return (
    <div
      aria-live="polite"
      className="flex w-full items-center justify-center text-sm font-medium"
    >
      Page {currentPage} of {totalPages}
    </div>
  );
}
