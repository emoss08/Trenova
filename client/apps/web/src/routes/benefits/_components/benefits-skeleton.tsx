import { KpiCard } from "@/components/kpi/kpi-card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Stands in twice: as the Suspense fallback while the console's chunk arrives,
 * and as the console's own loading state while the plans are read. One
 * component for both keeps the page from reflowing between the two.
 *
 * Every measurement mirrors the loaded layout: the KPI strip at the shared KPI
 * height, the plan-year toolbar, plans grouped by kind with a heading over each
 * card, and the three aside panels at the aside's fixed width. Group sizes and
 * widths are fixed so the shapes never reshuffle.
 */

const KPI_LABEL_WIDTHS = ["w-20", "w-20", "w-24", "w-22"] as const;
// Two kinds of plan, the way a small carrier's offer usually reads: a couple
// of medical plans and one dental.
const PLAN_GROUPS = [
  { label: "Medical", rows: ["w-32", "w-40"] },
  { label: "Dental", rows: ["w-28"] },
] as const;
const PLAN_BAR_WIDTHS = ["w-2/3", "w-1/3", "w-1/2"] as const;
const ASIDE_PANELS = ["Starting soon", "Ending soon", "Recently declined"] as const;
const ASIDE_ROW_COUNT = 2;

function OverviewSkeleton() {
  return (
    <div className="grid grid-cols-4 gap-3 lg:grid-cols-8">
      {KPI_LABEL_WIDTHS.map((width, index) => (
        <KpiCard key={index} span={2}>
          <div className="flex min-h-[14px] items-center gap-1.5">
            <Skeleton className="size-[11px] rounded-sm" />
            <Skeleton className={cn("h-2.5", width)} />
          </div>
          <Skeleton className={cn("h-6.5", index >= 2 ? "w-24" : "w-10")} />
          {index === 0 || index === 2 ? (
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
      <div className="flex flex-wrap items-center gap-3">
        <Skeleton className="h-7 w-36 rounded-md" />
        <Skeleton className="h-3 w-80 max-w-full" />
      </div>
      <Skeleton className="h-8 w-28 rounded-md" />
    </div>
  );
}

function PlanRowSkeleton({ titleWidth, bar }: { titleWidth: string; bar: string }) {
  return (
    <li className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2.5">
      <div className="flex min-w-0 flex-col gap-1">
        <span className="flex h-5 items-center gap-2">
          <Skeleton className={cn("h-3.5", titleWidth)} />
          <Skeleton className="h-3 w-10" />
        </span>
        <Skeleton className="h-3 w-64 max-w-full" />
        <div className="flex items-center gap-2">
          <span className="bg-muted flex h-1 w-32 overflow-hidden rounded-full">
            <Skeleton className={cn("h-full rounded-full", bar)} />
          </span>
          <Skeleton className="h-2.5 w-16" />
        </div>
      </div>
      <div className="flex items-center gap-3">
        <div className="hidden sm:grid sm:grid-cols-[auto_auto] sm:gap-x-2 sm:gap-y-1">
          <Skeleton className="h-3 w-14" />
          <Skeleton className="h-3 w-12" />
          <Skeleton className="h-3 w-14" />
          <Skeleton className="h-3 w-12" />
        </div>
        <div className="flex items-center gap-1">
          <Skeleton className="h-6 w-24 rounded-md" />
          <Skeleton className="size-6 rounded-md" />
        </div>
      </div>
    </li>
  );
}

function PlanGroupsSkeleton() {
  let position = 0;
  return (
    <div className="flex flex-col gap-4">
      {PLAN_GROUPS.map((group) => (
        <section key={group.label} aria-label={group.label} className="flex flex-col gap-1.5">
          <header className="flex items-center justify-between gap-2 px-1">
            <span className="flex items-center gap-1.5">
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-3 w-10" />
            </span>
            <Skeleton className="h-3 w-36" />
          </header>
          <ul className="bg-card divide-y overflow-hidden rounded-lg border">
            {group.rows.map((titleWidth) => (
              <PlanRowSkeleton
                key={titleWidth}
                titleWidth={titleWidth}
                bar={PLAN_BAR_WIDTHS[position++ % PLAN_BAR_WIDTHS.length] ?? "w-1/2"}
              />
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}

function AsidePanelSkeleton({ label }: { label: string }) {
  return (
    <section aria-label={label} className="bg-card flex flex-col overflow-hidden rounded-lg border">
      <header className="flex min-h-9 items-center gap-2 border-b px-3 py-1.5">
        <Skeleton className="size-3.5 rounded-sm" />
        <Skeleton className="h-3.5 w-24" />
      </header>
      <ul className="divide-y">
        {Array.from({ length: ASIDE_ROW_COUNT }, (_, index) => (
          <li key={index} className="flex flex-col gap-1 px-3 py-2">
            <span className="flex items-center justify-between gap-2">
              <Skeleton className="h-3 w-28" />
              <Skeleton className="h-2.5 w-16" />
            </span>
            <Skeleton className="h-2.5 w-20" />
          </li>
        ))}
      </ul>
    </section>
  );
}

/**
 * The root announces itself once; everything inside is shape, not content,
 * so it is hidden from assistive technology. The sections keep the loaded
 * panels' labels only so the two trees can be compared like for like.
 */
export function BenefitsSkeleton() {
  return (
    <div className="flex flex-col gap-4" aria-busy aria-label="Loading benefits">
      <div className="contents" aria-hidden>
        <OverviewSkeleton />
        <ToolbarSkeleton />
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_20rem]">
          <div className="flex min-w-0 flex-col gap-4">
            <PlanGroupsSkeleton />
          </div>
          <aside className="flex min-w-0 flex-col gap-4">
            {ASIDE_PANELS.map((label) => (
              <AsidePanelSkeleton key={label} label={label} />
            ))}
          </aside>
        </div>
      </div>
    </div>
  );
}
