import { useT } from "@trenova/shared/i18n/use-t";
import { Card, CardContent, CardHeader } from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";

/**
 * The three tabs of fuel management, each drawn in its own layout. Every one
 * stands in twice: as the Suspense fallback while the tab's chunk arrives, and
 * as the tab's own loading state while its list is read, so the tab does not
 * reflow between "loading the code" and "loading the data".
 *
 * Each text bar sits in a box the height of the line it stands for, and the
 * counts follow what the page draws once loaded: the DOE diesel series as
 * price cards, a page of program cards, and the eleven DOE rows of the index
 * table.
 */

const PRICE_CARD_COUNT = 8;
const PRICE_CARD_LABEL_WIDTHS = ["w-24", "w-20", "w-28", "w-24", "w-16", "w-28", "w-20", "w-24"];
const RANGE_COUNT = 3;
const CHART_Y_TICKS = 5;
const CHART_X_TICKS = 6;
const PROGRAM_CARD_COUNT = 6;
const PROGRAM_NAME_WIDTHS = ["w-36", "w-28", "w-40", "w-32", "w-28", "w-36"];
const INDEX_ROW_COUNT = 11;
const INDEX_HEAD_WIDTHS = ["w-8", "w-10", "w-10", "w-6", "w-10", "w-16", "w-8", "w-10"];
const INDEX_NAME_WIDTHS = ["w-40", "w-36", "w-44", "w-32", "w-40", "w-36"];

function Line({ height, width }: { height: string; width: string }) {
  return (
    <span className={cn("flex items-center", height)}>
      <Skeleton className={cn("h-3", width)} />
    </span>
  );
}

function SectionHeaderSkeleton({ buttonWidth }: { buttonWidth: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <div className="min-w-0 flex-1">
        <Line height="h-4.25" width="w-2xl max-w-full" />
      </div>
      <Skeleton className={cn("h-7 shrink-0 rounded-md", buttonWidth)} />
    </div>
  );
}

function PriceCardSkeleton({ labelWidth }: { labelWidth: string }) {
  return (
    <div className="bg-card rounded-lg border p-3">
      <div className="flex h-4.5 items-center justify-between gap-2">
        <Skeleton className={cn("h-2.5", labelWidth)} />
        <Skeleton className="h-4.5 w-10 rounded-md" />
      </div>
      <div className="mt-1.5 flex h-5.5 items-center gap-2">
        <Skeleton className="h-4.5 w-16" />
        <Skeleton className="h-2.5 w-10" />
      </div>
      <span className="mt-1 flex h-3.5 items-center">
        <Skeleton className="h-2 w-20" />
      </span>
    </div>
  );
}

/**
 * The plot area of the weekly price chart while its history is read: the
 * y-axis prices, the dashed grid and the week labels, at the chart's height.
 */
export function PriceChartBodySkeleton() {
  const t = useT();

  return (
    <div className="flex h-64 w-full gap-2 pt-2" aria-busy aria-label={t("Loading price history")}>
      <div aria-hidden className="flex w-11 shrink-0 flex-col items-end justify-between pb-7.5">
        {Array.from({ length: CHART_Y_TICKS }, (_, index) => (
          <Skeleton key={index} className="h-2 w-9" />
        ))}
      </div>
      <div aria-hidden className="flex min-w-0 flex-1 flex-col">
        <div className="flex flex-1 flex-col justify-between">
          {Array.from({ length: CHART_Y_TICKS }, (_, index) => (
            <span key={index} className="border-border border-t border-dashed" />
          ))}
        </div>
        <div className="flex h-7.5 items-center justify-between pt-2">
          {Array.from({ length: CHART_X_TICKS }, (_, index) => (
            <Skeleton key={index} className="h-2 w-10" />
          ))}
        </div>
      </div>
    </div>
  );
}

function PriceChartSkeleton() {
  return (
    <Card className="gap-0 p-0">
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <Line height="h-4" width="w-56" />
        <div className="flex gap-1">
          {Array.from({ length: RANGE_COUNT }, (_, index) => (
            <Skeleton key={index} className="h-7 w-9 rounded-md" />
          ))}
        </div>
      </CardHeader>
      <CardContent className="p-4">
        <PriceChartBodySkeleton />
      </CardContent>
    </Card>
  );
}

/** The price dashboard: the latest DOE week, a card per index, and the trend chart. */
export function FuelDashboardSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Loading fuel prices")}>
      <div className="contents" aria-hidden>
        <Line height="h-4.25" width="w-lg max-w-full" />
        <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4">
          {Array.from({ length: PRICE_CARD_COUNT }, (_, index) => (
            <PriceCardSkeleton key={index} labelWidth={PRICE_CARD_LABEL_WIDTHS[index]} />
          ))}
        </div>
        <PriceChartSkeleton />
      </div>
    </div>
  );
}

/** The program cards while the current rates are read, in the cards' own grid. */
export function ProgramCardsSkeleton() {
  const t = useT();

  return (
    <div
      className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3"
      aria-busy
      aria-label={t("Loading programs")}
    >
      {Array.from({ length: PROGRAM_CARD_COUNT }, (_, index) => (
        <div key={index} aria-hidden className="bg-card flex flex-col rounded-lg border p-4">
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <Line height="h-4.25" width={PROGRAM_NAME_WIDTHS[index]} />
              <span className="mt-0.5 flex h-4 items-center">
                <Skeleton className="h-2.5 w-28" />
              </span>
            </div>
            <span className="size-7 shrink-0" />
          </div>
          <div className="mt-4 flex h-5.5 items-center gap-1.5">
            <Skeleton className="h-4.5 w-20" />
            <Skeleton className="h-2.5 w-12" />
          </div>
          <div className="mt-3 flex flex-wrap items-center gap-1.5">
            <Skeleton className="h-5 w-20 rounded-full" />
            <Skeleton className="h-5 w-18 rounded-full" />
          </div>
        </div>
      ))}
    </div>
  );
}

/** The surcharge programs tab: its note and "New program", then the cards. */
export function ProgramSectionSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Loading surcharge programs")}>
      <div className="contents" aria-hidden>
        <SectionHeaderSkeleton buttonWidth="w-28" />
        <ProgramCardsSkeleton />
      </div>
    </div>
  );
}

/** The index table while the indices are read, row for row in the table's own cells. */
export function IndexTableSkeleton() {
  const t = useT();

  return (
    <div className="overflow-hidden rounded-lg border" aria-busy aria-label={t("Loading indices")}>
      <table aria-hidden className="w-full text-sm">
        <thead className="bg-muted/60">
          <tr className="text-left text-xs">
            {INDEX_HEAD_WIDTHS.map((width, index) => (
              <th key={index} className="px-4 py-2.5">
                <span className="flex h-4 items-center">
                  <Skeleton className={cn("h-2.5", width)} />
                </span>
              </th>
            ))}
            <th className="px-4 py-2.5" />
          </tr>
        </thead>
        <tbody>
          {Array.from({ length: INDEX_ROW_COUNT }, (_, index) => (
            <tr key={index} className="border-t">
              <td className="px-4 py-2.5">
                <Skeleton className="h-3 w-24" />
              </td>
              <td className="px-4 py-2.5">
                <Skeleton
                  className={cn("h-3", INDEX_NAME_WIDTHS[index % INDEX_NAME_WIDTHS.length])}
                />
              </td>
              <td className="px-4 py-2.5">
                <Skeleton className="h-3 w-12" />
              </td>
              <td className="px-4 py-2.5">
                <Skeleton className="h-3 w-12" />
              </td>
              <td className="px-4 py-2.5">
                <Skeleton className="h-5 w-16 rounded-full" />
              </td>
              <td className="px-4 py-2.5">
                <Skeleton className="h-3 w-12" />
              </td>
              <td className="px-4 py-2.5">
                <Skeleton className="h-3 w-16" />
              </td>
              <td className="px-4 py-2.5">
                <Skeleton className="h-5 w-14 rounded-full" />
              </td>
              <td className="px-4 py-2.5">
                <div className="flex justify-end gap-1">
                  <span className="flex size-7 items-center justify-center">
                    <Skeleton className="size-3.5 rounded-sm" />
                  </span>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

/** The fuel indices tab: its note and "New custom index", then the table. */
export function IndexSectionSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Loading fuel indices")}>
      <div className="contents" aria-hidden>
        <SectionHeaderSkeleton buttonWidth="w-34" />
        <IndexTableSkeleton />
      </div>
    </div>
  );
}
