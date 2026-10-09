import type { DataTableAggregateValues, DataTableTotals } from "@/hooks/data-table/use-data-table-insights";
import { columnCellStyle, pinnedCellClass } from "@/lib/data-table";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { TableCell, TableFooter, TableRow } from "@trenova/shared/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type {
  Column,
  DataTableAggregateKind,
  DataTableColumnAggregate,
} from "@trenova/shared/types/data-table";
import type { RowData } from "@tanstack/react-table";

type DataTableTotalsRowProps<TData extends RowData> = {
  columns: Column<TData, unknown>[];
  totals: DataTableTotals | undefined;
  loading: boolean;
  /** The fields the server totals; a column on any other field is left blank. */
  summable: ReadonlySet<string>;
};

const NUMBER_FORMAT = new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 });

const TOTAL_SIZE: Record<NonNullable<DataTableColumnAggregate["size"]>, string> = {
  sm: "text-sm",
  md: "text-lg",
  lg: "text-xl",
};

function formatTotal(raw: string | null | undefined, format: DataTableColumnAggregate["format"]) {
  if (raw === null || raw === undefined) return "—";
  const value = Number(raw);
  if (!Number.isFinite(value)) return raw;
  return format === "money" ? formatCurrency(value) : NUMBER_FORMAT.format(value);
}

function pick(value: DataTableAggregateValues | undefined, kind: DataTableAggregateKind) {
  if (!value) return undefined;
  switch (kind) {
    case "average":
      return value.average;
    case "min":
      return value.min;
    case "max":
      return value.max;
    default:
      return value.sum;
  }
}

/** Where a column's total comes from, or null when it has none the server can give. */
export function aggregateFieldOf<TData extends RowData>(column: Column<TData, unknown>) {
  const meta = column.columnDef.meta;
  if (!meta?.aggregate) return null;
  return meta.aggregate.field ?? meta.apiField ?? null;
}

/**
 * The table's totals: each totalled column summed (or averaged) across every row the
 * filters match, on every page, with the row count. It sticks to the bottom of the
 * table so it stays in view while the rows scroll.
 */
export function DataTableTotalsRow<TData extends RowData>({
  columns,
  totals,
  loading,
  summable,
}: DataTableTotalsRowProps<TData>) {
  const t = useT();

  return (
    <TableFooter className="bg-canvas sticky bottom-0 z-15 font-normal">
      <TableRow className="hover:bg-transparent">
        {columns.map((column) => {
          const field = aggregateFieldOf(column);
          const aggregate = column.columnDef.meta?.aggregate;
          const value = field ? totals?.byField.get(field) : undefined;
          const kind = aggregate?.kind ?? "sum";
          const shown = field && summable.has(field);

          return (
            <TableCell
              key={column.id}
              data-column-id={column.id}
              className={cn(
                "bg-canvas border-border border-t text-sm tabular-nums",
                // Room under the totals for the sideways scrollbar, so it never covers them.
                "in-data-has-overflow-x:pb-3.5",
                shown && "text-right",
                pinnedCellClass(column),
              )}
              style={columnCellStyle(column)}
            >
              {shown ? (
                loading && !totals ? (
                  <Skeleton className="ml-auto h-3 w-14" />
                ) : (
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <span className="inline-flex cursor-default items-baseline justify-end gap-1.5" />
                      }
                    >
                      {aggregate?.label ? (
                        <span className="text-muted-foreground text-xs font-medium">
                          {aggregate.label}
                        </span>
                      ) : null}
                      <span
                        className={cn("font-table font-semibold", TOTAL_SIZE[aggregate?.size ?? "md"])}
                      >
                        {formatTotal(pick(value, kind), aggregate?.format)}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>
                      <dl className="grid grid-cols-[auto_auto] gap-x-3 gap-y-0.5 text-xs">
                        <dt>{t("Total")}</dt>
                        <dd className="text-right tabular-nums">
                          {formatTotal(value?.sum, aggregate?.format)}
                        </dd>
                        <dt>{t("Average")}</dt>
                        <dd className="text-right tabular-nums">
                          {formatTotal(value?.average, aggregate?.format)}
                        </dd>
                        <dt>{t("Lowest")}</dt>
                        <dd className="text-right tabular-nums">
                          {formatTotal(value?.min, aggregate?.format)}
                        </dd>
                        <dt>{t("Highest")}</dt>
                        <dd className="text-right tabular-nums">
                          {formatTotal(value?.max, aggregate?.format)}
                        </dd>
                      </dl>
                    </TooltipContent>
                  </Tooltip>
                )
              ) : null}
            </TableCell>
          );
        })}
      </TableRow>
    </TableFooter>
  );
}
