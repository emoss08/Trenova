import { useT } from "@trenova/shared/i18n/use-t";
import { KpiCard } from "@/components/kpi/kpi-card";
import { YEARS_OFFERED } from "@/lib/osha-log";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

/**
 * Stands in twice: as the Suspense fallback while the console's chunk arrives,
 * and as the console's own loading state while a year's log is read. One
 * component for both keeps the page from reflowing between the two.
 *
 * Every measurement mirrors the loaded layout: the captioned year picker, the
 * KPI strip at the shared KPI height, the 300A card with its blocks of
 * figures and establishment rows beside the certification track, and the
 * Form 300 table with its toolbar. Widths are fixed so the shapes never
 * reshuffle.
 */

const KPI_LABEL_WIDTHS = ["w-24", "w-20", "w-16", "w-14"] as const;
// The 300A's blocks: four case columns (G to J), two day totals (K and L)
// and six illness types, the way the paper form lays them out.
const CASE_COLUMN_COUNT = 4;
const DAY_TOTAL_COUNT = 2;
const ILLNESS_TYPE_COUNT = 6;
const ESTABLISHMENT_ROW_WIDTHS = ["w-16", "w-40", "w-44"] as const;
const TRACK_STEP_COUNT = 4;
const TRACK_LABEL_WIDTHS = ["w-24", "w-36", "w-32", "w-40"] as const;
const CASE_ROW_COUNT = 5;
const CASE_NAME_WIDTHS = ["w-24", "w-20", "w-28", "w-16", "w-24"] as const;
const CASE_WHAT_WIDTHS = ["w-48", "w-56", "w-40", "w-52", "w-44"] as const;

function YearPickerSkeleton() {
  return (
    <div className="flex h-9 w-fit items-center gap-1 rounded-md border p-1">
      {Array.from({ length: YEARS_OFFERED }, (_, index) => (
        <span key={index} className="flex flex-col items-center gap-1 px-3">
          <Skeleton className="h-2.5 w-8" />
          <Skeleton className="h-2 w-10" />
        </span>
      ))}
    </div>
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
          <Skeleton className={cn("h-6.5", index === 1 || index === 2 ? "w-14" : "w-10")} />
          {index === 0 || index === 3 ? (
            <Skeleton className="mt-auto h-1.5 w-full rounded-full" />
          ) : (
            <Skeleton className="mt-auto h-2.5 w-3/4" />
          )}
        </KpiCard>
      ))}
    </div>
  );
}

function FigureSkeleton() {
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <div className="flex items-center gap-1.5">
        <Skeleton className="size-4 rounded-sm" />
        <Skeleton className="h-2.5 w-3/4" />
      </div>
      <Skeleton className="h-4 w-6" />
    </div>
  );
}

function BlockSkeleton({
  label,
  titleWidth,
  className,
  children,
}: {
  label: string;
  titleWidth: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <section aria-label={label} className={cn("flex min-w-0 flex-col gap-2", className)}>
      <Skeleton className={cn("h-3", titleWidth)} />
      {children}
    </section>
  );
}

function SummaryCardSkeleton() {
  const t = useT();

  return (
    <section
      aria-label={t("Form 300A")}
      className="bg-card flex min-w-0 flex-col rounded-lg border"
    >
      <header className="flex flex-wrap items-start justify-between gap-3 border-b px-4 py-3">
        <div className="flex flex-col gap-1.5">
          <span className="flex items-center gap-2">
            <Skeleton className="h-3.5 w-44" />
            <Skeleton className="h-4 w-12 rounded-full" />
          </span>
          <Skeleton className="h-3 w-64" />
        </div>
        <div className="flex items-center gap-2">
          <Skeleton className="h-8 w-28 rounded-md" />
          <Skeleton className="h-8 w-24 rounded-md" />
        </div>
      </header>
      <div className="grid gap-x-8 gap-y-5 px-4 py-4 md:grid-cols-[3fr_2fr]">
        <BlockSkeleton label={t("Number of cases")} titleWidth="w-24">
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            {Array.from({ length: CASE_COLUMN_COUNT }, (_, index) => (
              <FigureSkeleton key={index} />
            ))}
          </div>
        </BlockSkeleton>
        <BlockSkeleton label={t("Number of days")} titleWidth="w-24">
          <div className="grid grid-cols-2 gap-3">
            {Array.from({ length: DAY_TOTAL_COUNT }, (_, index) => (
              <FigureSkeleton key={index} />
            ))}
          </div>
        </BlockSkeleton>
        <BlockSkeleton
          label={t("Injury and illness types")}
          titleWidth="w-36"
          className="md:col-span-2"
        >
          <div className="grid grid-cols-3 gap-3 sm:grid-cols-6">
            {Array.from({ length: ILLNESS_TYPE_COUNT }, (_, index) => (
              <FigureSkeleton key={index} />
            ))}
          </div>
        </BlockSkeleton>
        <BlockSkeleton label={t("Establishment information")} titleWidth="w-40">
          <div className="divide-border/60 flex flex-col divide-y">
            {ESTABLISHMENT_ROW_WIDTHS.map((width, index) => (
              <div
                key={index}
                className="flex items-center justify-between gap-3 py-1.5 first:pt-0 last:pb-0"
              >
                <Skeleton className={cn("h-3", width)} />
                <Skeleton className="h-3 w-12" />
              </div>
            ))}
          </div>
        </BlockSkeleton>
        <BlockSkeleton label={t("Certification")} titleWidth="w-24">
          <div className="flex flex-col gap-1.5">
            <Skeleton className="h-3 w-40" />
            <Skeleton className="h-3 w-52" />
            <Skeleton className="h-3 w-32" />
          </div>
        </BlockSkeleton>
      </div>
    </section>
  );
}

function TrackSkeleton() {
  const t = useT();

  return (
    <aside className="bg-card flex min-w-0 flex-col rounded-lg border">
      <header className="flex items-center gap-2 border-b px-3 py-2">
        <Skeleton className="size-3.5 rounded-sm" />
        <Skeleton className="h-3.5 w-32" />
        <Skeleton className="size-3.5 rounded-full" />
      </header>
      <ol aria-label={t("Where the year stands")} className="flex flex-col p-3">
        {Array.from({ length: TRACK_STEP_COUNT }, (_, index) => {
          const last = index === TRACK_STEP_COUNT - 1;
          return (
            <li key={index} className="grid grid-cols-[1rem_minmax(0,1fr)] gap-x-3">
              <div className="flex flex-col items-center">
                <Skeleton className="size-4 rounded-full" />
                {last ? null : <span className="bg-border relative my-1 w-px flex-1" />}
              </div>
              <div className={cn("flex min-w-0 flex-col gap-1", last ? "pb-0" : "pb-4")}>
                <Skeleton className={cn("h-3", TRACK_LABEL_WIDTHS[index])} />
                <Skeleton className="h-2.5 w-full max-w-48" />
              </div>
            </li>
          );
        })}
      </ol>
    </aside>
  );
}

function CaseTableSkeleton() {
  const t = useT();

  return (
    <section aria-label={t("Form 300")} className="flex min-w-0 flex-col gap-3">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="flex flex-col gap-1.5">
          <Skeleton className="h-3.5 w-40" />
          <Skeleton className="h-3 w-72" />
        </div>
        <div className="flex items-center gap-2">
          <Skeleton className="h-7 w-44 rounded-md" />
          <Skeleton className="h-7 w-56 rounded-md" />
        </div>
      </div>
      <div className="bg-card overflow-hidden rounded-lg border">
        <table aria-label={t("Cases")} className="w-full text-sm">
          <thead>
            <tr className="bg-sidebar h-10 border-b">
              <th className="w-20 px-2">
                <Skeleton className="h-3 w-10" />
              </th>
              <th className="px-2">
                <Skeleton className="h-3 w-16" />
              </th>
              <th className="w-24 px-2">
                <Skeleton className="h-3 w-10" />
              </th>
              <th className="px-2">
                <Skeleton className="h-3 w-24" />
              </th>
              <th className="px-2">
                <Skeleton className="h-3 w-28" />
              </th>
              <th className="w-16 px-2">
                <Skeleton className="mx-auto h-3 w-12" />
              </th>
              <th className="w-16 px-2">
                <Skeleton className="ml-auto h-3 w-10" />
              </th>
              <th className="w-20 px-2">
                <Skeleton className="ml-auto h-3 w-16" />
              </th>
              <th className="w-36 px-2">
                <Skeleton className="h-3 w-10" />
              </th>
              <th className="w-24 px-2">
                <Skeleton className="h-3 w-12" />
              </th>
              <th className="w-10 px-2" />
            </tr>
          </thead>
          <tbody>
            {Array.from({ length: CASE_ROW_COUNT }, (_, index) => (
              <tr key={index} className="border-b last:border-b-0">
                <td className="px-2 py-2">
                  <Skeleton className="h-3.5 w-14" />
                </td>
                <td className="px-2 py-2">
                  <Skeleton className={cn("h-3.5", CASE_NAME_WIDTHS[index])} />
                </td>
                <td className="px-2 py-2">
                  <Skeleton className="h-3.5 w-16" />
                </td>
                <td className="px-2 py-2">
                  <Skeleton className="h-3.5 w-24" />
                </td>
                <td className="px-2 py-2">
                  <span className="flex flex-col gap-1">
                    <Skeleton className={cn("h-3.5", CASE_WHAT_WIDTHS[index])} />
                    <Skeleton className="h-2.5 w-20" />
                  </span>
                </td>
                <td className="px-2 py-2">
                  <Skeleton className="mx-auto size-4 rounded-sm" />
                </td>
                <td className="px-2 py-2">
                  <Skeleton className="ml-auto h-3.5 w-6" />
                </td>
                <td className="px-2 py-2">
                  <Skeleton className="ml-auto h-3.5 w-6" />
                </td>
                <td className="px-2 py-2">
                  <span className="flex items-center gap-1.5">
                    <Skeleton className="size-4 rounded-sm" />
                    <Skeleton className="h-3.5 w-20" />
                  </span>
                </td>
                <td className="px-2 py-2">
                  <Skeleton className="h-3.5 w-12" />
                </td>
                <td className="px-1 py-1">
                  <Skeleton className="ml-auto size-6 rounded-md" />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

/**
 * The root announces itself once; everything inside is shape, not content,
 * so it is hidden from assistive technology. The sections keep the loaded
 * panels' labels only so the two trees can be compared like for like.
 */
export function OshaLogSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Loading the log")}>
      <div className="contents" aria-hidden>
        <YearPickerSkeleton />
        <OverviewSkeleton />
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_18rem]">
          <SummaryCardSkeleton />
          <TrackSkeleton />
        </div>
        <CaseTableSkeleton />
      </div>
    </div>
  );
}
