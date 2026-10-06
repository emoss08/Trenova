"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import type { RowData } from "@tanstack/react-table";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import type { Table } from "@trenova/shared/types/data-table";
import type { ReactNode } from "react";
import {
  ChevronLeftDoubleIcon,
  ChevronLeftIcon,
  ChevronRightDoubleIcon,
  ChevronRightIcon,
} from "@trenova/shared/components/icons";

type DataTablePaginationProps<TData extends RowData> = {
  table: Table<TData>;
  mode?: "offset" | "cursor";
  hasNextPage?: boolean;
  currentPageRowCount?: number;
  totalCount?: number | null;
  pageSizeOptions?: readonly number[];
  onPageChange?: (pageIndex: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
  /** Placed before the result count, such as a keyboard-shortcuts button. */
  leading?: ReactNode;
};

const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 30, 40, 50] as const;

export function DataTablePagination<TData extends RowData>({
  table,
  mode = "offset",
  hasNextPage,
  currentPageRowCount,
  totalCount = null,
  pageSizeOptions = DEFAULT_PAGE_SIZE_OPTIONS,
  onPageChange,
  onPageSizeChange,
  leading,
}: DataTablePaginationProps<TData>) {
  const t = useT();

  const { pageIndex, pageSize } = table.state.pagination;
  const pageCount = table.getPageCount();
  const rowCount = table.getRowCount();
  const cursorMode = mode === "cursor";
  const visibleRowCount = currentPageRowCount ?? table.getRowModel().rows.length;

  const canPreviousPage = pageIndex > 0;
  const canNextPage = cursorMode ? Boolean(hasNextPage) : pageIndex < pageCount - 1;

  const handlePageChange = (newPageIndex: number) => {
    onPageChange?.(newPageIndex);
  };

  const handlePageSizeChange = (newPageSize: number) => {
    onPageSizeChange?.(newPageSize);
  };

  const startRow = visibleRowCount > 0 ? pageIndex * pageSize + 1 : 0;
  const endRow = cursorMode
    ? pageIndex * pageSize + visibleRowCount
    : Math.min((pageIndex + 1) * pageSize, rowCount);

  if (visibleRowCount < 1 && pageIndex === 0) {
    return null;
  }

  return (
    <div className="bleed:border-border bleed:min-h-10 bleed:shrink-0 bleed:border-t bleed:px-3 bleed:py-1.5 flex items-center justify-between gap-4 px-2 tabular-nums">
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        {leading}
        <span>
          {visibleRowCount < 1 ? (
            <>{t("No results on this page")}</>
          ) : (
            <>
              {t("Showing")} <span className="text-foreground font-medium">{startRow}</span> to{" "}
              <span className="text-foreground font-medium">{endRow}</span>
              {cursorMode ? (
                totalCount != null ? (
                  <>
                    {" "}
                    of{" "}
                    <span className="text-foreground font-medium">
                      {totalCount.toLocaleString()}
                    </span>{" "}
                    results
                  </>
                ) : (
                  <> results</>
                )
              ) : (
                <>
                  {" "}
                  of <span className="text-foreground font-medium">{rowCount}</span> results
                </>
              )}
            </>
          )}
        </span>
      </div>

      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground text-sm">{t("Rows per page")}</span>
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
          {!cursorMode && (
            <Button
              variant="outline"
              size="icon-sm"
              onClick={() => handlePageChange(0)}
              disabled={!canPreviousPage}
              aria-label={t("Go to first page")}
            >
              <ChevronLeftDoubleIcon className="size-4" />
            </Button>
          )}
          <Button
            variant="outline"
            size="icon-sm"
            onClick={() => handlePageChange(pageIndex - 1)}
            disabled={!canPreviousPage}
            aria-label={t("Go to previous page")}
          >
            <ChevronLeftIcon className="size-4" />
          </Button>
          <div className="flex items-center gap-1 px-2 text-sm">
            <span className="text-muted-foreground">{t("Page")}</span>
            <span className="font-medium">{pageIndex + 1}</span>
            {(!cursorMode || totalCount != null) && (
              <>
                <span className="text-muted-foreground">of</span>
                <span className="font-medium">{pageCount || 1}</span>
              </>
            )}
          </div>
          <Button
            variant="outline"
            size="icon-sm"
            onClick={() => handlePageChange(pageIndex + 1)}
            disabled={!canNextPage}
            aria-label={t("Go to next page")}
          >
            <ChevronRightIcon className="size-4" />
          </Button>
          {!cursorMode && (
            <Button
              variant="outline"
              size="icon-sm"
              onClick={() => handlePageChange(pageCount - 1)}
              disabled={!canNextPage}
              aria-label={t("Go to last page")}
            >
              <ChevronRightDoubleIcon className="size-4" />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
