import { useT } from "@trenova/shared/i18n/use-t";
import { KpiCard } from "@/components/kpi/kpi-card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Stands in twice: as the Suspense fallback while the console's chunk arrives,
 * and as the console's own loading state while the headcount and positions
 * are read. One component for both keeps the page from reflowing between the
 * two.
 *
 * Every measurement mirrors the loaded layout: the KPI strip at the shared KPI
 * height, the chart toolbar, an indented tree of positions, the approval
 * cover panel, and the three aside panels at the aside's fixed width. Depths
 * and widths are fixed so the shapes never reshuffle.
 */

const KPI_LABEL_WIDTHS = ["w-12", "w-16", "w-16", "w-20"] as const;
const INDENT_REM = 1.25;
// Depth and title width per tree row: a top position, its reports, and the
// reports under those, the way a small carrier's chart reads.
const TREE_ROWS = [
  { depth: 0, title: "w-36" },
  { depth: 1, title: "w-32" },
  { depth: 2, title: "w-40" },
  { depth: 2, title: "w-28" },
  { depth: 1, title: "w-30" },
  { depth: 2, title: "w-36" },
  { depth: 1, title: "w-24" },
  { depth: 0, title: "w-32" },
] as const;
const COVER_ROW_COUNT = 2;
const BREAKDOWN_LABEL_WIDTHS = ["w-16", "w-12", "w-20"] as const;
const BREAKDOWN_BAR_WIDTHS = ["w-2/3", "w-1/3", "w-1/6"] as const;
const VACANT_ROW_COUNT = 2;

function PanelHeaderSkeleton({ titleWidth, right }: { titleWidth: string; right?: string }) {
  return (
    <header className="flex min-h-9 items-center justify-between gap-2 border-b px-3 py-1.5">
      <div className="flex items-center gap-2">
        <Skeleton className="size-3.5 rounded-sm" />
        <Skeleton className={cn("h-3.5", titleWidth)} />
      </div>
      {right ? <Skeleton className={cn("h-3", right)} /> : null}
    </header>
  );
}

function OverviewSkeleton({ showCover }: { showCover: boolean }) {
  const count = showCover ? 4 : 3;
  return (
    <div
      className={
        showCover
          ? "grid grid-cols-4 gap-3 lg:grid-cols-8"
          : "grid grid-cols-4 gap-3 lg:grid-cols-6"
      }
    >
      {Array.from({ length: count }, (_, index) => (
        <KpiCard key={index} span={2}>
          <div className="flex min-h-[14px] items-center gap-1.5">
            <Skeleton className="size-[11px] rounded-sm" />
            <Skeleton className={cn("h-2.5", KPI_LABEL_WIDTHS[index])} />
          </div>
          <Skeleton className="h-6.5 w-10" />
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

function TreeSkeleton() {
  const t = useT();

  return (
    <section aria-label={t("Org chart")} className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Skeleton className="h-7 w-72 max-w-full rounded-md" />
          <Skeleton className="h-6 w-18 rounded-md" />
          <Skeleton className="h-6 w-18 rounded-md" />
        </div>
        <Skeleton className="h-8 w-32 rounded-md" />
      </div>
      <div className="bg-card overflow-hidden rounded-lg border">
        <ul aria-label={t("Positions")}>
          {TREE_ROWS.map((row, index) => (
            <li
              key={index}
              className="relative grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 border-b py-1.5 pr-2 last:border-b-0"
              style={{ paddingLeft: `${0.5 + row.depth * INDENT_REM}rem` }}
            >
              {row.depth > 0 ? (
                <span
                  className="bg-border absolute top-0 bottom-0 w-px"
                  style={{ left: `${0.5 + (row.depth - 1) * INDENT_REM + 0.6}rem` }}
                />
              ) : null}
              <div className="flex min-w-0 items-center gap-1.5">
                <Skeleton className="size-5 shrink-0 rounded-md" />
                <div className="flex min-w-0 flex-col gap-1">
                  <span className="flex h-5 items-center gap-2">
                    <Skeleton className={cn("h-3.5", row.title)} />
                    <Skeleton className="h-3 w-8" />
                  </span>
                  <Skeleton className="h-3 w-24" />
                </div>
              </div>
              <Skeleton className="h-4 w-12" />
            </li>
          ))}
        </ul>
      </div>
      <Skeleton className="h-3 w-3/4 max-w-lg" />
    </section>
  );
}

function CoverSkeleton() {
  const t = useT();

  return (
    <section aria-label={t("Approval cover")} className="bg-card overflow-hidden rounded-lg border">
      <header className="flex flex-wrap items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex items-center gap-2">
          <Skeleton className="size-3.5 rounded-sm" />
          <Skeleton className="h-3.5 w-24" />
          <Skeleton className="hidden h-3 w-64 md:block" />
        </div>
        <div className="flex items-center gap-2">
          <Skeleton className="h-7 w-44 rounded-md" />
          <Skeleton className="h-8 w-28 rounded-md" />
        </div>
      </header>
      <ul className="divide-y" aria-label={t("Cover")}>
        {Array.from({ length: COVER_ROW_COUNT }, (_, index) => (
          <li
            key={index}
            className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2"
          >
            <div className="flex flex-col gap-1">
              <span className="flex items-center gap-1.5">
                <Skeleton className="h-3.5 w-24" />
                <Skeleton className="size-3.5 rounded-sm" />
                <Skeleton className="h-3.5 w-28" />
              </span>
              <Skeleton className="h-3 w-40" />
            </div>
            <Skeleton className="h-6 w-20 rounded-md" />
          </li>
        ))}
      </ul>
    </section>
  );
}

function BreakdownSkeleton({ label, right }: { label: string; right?: string }) {
  return (
    <section aria-label={label} className="bg-card flex flex-col overflow-hidden rounded-lg border">
      <PanelHeaderSkeleton titleWidth="w-24" right={right} />
      <ul className="divide-y">
        {BREAKDOWN_LABEL_WIDTHS.map((width, index) => (
          <li key={index} className="flex flex-col gap-1.5 px-3 py-2">
            <span className="flex h-4 items-center justify-between gap-2">
              <span className="flex items-center gap-2">
                <Skeleton className="size-2 rounded-full" />
                <Skeleton className={cn("h-3", width)} />
              </span>
              <Skeleton className="h-3 w-6" />
            </span>
            <span className="bg-muted flex h-1 w-full overflow-hidden rounded-full">
              <Skeleton className={cn("h-full rounded-full", BREAKDOWN_BAR_WIDTHS[index])} />
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

function VacantSkeleton() {
  const t = useT();

  return (
    <section
      aria-label={t("Titles nobody holds yet")}
      className="bg-card flex flex-col overflow-hidden rounded-lg border"
    >
      <PanelHeaderSkeleton titleWidth="w-36" />
      <ul className="divide-y">
        {Array.from({ length: VACANT_ROW_COUNT }, (_, index) => (
          <li key={index} className="flex items-center justify-between gap-2 px-3 py-2">
            <span className="flex flex-col gap-1">
              <Skeleton className="h-3 w-32" />
              <Skeleton className="h-2.5 w-24" />
            </span>
            <Skeleton className="h-6 w-10 rounded-md" />
          </li>
        ))}
      </ul>
    </section>
  );
}

type OrgStructureSkeletonProps = {
  /** Whether the strip carries the cover card; false for somebody who may not read delegations. */
  showCover?: boolean;
};

/**
 * The root announces itself once; everything inside is shape, not content,
 * so it is hidden from assistive technology. The sections keep the loaded
 * panels' labels only so the two trees can be compared like for like.
 */
export function OrgStructureSkeleton({ showCover = true }: OrgStructureSkeletonProps) {
  const t = useT();

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Loading the organisation")}>
      <div className="contents" aria-hidden>
        <OverviewSkeleton showCover={showCover} />
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_20rem]">
          <div className="flex min-w-0 flex-col gap-4">
            <TreeSkeleton />
            {showCover ? <CoverSkeleton /> : null}
          </div>
          <aside className="flex min-w-0 flex-col gap-4">
            <BreakdownSkeleton label={t("By terminal")} right="w-14" />
            <BreakdownSkeleton label={t("By department")} />
            <VacantSkeleton />
          </aside>
        </div>
      </div>
    </div>
  );
}
