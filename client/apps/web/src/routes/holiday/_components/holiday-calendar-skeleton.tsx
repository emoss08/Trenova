import { useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { daysInMonth } from "@trenova/shared/lib/holiday";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";

/**
 * Stands in twice: the page skeleton is the Suspense fallback while the
 * calendar's chunk arrives, and the month grid inside it is the calendar's own
 * loading state while a year is read. Sharing the grid keeps the page from
 * reflowing between "loading the code" and "loading the year".
 *
 * Every measurement mirrors the loaded layout: the year stepper, the "This
 * year" button, the count and the legend, then twelve month cards with the
 * weekday letters and one cell per day, offset to the weekday the month starts
 * on, so a six-week month is as tall here as it is once loaded.
 */

const MONTH_NAME_WIDTHS = [
  "w-14",
  "w-16",
  "w-10",
  "w-8",
  "w-7",
  "w-8",
  "w-7",
  "w-12",
  "w-18",
  "w-14",
  "w-16",
  "w-16",
] as const;
const WEEKDAY_COUNT = 7;
const LEGEND_WIDTHS = ["w-12", "w-14"] as const;

type HolidayMonthsSkeletonProps = {
  year: number;
};

/** The twelve month cards of `year`, each with its days where the real ones fall. */
export function HolidayMonthsSkeleton({ year }: HolidayMonthsSkeletonProps) {
  const t = useT();

  return (
    <div
      className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4"
      aria-busy
      aria-label={t("Loading the calendar for {0}", year)}
    >
      {MONTH_NAME_WIDTHS.map((width, month) => {
        const leading = new Date(Date.UTC(year, month, 1)).getUTCDay();
        const total = daysInMonth(year, month);
        return (
          <div key={month} aria-hidden className="bg-card rounded-lg border p-3">
            <span className="mb-2 flex h-4.25 items-center">
              <Skeleton className={cn("h-3", width)} />
            </span>
            <div className="grid grid-cols-7 gap-y-0.5">
              {Array.from({ length: WEEKDAY_COUNT }, (_, index) => (
                <span key={`weekday-${index}`} className="flex h-3.5 items-center justify-center">
                  <Skeleton className="h-2 w-1.5 rounded-sm" />
                </span>
              ))}
              {Array.from({ length: leading }, (_, index) => (
                <span key={`pad-${index}`} />
              ))}
              {Array.from({ length: total }, (_, index) => (
                <span key={index} className="mx-auto flex size-6 items-center justify-center">
                  <Skeleton className="h-2 w-2.5 rounded-sm" />
                </span>
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}

function ToolbarSkeleton({ canCreate }: { canCreate: boolean }) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="flex items-center gap-2">
        <div className="bg-muted/40 flex items-center rounded-lg border p-0.5">
          <span className="flex size-7 items-center justify-center">
            <Skeleton className="size-3.5 rounded-sm" />
          </span>
          <span className="flex h-4.25 min-w-14 items-center justify-center">
            <Skeleton className="h-3 w-8" />
          </span>
          <span className="flex size-7 items-center justify-center">
            <Skeleton className="size-3.5 rounded-sm" />
          </span>
        </div>
        <Skeleton className="h-7 w-18 rounded-md" />
        <span className="hidden h-4 items-center sm:flex">
          <Skeleton className="h-2.5 w-36" />
        </span>
        <Skeleton className="size-4 rounded-sm" />
      </div>
      <div className="flex items-center gap-3">
        <div className="hidden h-4 items-center gap-3 sm:flex">
          {LEGEND_WIDTHS.map((width, index) => (
            <span key={index} className="flex items-center gap-1">
              <Skeleton className="size-2 rounded-full" />
              <Skeleton className={cn("h-2.5", width)} />
            </span>
          ))}
        </div>
        {canCreate ? <Skeleton className="h-7 w-22 rounded-md" /> : null}
      </div>
    </div>
  );
}

type HolidayCalendarSkeletonProps = {
  /** Whether the toolbar carries "Add date"; false for somebody who may not create one. */
  canCreate: boolean;
};

/**
 * The root announces itself once; everything inside is shape, not content,
 * so it is hidden from assistive technology.
 */
export function HolidayCalendarSkeleton({ canCreate }: HolidayCalendarSkeletonProps) {
  const t = useT();
  const [year] = useState(() => new Date().getUTCFullYear());

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Loading the holiday calendar")}>
      <div className="contents" aria-hidden>
        <ToolbarSkeleton canCreate={canCreate} />
        <HolidayMonthsSkeleton year={year} />
      </div>
    </div>
  );
}
