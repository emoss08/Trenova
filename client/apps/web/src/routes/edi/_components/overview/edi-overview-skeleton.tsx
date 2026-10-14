import { useT } from "@trenova/shared/i18n/use-t";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Stands in twice: as the Suspense fallback while the EDI pages' chunk
 * arrives on the overview, and as the overview's own loading state while the
 * window's summary is read. One component for both keeps the page from
 * reflowing between the two.
 *
 * Every measurement mirrors the loaded overview: the window note and range
 * control, the two strips of four figures, the two trend charts with their
 * axes and legends, the partner scorecard table and the recent failures list.
 * Each text bar sits in a box the height of the line it stands for.
 */

const RANGE_COUNT = 5;
const STRIP_COUNT = 2;
const STRIP_ITEM_COUNT = 4;
const STRIP_LABEL_WIDTHS = ["w-32", "w-24", "w-24", "w-36"] as const;
const STRIP_SUB_WIDTHS = ["w-40", "w-36", "w-44", "w-40"] as const;
const SECTION_TITLE_WIDTHS = {
  attention: "w-24",
  pipeline: "w-22",
  trends: "w-12",
  scorecards: "w-28",
  failures: "w-24",
} as const;
const CHART_COUNT = 2;
const CHART_Y_TICKS = 5;
const CHART_X_TICKS = 6;
const CHART_LEGEND_WIDTHS = [["w-10", "w-12", "w-16"], ["w-20"]] as const;
const SCORECARD_METRIC_COUNT = 11;
const SCORECARD_ROW_COUNT = 3;
const SCORECARD_NAME_WIDTHS = ["w-32", "w-24", "w-28"] as const;
const FAILURE_ROW_COUNT = 3;
const FAILURE_ERROR_WIDTHS = ["w-3/4", "w-2/3", "w-1/2"] as const;

function SectionTitle({ width }: { width: string }) {
  return (
    <span className="flex h-4.25 items-center">
      <Skeleton className={cn("h-3", width)} />
    </span>
  );
}

function FigureStripSkeleton() {
  return (
    <KpiStrip>
      {Array.from({ length: STRIP_ITEM_COUNT }, (_, index) => (
        <KpiStripItem
          key={index}
          label={
            <span className="flex h-4 items-center">
              <Skeleton className={cn("h-2.5", STRIP_LABEL_WIDTHS[index])} />
            </span>
          }
          value={<Skeleton className="h-4.25 w-6" />}
          sub={
            <span className="flex h-4 items-center">
              <Skeleton className={cn("h-2.5", STRIP_SUB_WIDTHS[index])} />
            </span>
          }
        />
      ))}
    </KpiStrip>
  );
}

function TrendChartSkeleton({ legendWidths }: { legendWidths: readonly string[] }) {
  return (
    <div className="bg-background rounded-md border p-3">
      <span className="flex h-4 items-center">
        <Skeleton className="h-2.5 w-32" />
      </span>
      <div className="mt-2 flex h-[220px] w-full flex-col">
        <div className="flex min-h-0 flex-1 gap-2 pt-2">
          <div className="flex w-7 shrink-0 flex-col items-end justify-between pb-6">
            {Array.from({ length: CHART_Y_TICKS }, (_, index) => (
              <Skeleton key={index} className="h-2 w-4" />
            ))}
          </div>
          <div className="flex min-w-0 flex-1 flex-col">
            <div className="flex flex-1 flex-col justify-between">
              {Array.from({ length: CHART_Y_TICKS }, (_, index) => (
                <span key={index} className="border-border border-t border-dashed" />
              ))}
            </div>
            <div className="flex h-6 items-center justify-between pt-2">
              {Array.from({ length: CHART_X_TICKS }, (_, index) => (
                <Skeleton key={index} className="h-2 w-8" />
              ))}
            </div>
          </div>
        </div>
        <div className="flex h-7 items-center justify-center gap-4 pt-3">
          {legendWidths.map((width, index) => (
            <span key={index} className="flex items-center gap-1.5">
              <Skeleton className="size-2 rounded-sm" />
              <Skeleton className={cn("h-2", width)} />
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}

function ScorecardsSkeleton() {
  return (
    <div className="bg-background overflow-x-auto rounded-md border">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead>
              <Skeleton className="h-2.5 w-12" />
            </TableHead>
            {Array.from({ length: SCORECARD_METRIC_COUNT }, (_, index) => (
              <TableHead key={index}>
                <Skeleton className="ml-auto h-2.5 w-10" />
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {SCORECARD_NAME_WIDTHS.slice(0, SCORECARD_ROW_COUNT).map((width, rowIndex) => (
            <TableRow key={rowIndex} className="hover:bg-transparent">
              <TableCell>
                <span className="flex flex-col">
                  <span className="flex h-4.25 items-center">
                    <Skeleton className={cn("h-3", width)} />
                  </span>
                  <span className="flex h-4 items-center">
                    <Skeleton className="h-2.5 w-12" />
                  </span>
                </span>
              </TableCell>
              {Array.from({ length: SCORECARD_METRIC_COUNT }, (_, index) => (
                <TableCell key={index}>
                  <Skeleton className="ml-auto h-3 w-6" />
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function RecentFailuresSkeleton() {
  return (
    <div className="bg-background flex flex-col divide-y rounded-md border">
      {FAILURE_ERROR_WIDTHS.slice(0, FAILURE_ROW_COUNT).map((width, index) => (
        <div key={index} className="flex items-start gap-3 p-3">
          <Skeleton className="mt-0.5 size-4 shrink-0 rounded-sm" />
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <Skeleton className="h-5 w-32 rounded-full" />
              <Skeleton className="h-3 w-24" />
              <Skeleton className="h-2.5 w-28" />
            </div>
            <span className="mt-1 flex h-4 items-center">
              <Skeleton className={cn("h-2.5", width)} />
            </span>
          </div>
          <span className="flex h-4 shrink-0 items-center">
            <Skeleton className="h-2.5 w-16" />
          </span>
        </div>
      ))}
    </div>
  );
}

/**
 * The root announces itself once; everything inside is shape, not content,
 * so it is hidden from assistive technology.
 */
export function EDIOverviewSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-6 p-3" aria-busy aria-label={t("Loading EDI operations")}>
      <div className="contents" aria-hidden>
        <div className="flex items-center justify-between gap-3">
          <span className="flex h-4 items-center">
            <Skeleton className="h-2.5 w-80" />
          </span>
          <div className="bg-background flex items-center gap-1 rounded-md border p-0.5">
            {Array.from({ length: RANGE_COUNT }, (_, index) => (
              <Skeleton key={index} className="h-6 w-8 rounded-md" />
            ))}
          </div>
        </div>
        {Array.from({ length: STRIP_COUNT }, (_, index) => (
          <section key={index} className="flex flex-col gap-3">
            <SectionTitle
              width={index === 0 ? SECTION_TITLE_WIDTHS.attention : SECTION_TITLE_WIDTHS.pipeline}
            />
            <FigureStripSkeleton />
          </section>
        ))}
        <section className="flex flex-col gap-3">
          <SectionTitle width={SECTION_TITLE_WIDTHS.trends} />
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            {CHART_LEGEND_WIDTHS.slice(0, CHART_COUNT).map((legend, index) => (
              <TrendChartSkeleton key={index} legendWidths={legend} />
            ))}
          </div>
        </section>
        <section className="flex flex-col gap-3">
          <SectionTitle width={SECTION_TITLE_WIDTHS.scorecards} />
          <ScorecardsSkeleton />
        </section>
        <section className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-2">
            <SectionTitle width={SECTION_TITLE_WIDTHS.failures} />
            <div className="flex items-center gap-3">
              <Skeleton className="h-2.5 w-24" />
              <Skeleton className="h-2.5 w-28" />
            </div>
          </div>
          <RecentFailuresSkeleton />
        </section>
      </div>
    </div>
  );
}
