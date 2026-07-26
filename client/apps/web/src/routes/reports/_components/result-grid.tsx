import { Spinner } from "@trenova/shared/components/ui/spinner";
import type { ReportPreviewColumn } from "@/lib/graphql/reports";
import {
  formatReportValue,
  isNumericDisplay,
  reportNumericValue,
  reportValueTone,
  type ReportDisplayTone,
} from "@trenova/shared/lib/report-format";
import { cn } from "@trenova/shared/lib/utils";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ArrowDownIcon, ArrowUpIcon } from "lucide-react";
import { useMemo, useRef } from "react";

export type GridSort = { columnId: string; direction: "asc" | "desc" };

export type ResultGridProps = {
  columns: ReportPreviewColumn[];
  rows: unknown[][];
  totals?: unknown[] | null;
  loading?: boolean;
  /** Column ids to leave out of the render without touching the data. */
  hiddenColumnIds?: ReadonlySet<string>;
  sort?: GridSort | null;
  onSortChange?: (sort: GridSort | null) => void;
  /** Receives indexes into the original, unsorted row array. */
  onCellClick?: (rowIndex: number, columnIndex: number) => void;
  quickFilter?: string;
  emptyMessage?: string;
  density?: "compact" | "default";
  className?: string;
};

const ROW_HEIGHT = { compact: 26, default: 30 } as const;
const COLUMN_WIDTH = "w-44";

/** Conditional-formatting emphasis, resolved from the column's rules. */
const TONE_CLASS: Record<Exclude<ReportDisplayTone, "">, string> = {
  positive: "text-emerald-600 dark:text-emerald-400",
  warning: "text-amber-600 dark:text-amber-400",
  negative: "text-destructive",
  neutral: "text-foreground",
};

function toneClass(tone: ReportDisplayTone): string | undefined {
  return tone === "" ? undefined : `${TONE_CLASS[tone]} font-medium`;
}

type IndexedRow = { row: unknown[]; index: number };

// Report cells only ever arrive as primitives — decimals as strings, epochs as
// numbers — so comparison stays on that ground rather than stringifying objects.
function sortableText(value: unknown): string {
  return typeof value === "string" || typeof value === "number" || typeof value === "boolean"
    ? String(value)
    : "";
}

function compareValues(a: unknown, b: unknown): number {
  const numericA = reportNumericValue(a);
  const numericB = reportNumericValue(b);
  if (numericA !== null && numericB !== null) return numericA - numericB;
  if (a === null || a === undefined) return -1;
  if (b === null || b === undefined) return 1;
  return sortableText(a).localeCompare(sortableText(b));
}

/**
 * Sorting and filtering happen on the rows already fetched — the server has
 * already applied the report's own ordering and limit, so this is about
 * looking through a result rather than re-running it.
 */
function useVisibleRows(
  columns: ReportPreviewColumn[],
  rows: unknown[][],
  sort: GridSort | null | undefined,
  quickFilter: string | undefined,
): IndexedRow[] {
  return useMemo(() => {
    let indexed: IndexedRow[] = rows.map((row, index) => ({ row, index }));

    const needle = quickFilter?.trim().toLowerCase();
    if (needle) {
      indexed = indexed.filter(({ row }) =>
        columns.some((column, columnIndex) =>
          formatReportValue(row[columnIndex], column.display).toLowerCase().includes(needle),
        ),
      );
    }

    if (sort) {
      const columnIndex = columns.findIndex((column) => column.id === sort.columnId);
      if (columnIndex >= 0) {
        const direction = sort.direction === "asc" ? 1 : -1;
        indexed = [...indexed].sort(
          (a, b) => compareValues(a.row[columnIndex], b.row[columnIndex]) * direction,
        );
      }
    }

    return indexed;
  }, [columns, rows, sort, quickFilter]);
}

function nextSort(current: GridSort | null | undefined, columnId: string): GridSort | null {
  if (!current || current.columnId !== columnId) return { columnId, direction: "desc" };
  if (current.direction === "desc") return { columnId, direction: "asc" };
  return null;
}

export function ResultGrid({
  columns,
  rows,
  totals,
  loading,
  hiddenColumnIds,
  sort,
  onSortChange,
  onCellClick,
  quickFilter,
  emptyMessage = "This report returned no rows.",
  density = "default",
  className,
}: ResultGridProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const rowHeight = ROW_HEIGHT[density];

  const visibleColumns = useMemo(
    () =>
      columns
        .map((column, index) => ({ column, index }))
        .filter(({ column }) => !hiddenColumnIds?.has(column.id)),
    [columns, hiddenColumnIds],
  );

  const visibleRows = useVisibleRows(columns, rows, sort, quickFilter);

  const virtualizer = useVirtualizer({
    count: visibleRows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => rowHeight,
    overscan: 20,
  });

  return (
    <div className={cn("relative flex min-h-0 flex-1 flex-col", className)}>
      <div ref={scrollRef} className="min-h-0 flex-1 overflow-auto">
        <div className="min-w-fit">
          <div className="sticky top-0 z-10 flex border-b border-border bg-background/95 backdrop-blur-sm">
            {visibleColumns.map(({ column }) => {
              const numeric = isNumericDisplay(column.display);
              const active = sort?.columnId === column.id;
              return (
                <button
                  key={column.id}
                  type="button"
                  disabled={!onSortChange}
                  onClick={() => onSortChange?.(nextSort(sort, column.id))}
                  className={cn(
                    COLUMN_WIDTH,
                    "flex shrink-0 items-center gap-1 px-3 py-1.5 text-2xs font-medium tracking-wide text-muted-foreground uppercase",
                    numeric && "justify-end",
                    onSortChange && "transition-colors hover:text-foreground",
                    active && "text-foreground",
                  )}
                  title={column.label}
                >
                  <span className="truncate">{column.label}</span>
                  {active &&
                    (sort?.direction === "asc" ? (
                      <ArrowUpIcon className="size-3 shrink-0" />
                    ) : (
                      <ArrowDownIcon className="size-3 shrink-0" />
                    ))}
                </button>
              );
            })}
          </div>

          <div
            className={cn("relative transition-opacity", loading && "opacity-50")}
            style={{ height: virtualizer.getTotalSize() }}
          >
            {virtualizer.getVirtualItems().map((virtualRow) => {
              const entry = visibleRows[virtualRow.index];
              return (
                <div
                  key={virtualRow.key}
                  className="absolute top-0 left-0 flex w-full border-b border-border/40 transition-colors hover:bg-accent/40"
                  style={{
                    height: virtualRow.size,
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
                >
                  {visibleColumns.map(({ column, index: columnIndex }) => {
                    const text = formatReportValue(entry.row[columnIndex], column.display);
                    const numeric = isNumericDisplay(column.display);
                    const clickable = Boolean(onCellClick);
                    const tone = reportValueTone(entry.row[columnIndex], column.display);
                    return (
                      <div
                        key={column.id}
                        className={cn(
                          COLUMN_WIDTH,
                          "shrink-0 truncate px-3 py-1.5 text-xs",
                          numeric ? "text-right font-mono tabular-nums" : "text-foreground/90",
                          toneClass(tone),
                          clickable &&
                            "cursor-pointer decoration-dotted underline-offset-4 hover:underline",
                        )}
                        title={text}
                        onClick={
                          clickable ? () => onCellClick?.(entry.index, columnIndex) : undefined
                        }
                      >
                        {text}
                      </div>
                    );
                  })}
                </div>
              );
            })}
          </div>

          {visibleRows.length === 0 && (
            <p className="p-8 text-center text-xs text-muted-foreground">{emptyMessage}</p>
          )}
        </div>
      </div>

      {totals && (
        <div className="flex shrink-0 border-t-2 border-border bg-muted/40">
          {visibleColumns.map(({ column, index: columnIndex }) => {
            const numeric = isNumericDisplay(column.display);
            const value = totals[columnIndex];
            const text =
              value === null || value === undefined
                ? columnIndex === 0
                  ? "Total"
                  : ""
                : formatReportValue(value, column.display);
            return (
              <div
                key={column.id}
                className={cn(
                  COLUMN_WIDTH,
                  "shrink-0 truncate px-3 py-1.5 text-xs font-semibold",
                  numeric && "text-right font-mono tabular-nums",
                )}
                title={text}
              >
                {text}
              </div>
            );
          })}
        </div>
      )}

      {loading && (
        <div className="pointer-events-none absolute top-2 right-2">
          <Spinner className="size-3.5 text-muted-foreground" />
        </div>
      )}
    </div>
  );
}
