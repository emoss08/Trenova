import { useT } from "@trenova/shared/i18n/use-t";
import { KpiCard } from "@/components/kpi/kpi-card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Stands in twice: as the Suspense fallback while the console's chunk arrives,
 * and as the console's own loading state while the team is read. One
 * component for both keeps the page from reflowing between the two.
 *
 * Every measurement mirrors the loaded layout: four KPI cards at the shared
 * KPI height, the attention panel, the roster toolbar and its grouped member
 * lists, and the aside at its fixed width with the terminal, coming-up and
 * cover panels. Widths are fixed so the shapes never reshuffle.
 */

const KPI_LABEL_WIDTHS = ["w-20", "w-24", "w-24", "w-24"] as const;
const ATTENTION_ROW_COUNT = 2;
// Two groups, the way a manager's team reads: their own reports first, then
// the people reached through a terminal they run.
const ROSTER_GROUPS = [
  { label: "Direct reports", names: ["w-28", "w-24", "w-32", "w-20"] },
  { label: "Through a terminal", names: ["w-24", "w-32", "w-28"] },
] as const;
const TERMINAL_BAR_WIDTHS = ["w-2/3", "w-1/3", "w-1/6"] as const;
const COMING_UP_ROW_COUNT = 3;
const COVER_ROW_COUNT = 2;

function PanelHeaderSkeleton({ titleWidth, right }: { titleWidth: string; right?: string }) {
  return (
    <header className="flex min-h-9 items-center justify-between gap-2 border-b px-3 py-1.5">
      <div className="flex items-center gap-2">
        <Skeleton className="size-3.5 rounded-sm" />
        <Skeleton className={cn("h-3.5", titleWidth)} />
        <Skeleton className="size-3.5 rounded-full" />
      </div>
      {right ? <Skeleton className={cn("h-3", right)} /> : null}
    </header>
  );
}

function IdentitySkeleton({ nameWidth, small }: { nameWidth: string; small?: boolean }) {
  return (
    <span className="flex min-w-0 items-center gap-2.5">
      <Skeleton className={cn("shrink-0 rounded-full", small ? "size-6" : "size-8")} />
      <span className="flex flex-col gap-1">
        <Skeleton className={cn("h-3.5", nameWidth)} />
        <Skeleton className="h-3 w-24" />
      </span>
    </span>
  );
}

function OverviewSkeleton() {
  return (
    <div className="grid grid-cols-4 gap-3 lg:grid-cols-8">
      {KPI_LABEL_WIDTHS.map((width, index) => (
        <KpiCard key={index} span={2}>
          <div className="flex min-h-[14px] items-center gap-1.5">
            <Skeleton className="size-[11px] rounded-sm" />
            <Skeleton className={cn("h-2.5", width)} />
          </div>
          {index === 1 ? (
            <div className="flex items-center gap-2.5">
              <Skeleton className="size-10 rounded-full" />
              <Skeleton className="h-6.5 w-14" />
            </div>
          ) : (
            <Skeleton className={cn("h-6.5", index === 3 ? "w-16" : "w-10")} />
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

function AttentionSkeleton() {
  const t = useT();

  return (
    <section
      aria-label={t("Needs your attention")}
      className="bg-card overflow-hidden rounded-lg border"
    >
      <header className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex items-center gap-2">
          <Skeleton className="size-3.5 rounded-sm" />
          <Skeleton className="h-3.5 w-32" />
          <Skeleton className="h-4 w-5 rounded-full" />
        </div>
        <Skeleton className="h-3 w-28" />
      </header>
      <ul className="divide-y">
        {Array.from({ length: ATTENTION_ROW_COUNT }, (_, index) => (
          <li
            key={index}
            className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2"
          >
            <div className="flex min-w-0 flex-wrap items-center gap-x-4 gap-y-1">
              <IdentitySkeleton nameWidth={index === 0 ? "w-28" : "w-24"} small />
              <span className="flex items-center gap-1">
                <Skeleton className="h-4 w-20 rounded-full" />
                {index === 0 ? <Skeleton className="h-4 w-24 rounded-full" /> : null}
              </span>
            </div>
            <Skeleton className="size-4 rounded-sm" />
          </li>
        ))}
      </ul>
    </section>
  );
}

function RosterSkeleton() {
  const t = useT();

  return (
    <section aria-label={t("Roster")} className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Skeleton className="h-7 w-72 max-w-full rounded-md" />
          <Skeleton className="h-6 w-28 rounded-md" />
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Skeleton className="h-7 w-48 rounded-md" />
          <span className="flex items-center gap-2">
            <Skeleton className="h-4.5 w-8 rounded-full" />
            <Skeleton className="h-3 w-40" />
          </span>
        </div>
      </div>
      {ROSTER_GROUPS.map((group) => (
        <section key={group.label} aria-label={group.label} className="flex flex-col gap-1.5">
          <header className="flex items-baseline justify-between gap-2 px-1">
            <span className="flex items-center gap-1.5">
              <Skeleton className="h-3 w-24" />
              <Skeleton className="h-3 w-4" />
            </span>
            <Skeleton className="h-3 w-24" />
          </header>
          <ul className="bg-card divide-y overflow-hidden rounded-lg border">
            {group.names.map((nameWidth, index) => (
              <li
                key={index}
                className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2"
              >
                <IdentitySkeleton nameWidth={nameWidth} />
                <div className="flex items-center gap-4">
                  <span className="hidden items-center gap-3 md:flex">
                    <Skeleton className="size-1.5 rounded-full" />
                    <Skeleton className="size-1.5 rounded-full" />
                    <Skeleton className="size-1.5 rounded-full" />
                  </span>
                  <Skeleton className="hidden h-3 w-14 sm:block" />
                  <Skeleton className="size-4 rounded-sm" />
                </div>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </section>
  );
}

function ByTerminalSkeleton() {
  const t = useT();

  return (
    <section
      aria-label={t("By terminal")}
      className="bg-card flex flex-col overflow-hidden rounded-lg border"
    >
      <PanelHeaderSkeleton titleWidth="w-20" right="w-12" />
      <ul className="divide-y">
        {TERMINAL_BAR_WIDTHS.map((width, index) => (
          <li key={index} className="flex flex-col gap-1.5 px-3 py-2">
            <span className="flex h-4 items-center justify-between gap-2">
              <span className="flex items-center gap-2">
                <Skeleton className="size-2 rounded-full" />
                <Skeleton className={cn("h-3", index === 1 ? "w-12" : "w-16")} />
              </span>
              <Skeleton className="h-3 w-4" />
            </span>
            <span className="bg-muted flex h-1 w-full overflow-hidden rounded-full">
              <Skeleton className={cn("h-full rounded-full", width)} />
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

function ComingUpSkeleton() {
  const t = useT();

  return (
    <section
      aria-label={t("Coming up")}
      className="bg-card flex flex-col overflow-hidden rounded-lg border"
    >
      <PanelHeaderSkeleton titleWidth="w-20" />
      <ul className="divide-y">
        {Array.from({ length: COMING_UP_ROW_COUNT }, (_, index) => (
          <li key={index} className="flex items-center gap-2.5 px-3 py-2">
            <Skeleton className="size-6 shrink-0 rounded-full" />
            <span className="flex min-w-0 flex-1 flex-col gap-1">
              <Skeleton className={cn("h-3", index === 1 ? "w-20" : "w-28")} />
              <Skeleton className="h-2.5 w-24" />
            </span>
            <Skeleton className="h-3 w-12" />
          </li>
        ))}
      </ul>
    </section>
  );
}

function CoverSkeleton() {
  const t = useT();

  return (
    <section
      aria-label={t("Approval cover")}
      className="bg-card flex flex-col overflow-hidden rounded-lg border"
    >
      <PanelHeaderSkeleton titleWidth="w-24" right="w-16" />
      <ul className="divide-y">
        {Array.from({ length: COVER_ROW_COUNT }, (_, index) => (
          <li key={index} className="flex flex-col gap-1.5 px-3 py-2">
            <span className="flex items-center justify-between gap-2">
              <Skeleton className={cn("h-3", index === 0 ? "w-24" : "w-28")} />
              <Skeleton className="h-4 w-14 rounded-full" />
            </span>
            <Skeleton className="h-2.5 w-36" />
          </li>
        ))}
      </ul>
    </section>
  );
}

type MyTeamSkeletonProps = {
  /** Whether the aside carries the cover panel; false for somebody who may not read delegations. */
  showCover?: boolean;
};

/**
 * The root announces itself once; everything inside is shape, not content,
 * so it is hidden from assistive technology. The sections keep the loaded
 * panels' labels only so the two trees can be compared like for like.
 */
export function MyTeamSkeleton({ showCover = true }: MyTeamSkeletonProps) {
  const t = useT();

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Loading your team")}>
      <div className="contents" aria-hidden>
        <OverviewSkeleton />
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_20rem]">
          <div className="flex min-w-0 flex-col gap-4">
            <AttentionSkeleton />
            <RosterSkeleton />
          </div>
          <aside className="flex min-w-0 flex-col gap-4">
            <ByTerminalSkeleton />
            <ComingUpSkeleton />
            {showCover ? <CoverSkeleton /> : null}
          </aside>
        </div>
      </div>
    </div>
  );
}
