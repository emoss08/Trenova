import { useT } from "@trenova/shared/i18n/use-t";
import { useRichT } from "@trenova/shared/i18n/rich";
import { formatNumber } from "@trenova/shared/i18n/format";
import { formatRowTotal } from "@trenova/shared/lib/row-total";
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
import { DEFAULT_PAGE_SIZE_OPTIONS } from "@/hooks/data-table/use-data-table-state";
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
}: DataTablePaginationProps<TData>) {
  const t = useT();
  const rt = useRichT();

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

  const figure = { b: (c: ReactNode) => <span className="text-foreground font-medium">{c}</span> };
  const start = formatNumber(startRow);
  const end = formatNumber(endRow);
  const total = cursorMode ? totalCount : rowCount;
  const exact = !cursorMode || totalCountExact;
  const shownTotal = total != null ? formatRowTotal(total, exact) : null;
  const shownPages = formatRowTotal(pageCount || 1, exact);

  return (
    <div className="bleed:border-border bleed:min-h-10 bleed:shrink-0 bleed:border-t bleed:px-3 bleed:py-1.5 flex items-center justify-between gap-4 px-2 tabular-nums">
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        {leading}
        <span>
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
          <div className="text-muted-foreground px-2 text-sm">
            {shownTotal === null
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
