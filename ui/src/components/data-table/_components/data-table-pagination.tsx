import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Table } from "@tanstack/react-table";
import React from "react";
import { toast } from "sonner";

import { PaginationLink } from "@/components/ui/pagination";

interface DataTablePaginationProps<TData> {
  table: Table<TData>;
  totalCount: number;
  pageSizeOptions?: Readonly<number[]>;
  isLoading?: boolean;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
}

const DEFAULT_PAGE_SIZE_OPTIONS = [10, 25, 50, 100];
const MIN_PAGE_SIZE = 1;
const MAX_PAGE_SIZE = 100;
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

export function DataTablePagination<TData>({
  table,
  totalCount,
  pageSizeOptions = DEFAULT_PAGE_SIZE_OPTIONS,
  isLoading = false,
  onPageChange,
  onPageSizeChange,
}: DataTablePaginationProps<TData>) {
  const [isTransitioning, startTransition] = React.useTransition();

  const { pageIndex, pageSize } = table.getState().pagination;
  const currentPage = pageIndex + 1;
  const totalPages = Math.ceil(totalCount / pageSize);

  const normalizedPageSizeOptions = React.useMemo(() => {
    return [...new Set(pageSizeOptions)]
      .filter((size) => size >= MIN_PAGE_SIZE && size <= MAX_PAGE_SIZE)
      .sort((a, b) => a - b);
  }, [pageSizeOptions]);

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

  const handlePageSizeChange = React.useCallback(
    (value: string) => {
      try {
        const newPageSize = Number(value);
        if (
          isNaN(newPageSize) ||
          newPageSize < MIN_PAGE_SIZE ||
          newPageSize > MAX_PAGE_SIZE
        ) {
          throw new Error("Invalid page size");
        }

        startTransition(() => {
          onPageSizeChange(newPageSize);
          // Reset to first page when changing page size
          onPageChange(1);
        });
      } catch (error) {
        console.error("Failed to update page size:", error);
        toast.error("Failed to update page size", {
          description:
            "Please try again or contact support if the issue persists.",
        });
      }
    },
    [onPageChange, onPageSizeChange],
  );

  const handlePageChange = React.useCallback(
    (page: number) => {
      if (page < 1 || page > totalPages || page === currentPage) return;

      try {
        startTransition(() => {
          onPageChange(page);
        });
      } catch (error) {
        console.error("Failed to update page:", error);
        toast.error("Failed to update page", {
          description:
            "Please try again or contact support if the issue persists.",
        });
      }
    },
    [currentPage, onPageChange, totalPages],
  );

  const selectedRows = table.getFilteredSelectedRowModel().rows.length;
  const totalRows = table.getFilteredRowModel().rows.length;

  return (
    totalPages > 1 && (
      <div className="flex w-full flex-col-reverse items-center justify-between gap-4 overflow-auto sm:flex-row sm:gap-8">
        <div className="flex-1 whitespace-nowrap text-sm text-muted-foreground">
          {selectedRows > 0 ? (
            <span>
              {selectedRows} of {totalRows} row(s) selected
            </span>
          ) : (
            <span>
              Showing {paginationInfo.firstRow} to {paginationInfo.lastRow} of{" "}
              {totalCount} entries
            </span>
          )}
        </div>

        <div className="flex flex-col-reverse items-center gap-4 sm:flex-row sm:gap-6 lg:gap-8">
          <div className="flex items-center space-x-2">
            <p className="whitespace-nowrap text-sm font-medium">
              Rows per page
            </p>
            <Select
              defaultValue={String(pageSize)}
              onValueChange={handlePageSizeChange}
              disabled={isLoading || isTransitioning}
            >
              <SelectTrigger className="h-8 w-16">
                <SelectValue />
              </SelectTrigger>
              <SelectContent align="center" side="top">
                {normalizedPageSizeOptions.map((size) => (
                  <SelectItem
                    key={size}
                    value={String(size)}
                    className="cursor-pointer"
                  >
                    {size}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <>
            <div
              aria-live="polite"
              className="flex w-full items-center justify-center text-sm font-medium"
            >
              Page {currentPage} of {paginationInfo.totalPages}
            </div>

            <Pagination>
              <PaginationContent>
                <PaginationPrevious
                  onClick={() => handlePageChange(currentPage - 1)}
                  className="h-8"
                  variant="ghost"
                  disabled={currentPage === 1 || isLoading || isTransitioning}
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
                      disabled={isLoading || isTransitioning}
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
                  disabled={
                    currentPage === paginationInfo.totalPages ||
                    isLoading ||
                    isTransitioning
                  }
                />
              </PaginationContent>
            </Pagination>
          </>
        </div>
      </div>
    )
  );
}
