"use no memo";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Table } from "@tanstack/react-table";
import { ChevronFirstIcon, ChevronLastIcon, ChevronLeftIcon, ChevronRightIcon } from "lucide-react";

type DataTablePaginationProps<TData> = {
  table: Table<TData>;
  pageSizeOptions?: readonly number[];
  onPageChange?: (pageIndex: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
};

const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 30, 40, 50] as const;

export function DataTablePagination<TData>({
  table,
  pageSizeOptions = DEFAULT_PAGE_SIZE_OPTIONS,
  onPageChange,
  onPageSizeChange,
}: DataTablePaginationProps<TData>) {
  const { pageIndex, pageSize } = table.getState().pagination;
  const pageCount = table.getPageCount();
  const rowCount = table.getRowCount();

  const canPreviousPage = pageIndex > 0;
  const canNextPage = pageIndex < pageCount - 1;

  const handlePageChange = (newPageIndex: number) => {
    onPageChange?.(newPageIndex);
  };

  const handlePageSizeChange = (newPageSize: number) => {
    onPageSizeChange?.(newPageSize);
  };

  const startRow = rowCount > 0 ? pageIndex * pageSize + 1 : 0;
  const endRow = Math.min((pageIndex + 1) * pageSize, rowCount);

  if (rowCount < 1) {
    return null;
  }

  return (
    <div className="flex items-center justify-between gap-4 px-2">
      <div className="text-sm text-muted-foreground">
        Showing <span className="font-medium text-foreground">{startRow}</span> to{" "}
        <span className="font-medium text-foreground">{endRow}</span> of{" "}
        <span className="font-medium text-foreground">{rowCount}</span> results
      </div>

      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">Rows per page</span>
          <Select
            value={String(pageSize)}
            onValueChange={(value) => handlePageSizeChange(Number(value))}
          >
            <SelectTrigger className="w-[60px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {pageSizeOptions.map((size) => (
                  <SelectItem key={size} value={String(size)}>
                    {size}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        <div className="flex items-center gap-1">
          <Button
            variant="outline"
            size="icon-sm"
            onClick={() => handlePageChange(0)}
            disabled={!canPreviousPage}
            aria-label="Go to first page"
          >
            <ChevronFirstIcon className="size-4" />
          </Button>
          <Button
            variant="outline"
            size="icon-sm"
            onClick={() => handlePageChange(pageIndex - 1)}
            disabled={!canPreviousPage}
            aria-label="Go to previous page"
          >
            <ChevronLeftIcon className="size-4" />
          </Button>
          <div className="flex items-center gap-1 px-2 text-sm">
            <span className="text-muted-foreground">Page</span>
            <span className="font-medium">{pageIndex + 1}</span>
            <span className="text-muted-foreground">of</span>
            <span className="font-medium">{pageCount || 1}</span>
          </div>
          <Button
            variant="outline"
            size="icon-sm"
            onClick={() => handlePageChange(pageIndex + 1)}
            disabled={!canNextPage}
            aria-label="Go to next page"
          >
            <ChevronRightIcon className="size-4" />
          </Button>
          <Button
            variant="outline"
            size="icon-sm"
            onClick={() => handlePageChange(pageCount - 1)}
            disabled={!canNextPage}
            aria-label="Go to last page"
          >
            <ChevronLastIcon className="size-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
