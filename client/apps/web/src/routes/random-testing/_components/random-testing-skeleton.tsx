import { useT } from "@trenova/shared/i18n/use-t";
import { KpiCard } from "@/components/kpi/kpi-card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

/**
 * Stands in twice: as the Suspense fallback while the console's chunk arrives,
 * and as the console's own loading state while the pools and rounds are read.
 * One component for both keeps the page from reflowing between the two.
 *
 * Every measurement mirrors the loaded layout: four KPI cards at the shared
 * KPI height, the pools panel with a row per pool carrying its year of round
 * slots, and the rounds panel with its status control and table. Widths are
 * fixed so the shapes never reshuffle.
 */

const KPI_LABEL_WIDTHS = ["w-16", "w-24", "w-28", "w-16"] as const;
// Two pools, the way a small carrier's programme usually reads: the DOT
// pool on a quarterly calendar and a second pool drawn monthly.
const POOLS = [
  { name: "w-28", slots: 4 },
  { name: "w-36", slots: 12 },
] as const;
const ROUND_ROW_COUNT = 5;
const ROUND_POOL_WIDTHS = ["w-24", "w-32", "w-24", "w-28", "w-32"] as const;
const ROUND_COLUMN_WIDTHS = ["w-12", "w-8", "w-12", "w-8", "w-12", "w-14", "w-12"] as const;

function OverviewSkeleton() {
  return (
    <div className="grid grid-cols-4 gap-3 lg:grid-cols-8">
      {KPI_LABEL_WIDTHS.map((width, index) => (
        <KpiCard key={index} span={2}>
          <div className="flex min-h-[14px] items-center gap-1.5">
            <Skeleton className="size-[11px] rounded-sm" />
            <Skeleton className={cn("h-2.5", width)} />
          </div>
          <Skeleton className="h-6.5 w-10" />
          {index === 2 ? (
            <div className="mt-auto flex flex-col gap-1">
              <Skeleton className="h-2.5 w-full" />
              <Skeleton className="h-2.5 w-full" />
            </div>
          ) : (
            <Skeleton className="mt-auto h-2.5 w-3/4" />
          )}
        </KpiCard>
      ))}
    </div>
  );
}

function PanelHeaderSkeleton({
  titleWidth,
  hint,
  action,
}: {
  titleWidth: string;
  hint?: string;
  action?: ReactNode;
}) {
  return (
    <header className="flex min-h-9 items-center justify-between gap-2 border-b px-3 py-1.5">
      <div className="flex items-center gap-2">
        <Skeleton className="size-3.5 rounded-sm" />
        <Skeleton className={cn("h-3.5", titleWidth)} />
        <Skeleton className="size-3.5 rounded-full" />
        <Skeleton className="h-3 w-3" />
      </div>
      <div className="flex items-center gap-2">
        {hint ? <Skeleton className={cn("h-3", hint)} /> : null}
        {action}
      </div>
    </header>
  );
}

function PoolsSkeleton() {
  const t = useT();

  return (
    <section aria-label={t("Pools")} className="bg-card flex flex-col overflow-hidden rounded-lg border">
      <PanelHeaderSkeleton
        titleWidth="w-10"
        hint="w-20"
        action={<Skeleton className="h-6 w-20 rounded-md" />}
      />
      <ul aria-label={t("Pools")} className="divide-y">
        {POOLS.map((pool, index) => (
          <li
            key={index}
            className="grid gap-x-6 gap-y-2 px-3 py-2.5 lg:grid-cols-[minmax(0,1.2fr)_auto_minmax(0,1fr)_auto] lg:items-center"
          >
            <div className="flex min-w-0 flex-col gap-1.5">
              <span className="flex h-5 items-center gap-1.5">
                <Skeleton className={cn("h-3.5", pool.name)} />
                <Skeleton className="h-4 w-10 rounded-full" />
                {index === 0 ? <Skeleton className="h-4 w-14 rounded-full" /> : null}
              </span>
              <Skeleton className="h-3 w-56 max-w-full" />
            </div>
            <span className="flex flex-wrap gap-1">
              {Array.from({ length: pool.slots }, (_, slot) => (
                <Skeleton key={slot} className="h-7 w-9 rounded-md" />
              ))}
            </span>
            <Skeleton className="h-3 w-44 max-w-full" />
            <span className="flex items-center justify-end gap-1.5">
              <Skeleton className="size-6 rounded-md" />
              <Skeleton className="size-6 rounded-md" />
              <Skeleton className="h-6 w-20 rounded-md" />
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

function RoundsSkeleton() {
  const t = useT();

  return (
    <section
      aria-label={t("Rounds")}
      className="bg-card flex flex-col overflow-hidden rounded-lg border"
    >
      <PanelHeaderSkeleton
        titleWidth="w-14"
        hint="w-16"
        action={<Skeleton className="h-7 w-52 rounded-md" />}
      />
      <table aria-label={t("Rounds")} className="w-full text-xs">
        <thead>
          <tr className="bg-sidebar h-8 border-b">
            {ROUND_COLUMN_WIDTHS.map((width, index) => (
              <th key={index} className={cn(index === 0 ? "px-3" : "px-2")}>
                <Skeleton className={cn("h-3", width, index === 5 && "ml-auto")} />
              </th>
            ))}
            <th className="px-3" />
          </tr>
        </thead>
        <tbody>
          {Array.from({ length: ROUND_ROW_COUNT }, (_, index) => (
            <tr key={index} className="border-b last:border-b-0">
              <td className="px-3 py-2">
                <span className="flex flex-col gap-1">
                  <Skeleton className="h-3.5 w-16" />
                  <Skeleton className="h-2.5 w-20" />
                </span>
              </td>
              <td className="px-2 py-2">
                <span className="flex items-center gap-1.5">
                  <Skeleton className="h-4 w-10 rounded-full" />
                  <Skeleton className={cn("h-3", ROUND_POOL_WIDTHS[index])} />
                </span>
              </td>
              <td className="px-2 py-2">
                <Skeleton className="h-4 w-14 rounded-full" />
              </td>
              <td className="px-2 py-2">
                <span className="flex items-center gap-2">
                  <Skeleton className="h-3 w-8" />
                  <Skeleton className="h-1 w-12 rounded-full" />
                </span>
              </td>
              <td className="px-2 py-2">
                <span className="flex items-center gap-2">
                  <Skeleton className="h-3 w-8" />
                  <Skeleton className="h-1 w-12 rounded-full" />
                </span>
              </td>
              <td className="px-2 py-2">
                <Skeleton className="ml-auto h-3 w-8" />
              </td>
              <td className="px-2 py-2">
                <Skeleton className="h-3 w-20" />
              </td>
              <td className="px-3 py-2">
                <span className="flex items-center justify-end gap-1">
                  <Skeleton className="size-6 rounded-md" />
                  <Skeleton className="size-6 rounded-md" />
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

/**
 * The root announces itself once; everything inside is shape, not content,
 * so it is hidden from assistive technology. The sections keep the loaded
 * panels' labels only so the two trees can be compared like for like.
 */
export function RandomTestingSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Loading random testing")}>
      <div className="contents" aria-hidden>
        <OverviewSkeleton />
        <PoolsSkeleton />
        <RoundsSkeleton />
      </div>
    </div>
  );
}
