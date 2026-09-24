"use no memo";
import { SectionPanelQuiet } from "@/components/section-panel";
import { flexRender, useTable, type RowData } from "@tanstack/react-table";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { dataTableFeatures } from "@trenova/shared/lib/table-features";
import { cn } from "@trenova/shared/lib/utils";
import type { ColumnDef, Row } from "@trenova/shared/types/data-table";
import { ChevronRightIcon, CircleAlertIcon } from "lucide-react";
import { Fragment, useMemo, useState, type ReactNode } from "react";
import { DataTablePagination } from "./_components/data-table-pagination";

/** Rows a first load draws while nothing has landed; enough to hold the panel's height. */
const SKELETON_ROWS = 8;
const EMPTY_ROWS: never[] = [];
const EXPAND_COLUMN_ID = "__expand";

export const SECTION_TABLE_PAGE_SIZES = [10, 25, 50] as const;
export const SECTION_TABLE_PAGE_SIZE = 25;

type PageControls = {
  pageIndex: number;
  pageSize: number;
  onPageChange: (pageIndex: number) => void;
  onPageSizeChange: (pageSize: number) => void;
};

export type SectionTablePagination =
  | (PageControls & {
      /** The server pages by cursor: next is known only from the page just read. */
      mode: "cursor";
      hasNextPage: boolean;
      totalCount: number | null;
    })
  | (PageControls & {
      /** Every row is already here; the table shows a page of them. */
      mode: "offset";
      totalCount: number;
    });

export type SectionTableProps<TData extends RowData> = {
  /** Names the table for assistive technology. */
  label: string;
  columns: ColumnDef<TData>[];
  /** The rows of the current page only. */
  rows: readonly TData[] | undefined;
  getRowId: (row: TData) => string;
  /** Nothing to show yet. */
  isLoading: boolean;
  /** A newer page or filter is on its way; what is shown is the previous answer. */
  isRefreshing?: boolean;
  /** Said in place of the rows when the read failed. */
  error?: string | null;
  onRetry?: () => void;
  /** Said when the page has no rows. */
  empty: ReactNode;
  /**
   * Long text that would make a wide cell: rationale, conditions. Rows that
   * have it open underneath themselves rather than stretching the table.
   */
  renderDetails?: (row: TData) => ReactNode;
  /** What a row is called, for the control that opens its details. */
  rowLabel?: (row: TData) => string;
  pagination: SectionTablePagination;
};

/**
 * A data table that shares its screen with other panels: the list page's
 * rows, header and pagination at the row rhythm tokens, without the URL
 * state, saved views and toolbar that belong to a page that is only a table.
 * It is handed one page of rows, so nothing it draws grows with the data.
 */
export function SectionTable<TData extends RowData>({
  label,
  columns,
  rows,
  getRowId,
  isLoading,
  isRefreshing = false,
  error = null,
  onRetry,
  empty,
  renderDetails,
  rowLabel,
  pagination,
}: SectionTableProps<TData>) {
  "use no memo";
  const t = useT();
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(() => new Set());

  const toggle = (id: string) => {
    setExpanded((current) => {
      const next = new Set(current);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const tableColumns = useMemo<ColumnDef<TData>[]>(() => {
    if (!renderDetails) {
      return columns;
    }
    const expander: ColumnDef<TData> = {
      id: EXPAND_COLUMN_ID,
      header: () => <span className="sr-only">{t("Details")}</span>,
      meta: { headerClassName: "w-8 pr-0", cellClassName: "w-8 pr-0" },
    };
    return [expander, ...columns];
  }, [columns, renderDetails, t]);

  const data = (rows ?? EMPTY_ROWS) as TData[];
  const { pageIndex, pageSize } = pagination;
  const rowCount =
    pagination.mode === "offset"
      ? pagination.totalCount
      : (pagination.totalCount ??
        pageIndex * pageSize + data.length + (pagination.hasNextPage ? pageSize : 0));
  const pageCount = Math.max(1, Math.ceil(rowCount / pageSize));

  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useTable({
    features: dataTableFeatures,
    data,
    columns: tableColumns,
    getRowId: (row) => getRowId(row),
    manualPagination: true,
    manualSorting: true,
    pageCount,
    rowCount,
    state: { pagination: { pageIndex, pageSize } },
  });

  const columnCount = tableColumns.length;

  if (error) {
    return (
      <div className="p-3">
        <Alert variant="destructive" size="sm">
          <CircleAlertIcon />
          <AlertDescription>{error}</AlertDescription>
          {onRetry ? (
            <AlertAction>
              <Button variant="outline" size="xs" onClick={onRetry}>
                {t("Try again")}
              </Button>
            </AlertAction>
          ) : null}
        </Alert>
      </div>
    );
  }

  if (!isLoading && data.length === 0 && pageIndex === 0) {
    return <SectionPanelQuiet>{empty}</SectionPanelQuiet>;
  }

  return (
    <div className="flex min-w-0 flex-col">
      <Table
        aria-label={label}
        aria-busy={isLoading || isRefreshing}
        className="border-separate border-spacing-0"
        containerClassName="max-h-[60vh]"
      >
        <TableHeader className="sticky top-0 z-10">
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id} className="hover:bg-transparent">
              {headerGroup.headers.map((header) => (
                <TableHead
                  key={header.id}
                  className={cn(
                    "border-border border-b",
                    header.column.columnDef.meta?.headerClassName,
                  )}
                >
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, header.getContext())}
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody className={cn("transition-opacity", isRefreshing && !isLoading && "opacity-60")}>
          {isLoading ? (
            <SkeletonRows columns={columnCount} rows={Math.min(pageSize, SKELETON_ROWS)} />
          ) : (
            table
              .getRowModel()
              .rows.map((row) => (
                <SectionTableRow
                  key={row.id}
                  row={row}
                  columnCount={columnCount}
                  expanded={expanded.has(row.id)}
                  onToggle={toggle}
                  renderDetails={renderDetails}
                  rowLabel={rowLabel}
                />
              ))
          )}
        </TableBody>
      </Table>
      {isLoading ? null : (
        <div className="border-border border-t py-1.5">
          <DataTablePagination
            table={table}
            mode={pagination.mode}
            hasNextPage={pagination.mode === "cursor" ? pagination.hasNextPage : undefined}
            currentPageRowCount={data.length}
            totalCount={pagination.totalCount}
            pageSizeOptions={SECTION_TABLE_PAGE_SIZES}
            onPageChange={pagination.onPageChange}
            onPageSizeChange={pagination.onPageSizeChange}
          />
        </div>
      )}
    </div>
  );
}

type SectionTableRowProps<TData extends RowData> = {
  row: Row<TData>;
  columnCount: number;
  expanded: boolean;
  onToggle: (id: string) => void;
  renderDetails?: (row: TData) => ReactNode;
  rowLabel?: (row: TData) => string;
};

function SectionTableRow<TData extends RowData>({
  row,
  columnCount,
  expanded,
  onToggle,
  renderDetails,
  rowLabel,
}: SectionTableRowProps<TData>) {
  const t = useT();
  const detailsId = `section-table-details-${row.id}`;
  const name = rowLabel?.(row.original) ?? row.id;

  return (
    <Fragment>
      <TableRow data-state={expanded ? "selected" : undefined}>
        {row.getVisibleCells().map((cell) => (
          <TableCell
            key={cell.id}
            className={cn("border-border border-b", cell.column.columnDef.meta?.cellClassName)}
          >
            {cell.column.id === EXPAND_COLUMN_ID ? (
              <Button
                variant="ghost"
                size="icon-xs"
                aria-expanded={expanded}
                aria-controls={expanded ? detailsId : undefined}
                aria-label={
                  expanded ? t("Hide details for {0}", name) : t("Show details for {0}", name)
                }
                onClick={() => onToggle(row.id)}
              >
                <ChevronRightIcon
                  className={cn("size-3.5 transition-transform", expanded && "rotate-90")}
                />
              </Button>
            ) : (
              flexRender(cell.column.columnDef.cell, cell.getContext())
            )}
          </TableCell>
        ))}
      </TableRow>
      {expanded && renderDetails ? (
        <TableRow id={detailsId} className="h-auto hover:bg-transparent">
          <TableCell
            colSpan={columnCount}
            className="border-border bg-sunken border-b px-(--cell-px) py-3 whitespace-normal"
          >
            {renderDetails(row.original)}
          </TableCell>
        </TableRow>
      ) : null}
    </Fragment>
  );
}

function SkeletonRows({ columns, rows }: { columns: number; rows: number }) {
  return Array.from({ length: rows }, (_, rowIndex) => (
    <TableRow key={rowIndex} className="hover:bg-transparent">
      {Array.from({ length: columns }, (_, columnIndex) => (
        <TableCell key={columnIndex} className="border-border border-b">
          <Skeleton className="h-3.5 w-full" />
        </TableCell>
      ))}
    </TableRow>
  ));
}
