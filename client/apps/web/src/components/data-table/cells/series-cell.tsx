import { Sparkline } from "@/components/kpi/sparkline";
import {
  seriesLoader,
  seriesSpecKey,
  type DataTableSeriesSpec,
} from "@/lib/data-table-series";
import { useQuery } from "@tanstack/react-query";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";

const SERIES_STALE_MS = 5 * 60_000;
const COUNT_FORMAT = new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 });
const WEEK_LABEL = { month: "short", day: "numeric" } as const;
const MONTH_LABEL = { month: "short", year: "numeric" } as const;

type SeriesCellProps = {
  spec: DataTableSeriesSpec;
  /** The row the chart belongs to, as the counted table names it. */
  id: string | null | undefined;
  format?: "count" | "money";
};

/**
 * A small chart of one row's activity over recent weeks or months, with its total.
 * Every row on the page is fetched in one request, and only while the column shows.
 */
export function SeriesCell({ spec, id, format = "count" }: SeriesCellProps) {
  const t = useT();
  const { data, isPending, isError } = useQuery({
    queryKey: ["dataTableSeries", seriesSpecKey(spec), id],
    queryFn: () => seriesLoader(spec).load(id as string),
    enabled: !!id,
    staleTime: SERIES_STALE_MS,
  });

  if (!id || isError) return <span className="text-muted-foreground font-mono text-sm">—</span>;
  if (isPending) return <Skeleton className="h-4 w-24" />;

  const points = data?.points ?? [];
  const periodStarts = data?.periodStarts ?? [];
  const show = (value: number) =>
    format === "money" ? formatCurrency(value) : COUNT_FORMAT.format(value);
  const total = points.reduce((sum, value) => sum + value, 0);
  const latest = points.at(-1) ?? 0;

  return (
    <Tooltip>
      <TooltipTrigger
        render={<span className="text-muted-foreground flex items-center gap-2" />}
        aria-label={t("{0} in the period, {1} in the latest", show(total), show(latest))}
      >
        <Sparkline data={points.length > 0 ? points : [0]} color="currentColor" width={72} height={20} />
        <span className="text-foreground font-mono text-sm tabular-nums">{show(total)}</span>
      </TooltipTrigger>
      <TooltipContent>
        <dl className="grid grid-cols-[auto_auto] gap-x-3 gap-y-0.5 text-xs">
          {periodStarts.map((start, index) => (
            <div key={start} className="contents">
              <dt>
                {formatUnixInUserTimezone(
                  start,
                  spec.interval === "month" ? MONTH_LABEL : WEEK_LABEL,
                )}
              </dt>
              <dd className="text-right tabular-nums">{show(points[index] ?? 0)}</dd>
            </div>
          ))}
        </dl>
      </TooltipContent>
    </Tooltip>
  );
}
