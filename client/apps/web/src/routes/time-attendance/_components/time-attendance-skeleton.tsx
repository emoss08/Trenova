import { useT } from "@trenova/shared/i18n/use-t";
import { KpiCard } from "@/components/kpi/kpi-card";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Stands in twice: as the Suspense fallback while the console's chunk arrives,
 * and as the console's own loading state while its first lists are read. One
 * component for both keeps the page from reflowing between the two.
 *
 * Every measurement mirrors the loaded layout: the overview strip with its
 * week-composition cell, the underline tab strip, and the clock tab the page
 * opens on (the section heading with its worker picker, then the board of
 * everybody on the clock). Each text bar sits in a box the height of the line
 * it stands for, so the loaded strip lands where its outline was.
 */

const STRIP_LABEL_WIDTHS = ["w-24", "w-24", "w-20", "w-24"] as const;
const STRIP_SUB_WIDTHS = ["w-32", "w-40", "", "w-36"] as const;
const LEGEND_WIDTHS = ["w-10", "w-12", "w-8"] as const;
const TAB_WIDTHS = ["w-8", "w-16", "w-12"] as const;
const BOARD_ROW_COUNT = 3;
const BOARD_NAME_WIDTHS = ["w-28", "w-24", "w-32"] as const;
const BOARD_METER_WIDTHS = ["w-3/4", "w-1/2", "w-1/4"] as const;

function StripLabel({ width }: { width: string }) {
  return (
    <span className="flex h-4 items-center">
      <Skeleton className={cn("h-2.5", width)} />
    </span>
  );
}

function StripSub({ width }: { width: string }) {
  return (
    <span className="flex h-4 items-center">
      <Skeleton className={cn("h-2.5", width)} />
    </span>
  );
}

function OverviewSkeleton({ showPayroll }: { showPayroll: boolean }) {
  return (
    <KpiStrip>
      {[0, 1].map((index) => (
        <KpiStripItem
          key={index}
          label={<StripLabel width={STRIP_LABEL_WIDTHS[index]} />}
          value={<Skeleton className="h-4.25 w-10" />}
          sub={<StripSub width={STRIP_SUB_WIDTHS[index]} />}
        />
      ))}
      <KpiCard span={2}>
        <div className="flex min-h-[14px] items-center justify-between">
          <StripLabel width={STRIP_LABEL_WIDTHS[2]} />
        </div>
        <Skeleton className="h-4.25 w-16" />
        <div className="mt-auto flex min-w-0 flex-col gap-1.5">
          <Skeleton className="h-1 w-full rounded-full" />
          <div className="flex flex-wrap gap-x-3 gap-y-0.5">
            {LEGEND_WIDTHS.map((width, index) => (
              <span key={index} className="flex h-3.5 items-center gap-1">
                <Skeleton className="size-1.5 rounded-full" />
                <Skeleton className={cn("h-2", width)} />
                <Skeleton className="h-2 w-5" />
              </span>
            ))}
          </div>
        </div>
      </KpiCard>
      {showPayroll ? (
        <KpiStripItem
          label={<StripLabel width={STRIP_LABEL_WIDTHS[3]} />}
          value={<Skeleton className="h-4.25 w-10" />}
          sub={<StripSub width={STRIP_SUB_WIDTHS[3]} />}
        />
      ) : null}
    </KpiStrip>
  );
}

function TabStripSkeleton({ count }: { count: number }) {
  return (
    <div className="flex w-fit items-center gap-x-0.5 py-1">
      {TAB_WIDTHS.slice(0, count).map((width, index) => (
        <span key={index} className="flex h-9 items-center gap-1.5 px-2.5 sm:h-8">
          <Skeleton className="size-3.5 rounded-sm" />
          <Skeleton className={cn("h-3", width)} />
        </span>
      ))}
    </div>
  );
}

function ClockHeadingSkeleton() {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1 space-y-1">
          <span className="flex h-3.25 items-center">
            <Skeleton className="h-3 w-10" />
          </span>
          <span className="flex h-4.25 items-center">
            <Skeleton className="h-3 w-xl max-w-full" />
          </span>
        </div>
        <div className="w-56 shrink-0 sm:w-72">
          <div className="flex flex-col gap-0.5">
            <span className="flex h-4 items-center">
              <Skeleton className="h-2.5 w-12" />
            </span>
            <Skeleton className="h-7 w-full" />
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * The rows of the board while the open punches are read: an avatar, the name
 * over a shift meter, the running time and the punch-out button, in the
 * board's own row grid. The board draws it under its real header too.
 */
export function ClockBoardRowsSkeleton() {
  return (
    <ul aria-hidden className="divide-y">
      {Array.from({ length: BOARD_ROW_COUNT }, (_, index) => (
        <li
          key={index}
          className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2"
        >
          <span className="flex min-w-0 items-center gap-2.5">
            <Skeleton className="size-6 shrink-0 rounded-md" />
            <span className="flex min-w-0 flex-1 flex-col gap-1">
              <span className="flex h-3.75 items-center gap-1.5">
                <Skeleton className={cn("h-3", BOARD_NAME_WIDTHS[index])} />
                <Skeleton className="h-2.5 w-16" />
              </span>
              <span className="bg-muted flex h-1 w-40 max-w-full overflow-hidden rounded-full">
                <Skeleton className={cn("h-full rounded-full", BOARD_METER_WIDTHS[index])} />
              </span>
            </span>
          </span>
          <span className="flex items-center gap-3">
            <span className="flex h-4.25 items-center">
              <Skeleton className="h-3 w-10" />
            </span>
            <Skeleton className="h-6 w-20 rounded-md" />
          </span>
        </li>
      ))}
    </ul>
  );
}

function ClockBoardSkeleton() {
  return (
    <div className="bg-card overflow-hidden rounded-lg border">
      <div className="border-border flex min-h-9 flex-wrap items-center justify-between gap-2 border-b px-3 py-1.5">
        <div className="flex items-center gap-2">
          <Skeleton className="size-3.5 rounded-sm" />
          <span className="flex h-4.25 items-center">
            <Skeleton className="h-3 w-28" />
          </span>
        </div>
        <Skeleton className="h-6 w-20 rounded-md" />
      </div>
      <ClockBoardRowsSkeleton />
    </div>
  );
}

type TimeAttendanceSkeletonProps = {
  /** Whether the strip and the tabs carry payroll; false for somebody who may not export. */
  showPayroll: boolean;
};

/**
 * The root announces itself once; everything inside is shape, not content,
 * so it is hidden from assistive technology.
 */
export function TimeAttendanceSkeleton({ showPayroll }: TimeAttendanceSkeletonProps) {
  const t = useT();

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Loading time and attendance")}>
      <div className="contents" aria-hidden>
        <OverviewSkeleton showPayroll={showPayroll} />
        <div className="flex flex-col gap-2">
          <TabStripSkeleton count={showPayroll ? 3 : 2} />
          <div className="flex flex-col gap-4">
            <ClockHeadingSkeleton />
            <ClockBoardSkeleton />
          </div>
        </div>
      </div>
    </div>
  );
}
