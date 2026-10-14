import { useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { CSA_BASIC_ORDER } from "@trenova/shared/lib/csa";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Stands in twice: as the Suspense fallback while the "My DOT" tab's chunk
 * arrives, and as the tab's own loading state while the organization's DOT
 * profile and the provider settings are read. One component for both keeps
 * the tab from reflowing between the two.
 *
 * Every measurement mirrors a pulled profile: the snapshot header with its
 * risk and freshness line and the refresh button, the decision headline over
 * its findings, the key facts grid, and the profile tabs opened on safety,
 * with the rating facts and a row for each of the seven CSA BASICs. Each text
 * bar sits in a box the height of the line it stands for.
 */

const META_WIDTHS = ["w-16", "w-28", "w-36"] as const;
const FINDING_COUNT = 2;
const FINDING_WIDTHS = ["w-72", "w-56"] as const;
const KEY_FACT_COUNT = 14;
const FACT_LABEL_WIDTHS = ["w-20", "w-24", "w-16", "w-20", "w-24", "w-28", "w-12"] as const;
const FACT_VALUE_WIDTHS = ["w-16", "w-28", "w-20", "w-24", "w-32", "w-14", "w-20"] as const;
const PROFILE_TAB_WIDTHS = ["w-12", "w-16", "w-14", "w-16"] as const;
const SAFETY_FACT_COUNT = 7;
const BASIC_COLUMN_COUNT = 4;

function Line({ height, width }: { height: string; width: string }) {
  return (
    <span className={cn("flex items-center", height)}>
      <Skeleton className={cn("h-3", width)} />
    </span>
  );
}

function SnapshotHeaderSkeleton() {
  return (
    <div className="flex flex-col gap-3 border-b pb-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          <Line height="h-4.25" width="w-56" />
          <div className="flex h-4 flex-wrap items-center gap-x-2">
            {META_WIDTHS.map((width, index) => (
              <span key={index} className="flex items-center gap-2">
                {index > 0 ? <Skeleton className="size-0.5 rounded-full" /> : null}
                {index === 0 ? <Skeleton className="size-2 rounded-full" /> : null}
                <Skeleton className={cn("h-2.5", width)} />
              </span>
            ))}
          </div>
          <span className="flex h-4 items-center">
            <Skeleton className="h-2.5 w-xl max-w-full" />
          </span>
        </div>
        <Skeleton className="h-7 w-18 shrink-0 rounded-md" />
      </div>
    </div>
  );
}

function DecisionSkeleton() {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex h-4.25 items-center gap-2">
        <Skeleton className="size-2 rounded-full" />
        <Skeleton className="h-3 w-32" />
      </div>
      <ul className="divide-border divide-y">
        {FINDING_WIDTHS.slice(0, FINDING_COUNT).map((width, index) => (
          <li key={index} className="flex items-start gap-2.5 py-2">
            <Skeleton className="mt-0.5 size-3.5 shrink-0 rounded-sm" />
            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
              <Line height="h-4.25" width={width} />
              <span className="flex h-4 items-center">
                <Skeleton className="h-2.5 w-40" />
              </span>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}

function FactGridSkeleton({ count }: { count: number }) {
  return (
    <div className="grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-2">
      {Array.from({ length: count }, (_, index) => (
        <div key={index} className="flex min-w-0 flex-col gap-0.5">
          <span className="flex h-4 items-center">
            <Skeleton
              className={cn("h-2.5", FACT_LABEL_WIDTHS[index % FACT_LABEL_WIDTHS.length])}
            />
          </span>
          <Line height="h-4.25" width={FACT_VALUE_WIDTHS[index % FACT_VALUE_WIDTHS.length]} />
        </div>
      ))}
    </div>
  );
}

function BasicsTableSkeleton() {
  return (
    <div className="flex flex-col gap-2">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b">
            <th className="h-8 pr-2 text-left">
              <Skeleton className="h-2.5 w-10" />
            </th>
            {Array.from({ length: BASIC_COLUMN_COUNT }, (_, index) => (
              <th key={index} className="h-8 px-2 last:pr-0">
                <Skeleton className="ml-auto h-2.5 w-14" />
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {CSA_BASIC_ORDER.map((basic) => (
            <tr key={basic} className="border-b last:border-b-0">
              <td className="h-9 pr-2 align-middle">
                <Skeleton className="h-3 w-36" />
              </td>
              {Array.from({ length: BASIC_COLUMN_COUNT }, (_, index) => (
                <td key={index} className="h-9 px-2 align-middle last:pr-0">
                  <Skeleton className="ml-auto h-3 w-8" />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      <span className="flex h-4 items-center">
        <Skeleton className="h-2.5 w-48" />
      </span>
    </div>
  );
}

function ProfileSkeleton() {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex w-full items-center justify-start gap-x-0.5 border-b py-1">
        {PROFILE_TAB_WIDTHS.map((width, index) => (
          <span key={index} className="flex h-9 items-center px-2 sm:h-8">
            <Skeleton className={cn("h-3", width)} />
          </span>
        ))}
      </div>
      <div className="flex flex-col gap-6">
        <div className="flex flex-col gap-2">
          <span className="flex h-4 items-center">
            <Skeleton className="h-2.5 w-12" />
          </span>
          <FactGridSkeleton count={SAFETY_FACT_COUNT} />
        </div>
        <div className="flex flex-col gap-2">
          <span className="flex h-4 items-center">
            <Skeleton className="h-2.5 w-20" />
          </span>
          <BasicsTableSkeleton />
        </div>
      </div>
    </div>
  );
}

/**
 * The root announces itself once; everything inside is shape, not content,
 * so it is hidden from assistive technology.
 */
export function MyDotSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Loading your DOT profile")}>
      <div className="contents" aria-hidden>
        <SnapshotHeaderSkeleton />
        <DecisionSkeleton />
        <div className="flex flex-col gap-6">
          <FactGridSkeleton count={KEY_FACT_COUNT} />
          <ProfileSkeleton />
        </div>
      </div>
    </div>
  );
}
