/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import type { Table } from "@tanstack/react-table";
import { useQueryState } from "nuqs";
import React, { useCallback } from "react";

const VISIBLE_PAGES = 5;

function calculateVisiblePages(
  currentPage: number,
  totalPages: number,
  visiblePages: number = VISIBLE_PAGES,
) {
  // Always show first and last page
  if (totalPages <= visiblePages) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }

  const sidePages = Math.floor((visiblePages - 2) / 2);
  const startPage = Math.max(2, currentPage - sidePages);
  const endPage = Math.min(totalPages - 1, currentPage + sidePages);

  const pages: (number | "ellipsis")[] = [1];

  if (startPage > 2) {
    pages.push("ellipsis");
  }

  for (let i = startPage; i <= endPage; i++) {
    if (i !== 1 && i !== totalPages) {
      pages.push(i);
    }
  }

  if (endPage < totalPages - 1) {
    pages.push("ellipsis");
  }

  if (totalPages > 1) {
    pages.push(totalPages);
  }

  return pages;
}

export function PaginationNavigation<TData>({
  table,
}: {
  table: Table<TData>;
}) {
  // Use URL state as single source of truth
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [_, setPage] = useQueryState("page", searchParamsParser.page);

  const { pageIndex, pageSize } = table.getState().pagination;
  const currentPage = pageIndex + 1;
  const totalCount = table.getRowCount();
  const totalPages = Math.ceil(totalCount / pageSize);

  const paginationInfo = React.useMemo(() => {
    const firstRow = Math.min((currentPage - 1) * pageSize + 1, totalCount);
    const lastRow = Math.min(currentPage * pageSize, totalCount);

    return {
      firstRow,
      lastRow,
      totalPages,
      visiblePages: calculateVisiblePages(currentPage, totalPages),
    };
  }, [currentPage, pageSize, totalCount, totalPages]);

  // Memoize handlers to prevent unnecessary re-renders
  const handlePageChange = useCallback(
    (newPage: number) => {
      setPage(newPage);
    },
    [setPage],
  );

  return (
    <Pagination>
      <PaginationContent>
        <PaginationPrevious
          onClick={() => handlePageChange(currentPage - 1)}
          className="h-8"
          variant="ghost"
          disabled={currentPage === 1}
        />

        {paginationInfo.visiblePages.map((page, index) => {
          if (page === "ellipsis") {
            return <PaginationEllipsis key={`ellipsis-${index}`} />;
          }

          const isActive = page === currentPage;

          return (
            <PaginationLink
              key={page}
              onClick={() => handlePageChange(page)}
              isActive={isActive}
              className="h-8"
            >
              {page}
            </PaginationLink>
          );
        })}

        <PaginationNext
          onClick={() => handlePageChange(currentPage + 1)}
          className="h-8"
          variant="ghost"
        />
      </PaginationContent>
    </Pagination>
  );
}
