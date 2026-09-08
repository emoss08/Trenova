import { KpiCard } from "@/components/kpi/kpi-card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { CSA_BASIC_ORDER, SAFETY_EVENT_KIND_LABELS } from "@trenova/shared/lib/csa";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Stands in twice: as the Suspense fallback while the console's chunk arrives,
 * and as the console's own loading state while the roll-up query runs. One
 * component for both keeps the page from reflowing between the two.
 *
 * Every measurement mirrors the loaded layout: four KPI cards at the shared
 * KPI height, seven BASIC rows because there are always seven, a chart the
 * height of the real one, and the three panels below with the row shapes they
 * draw once data lands. Widths are fixed so the shapes never reshuffle.
 */

const KPI_COUNT = 4;
const KPI_LABEL_WIDTHS = ["w-14", "w-20", "w-24", "w-20"] as const;
const BASIC_LABEL_WIDTHS = ["w-28", "w-30", "w-26", "w-36", "w-34", "w-32", "w-28"] as const;
const BASIC_BAR_WIDTHS = ["w-full", "w-3/4", "w-1/2", "w-1/3", "w-1/4", "w-1/5", "w-1/6"] as const;
const TREND_BAR_HEIGHTS = [
  "h-1/3",
  "h-1/2",
  "h-2/5",
  "h-3/5",
  "h-1/2",
  "h-4/5",
  "h-full",
  "h-3/5",
  "h-2/5",
  "h-1/2",
  "h-1/3",
  "h-2/5",
] as const;
const TREND_KINDS = Object.keys(SAFETY_EVENT_KIND_LABELS);
const TERMINAL_ROW_COUNT = 4;
const TERMINAL_LABEL_WIDTHS = ["w-16", "w-12", "w-14", "w-20"] as const;
const TERMINAL_BAR_WIDTHS = ["w-1/2", "w-1/3", "w-1/4", "w-1/6"] as const;
const RANK_ROW_COUNT = 5;
const RANK_NAME_WIDTHS = ["w-32", "w-28", "w-36", "w-24", "w-30"] as const;

function CardHeaderSkeleton({ titleWidth, right }: { titleWidth: string; right?: string }) {
  return (
    <header className="flex min-h-9 items-center justify-between gap-2 border-b px-3 py-2">
      <div className="flex items-center gap-2">
        <Skeleton className="size-3.5 rounded-sm" />
        <Skeleton className={cn("h-3.5", titleWidth)} />
      </div>
      {right ? <Skeleton className={cn("h-3", right)} /> : null}
    </header>
  );
}

function OverviewSkeleton() {
  return (
    <div className="grid grid-cols-4 gap-3 lg:grid-cols-8">
      {Array.from({ length: KPI_COUNT }, (_, index) => (
        <KpiCard key={index} span={2}>
          <div className="flex min-h-[14px] items-center gap-1.5">
            <Skeleton className="size-[11px] rounded-sm" />
            <Skeleton className={cn("h-2.5", KPI_LABEL_WIDTHS[index])} />
          </div>
          {index === 1 ? (
            <div className="flex items-center gap-2.5">
              <Skeleton className="size-10 rounded-full" />
              <Skeleton className="h-6.5 w-14" />
            </div>
          ) : (
            <Skeleton className="h-6.5 w-10" />
          )}
          {index === 0 ? (
            <Skeleton className="mt-auto h-1.5 w-full rounded-full" />
          ) : (
            <Skeleton className="mt-auto h-2.5 w-3/4" />
          )}
        </KpiCard>
      ))}
    </div>
  );
}

function ToolbarSkeleton() {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <Skeleton className="h-7 w-72 rounded-md" />
        <Skeleton className="h-3 w-56" />
      </div>
      <Skeleton className="h-3 w-24" />
    </div>
  );
}

function BasicsSkeleton() {
  return (
    <section aria-label="CSA BASICs" className="bg-card overflow-hidden rounded-lg border">
      <CardHeaderSkeleton titleWidth="w-20" right="w-32" />
      <ul className="divide-y">
        {CSA_BASIC_ORDER.map((basic, index) => (
          <li
            key={basic}
            className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2"
          >
            <div className="flex min-w-0 flex-col gap-1.5">
              <span className="flex h-5 items-center gap-2">
                <Skeleton className={cn("h-3.5", BASIC_LABEL_WIDTHS[index])} />
                {index === 0 ? <Skeleton className="h-4 w-14 rounded-full" /> : null}
              </span>
              <span className="bg-muted flex h-1.5 w-full max-w-md overflow-hidden rounded-full">
                <Skeleton className={cn("h-full rounded-full", BASIC_BAR_WIDTHS[index])} />
              </span>
            </div>
            <Skeleton className="h-3 w-16" />
          </li>
        ))}
      </ul>
    </section>
  );
}

function TrendSkeleton() {
  return (
    <section aria-label="Events by month" className="bg-card overflow-hidden rounded-lg border">
      <CardHeaderSkeleton titleWidth="w-28" right="w-36" />
      <div className="p-3">
        <div className="flex h-44 w-full gap-2 pl-7">
          <div className="flex flex-1 items-end gap-[6%] border-b pb-px">
            {TREND_BAR_HEIGHTS.map((height, index) => (
              <Skeleton key={index} className={cn("w-full rounded-t", height)} />
            ))}
          </div>
        </div>
      </div>
      <ul className="divide-y border-t" aria-label="Events by kind">
        {TREND_KINDS.map((kind) => (
          <li key={kind} className="flex h-7 items-center justify-between gap-2 px-3">
            <Skeleton className="h-3 w-20" />
            <Skeleton className="h-3 w-28" />
          </li>
        ))}
        <li className="flex h-7 items-center justify-between gap-2 px-3">
          <Skeleton className="h-3 w-14" />
          <Skeleton className="h-3 w-20" />
        </li>
      </ul>
    </section>
  );
}

function TerminalsSkeleton() {
  return (
    <section
      aria-label="By terminal"
      className="bg-card flex flex-col overflow-hidden rounded-lg border"
    >
      <CardHeaderSkeleton titleWidth="w-20" right="w-16" />
      <ul className="divide-y">
        {Array.from({ length: TERMINAL_ROW_COUNT }, (_, index) => (
          <li key={index} className="flex flex-col gap-1.5 px-3 py-2">
            <span className="flex h-4 items-center justify-between gap-2">
              <span className="flex items-center gap-2">
                <Skeleton className="size-2 rounded-full" />
                <Skeleton className={cn("h-3", TERMINAL_LABEL_WIDTHS[index])} />
              </span>
              <Skeleton className="h-3 w-16" />
            </span>
            <span className="bg-muted flex h-1 w-full overflow-hidden rounded-full">
              <Skeleton className={cn("h-full rounded-full", TERMINAL_BAR_WIDTHS[index])} />
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

function RankListSkeleton({ label }: { label: string }) {
  return (
    <section aria-label={label} className="bg-card flex flex-col overflow-hidden rounded-lg border">
      <CardHeaderSkeleton titleWidth="w-24" right="w-14" />
      <ul className="divide-y">
        {Array.from({ length: RANK_ROW_COUNT }, (_, index) => (
          <li
            key={index}
            className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2"
          >
            <span className="flex min-w-0 items-center gap-2.5">
              <Skeleton className="size-6 shrink-0 rounded-full" />
              <span className="flex flex-col gap-1">
                <Skeleton className={cn("h-3.5", RANK_NAME_WIDTHS[index])} />
                <Skeleton className="h-2.5 w-24" />
              </span>
            </span>
            <span className="flex items-center gap-2">
              <Skeleton className="h-3 w-8" />
              <Skeleton className="h-4 w-14 rounded-full" />
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

/**
 * The root announces itself once; everything inside is shape, not content,
 * so it is hidden from assistive technology. The sections keep the loaded
 * cards' labels only so the two trees can be compared like for like.
 */
export function FleetSafetySkeleton() {
  return (
    <div className="flex flex-col gap-4" aria-busy aria-label="Loading fleet safety">
      <div className="contents" aria-hidden>
        <OverviewSkeleton />
        <ToolbarSkeleton />
        <div className="grid gap-4 lg:grid-cols-2">
          <BasicsSkeleton />
          <TrendSkeleton />
        </div>
        <div className="grid gap-4 lg:grid-cols-3">
          <TerminalsSkeleton />
          <RankListSkeleton label="Needs attention" />
          <RankListSkeleton label="Best records" />
        </div>
      </div>
    </div>
  );
}
