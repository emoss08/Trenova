import { useT } from "@trenova/shared/i18n/use-t";
import { useRichT } from "@trenova/shared/i18n/rich";
import { formatNumber } from "@trenova/shared/i18n/format";
import { formatRowTotal } from "@trenova/shared/lib/row-total";
import type { RowData } from "@tanstack/react-table";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
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
import { DEFAULT_PAGE_SIZE_OPTIONS } from "@/hooks/data-table/use-data-table-state";
import { TABLE_PAGER_CLASS } from "../data-table-chrome";
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
  /** False when totalCount is the server's estimate of a list too large to count. */
  totalCountExact?: boolean;
  pageSizeOptions?: readonly number[];
  onPageChange?: (pageIndex: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
  /** Placed before the result count, such as a keyboard-shortcuts button. */
  leading?: ReactNode;
  /** The figures here come from the server and are about to change: hold their place. */
  figuresLoading?: boolean;
};

export function DataTablePagination<TData extends RowData>({
  table,
  mode = "offset",
  hasNextPage,
  currentPageRowCount,
  totalCount = null,
  totalCountExact = true,
  pageSizeOptions = DEFAULT_PAGE_SIZE_OPTIONS,
  onPageChange,
  onPageSizeChange,
  leading,
  figuresLoading = false,
}: DataTablePaginationProps<TData>) {
  const t = useT();
  const rt = useRichT();

  const { pageIndex, pageSize } = table.state.pagination;
  const pageCount = table.getPageCount();
  const rowCount = table.getRowCount();
  const cursorMode = mode === "cursor";
  const visibleRowCount = currentPageRowCount ?? table.getRowModel().rows.length;
  // An estimated total says nothing about where the list ends, so offset paging over one
  // follows full pages and never jumps to a last page computed from it.
  const offsetEstimate = !cursorMode && !totalCountExact;

  const canPreviousPage = pageIndex > 0;
  const canNextPage = cursorMode
    ? Boolean(hasNextPage)
    : offsetEstimate
      ? visibleRowCount >= pageSize
      : pageIndex < pageCount - 1;

  const handlePageChange = (newPageIndex: number) => {
    onPageChange?.(newPageIndex);
  };

  const handlePageSizeChange = (newPageSize: number) => {
    onPageSizeChange?.(newPageSize);
  };

  const startRow = visibleRowCount > 0 ? pageIndex * pageSize + 1 : 0;
  const endRow =
    cursorMode || offsetEstimate
      ? pageIndex * pageSize + visibleRowCount
      : Math.min((pageIndex + 1) * pageSize, rowCount);

  if (visibleRowCount < 1 && pageIndex === 0) {
    return null;
  }

  const figure = { b: (c: ReactNode) => <span className="text-foreground font-medium">{c}</span> };
  const start = formatNumber(startRow);
  const end = formatNumber(endRow);
  const total = cursorMode ? totalCount : rowCount;
  const exact = totalCountExact;
  const shownTotal = total != null ? formatRowTotal(total, exact) : null;
  const shownPages = formatRowTotal(pageCount || 1, exact);

  return (
    <div className={TABLE_PAGER_CLASS}>
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        {leading}
        {figuresLoading ? <Skeleton data-slot="pager-figures" className="h-3.5 w-60" /> : null}
        <span className={figuresLoading ? "hidden" : undefined}>
          {visibleRowCount < 1
            ? t("No results on this page")
            : shownTotal === null
              ? rt("Showing <b>{0}</b> to <b>{1}</b> results", figure, start, end)
              : exact
                ? rt(
                    "Showing <b>{0}</b> to <b>{1}</b> of <b>{2}</b> results",
                    figure,
                    start,
                    end,
                    shownTotal,
                  )
                : rt(
                    "Showing <b>{0}</b> to <b>{1}</b> of about <b>{2}</b> results",
                    figure,
                    start,
                    end,
                    shownTotal,
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
          {figuresLoading ? (
            <Skeleton data-slot="pager-page" className="mx-2 h-3.5 w-28.75" />
          ) : null}
          <div className={cn("text-muted-foreground px-2 text-sm", figuresLoading && "hidden")}>
            {shownTotal === null || offsetEstimate
              ? rt("Page <b>{0}</b>", figure, formatNumber(pageIndex + 1))
              : exact
                ? rt(
                    "Page <b>{0}</b> of <b>{1}</b>",
                    figure,
                    formatNumber(pageIndex + 1),
                    shownPages,
                  )
                : rt(
                    "Page <b>{0}</b> of about <b>{1}</b>",
                    figure,
                    formatNumber(pageIndex + 1),
                    shownPages,
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
          {!cursorMode && !offsetEstimate && (
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
