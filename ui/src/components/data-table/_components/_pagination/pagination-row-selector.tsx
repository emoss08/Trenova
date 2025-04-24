import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { DataTableSearchParams } from "@/hooks/use-data-table-state";
import { useQueryState } from "nuqs";
import React, { useCallback } from "react";
import { toast } from "sonner";

const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 30, 40, 50] as const;
const MIN_PAGE_SIZE = 1;
const MAX_PAGE_SIZE = 100;

export function PaginationRowSelector() {
  const [pageSize, setPageSize] = useQueryState(
    "pageSize",
    DataTableSearchParams.pageSize.withOptions({
      shallow: false,
    }),
  );

  // Use URL state as single source of truth
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [_, setPage] = useQueryState(
    "page",
    DataTableSearchParams.page.withOptions({
      shallow: false,
    }),
  );

  const normalizedPageSizeOptions = React.useMemo(() => {
    return [...new Set(DEFAULT_PAGE_SIZE_OPTIONS)]
      .filter((size) => size >= MIN_PAGE_SIZE && size <= MAX_PAGE_SIZE)
      .sort((a, b) => a - b);
  }, []);

  const onPageSizeChange = useCallback(
    (newPageSize: number) => {
      setPage(1);
      setPageSize(newPageSize);
    },
    [setPage, setPageSize],
  );

  // Memoize handlers to prevent unnecessary re-renders
  const onPageChange = useCallback(
    (newPage: number) => {
      setPage(newPage);
    },
    [setPage],
  );

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

        onPageSizeChange(newPageSize);
        // Reset to first page when changing page size
        onPageChange(1);
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

  return (
    <div className="flex items-center space-x-2">
      <p className="whitespace-nowrap text-sm font-medium">Rows per page</p>
      <Select
        defaultValue={String(pageSize)}
        onValueChange={handlePageSizeChange}
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
  );
}
