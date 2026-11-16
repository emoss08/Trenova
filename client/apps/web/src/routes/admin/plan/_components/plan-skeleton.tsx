import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";

function PanelSkeleton({ children }: { children: ReactNode }) {
  return (
    <div className="flex flex-col overflow-hidden rounded-lg border bg-card">
      <div className="flex min-h-9 items-center border-b border-border px-3 py-1.5">
        <Skeleton className="h-4 w-32" />
      </div>
      {children}
    </div>
  );
}

function UsageSkeleton() {
  return (
    <PanelSkeleton>
      <div className="grid grid-cols-1 sm:grid-cols-[repeat(auto-fit,minmax(200px,1fr))]">
        {Array.from({ length: 3 }, (_, index) => (
          <div
            key={index}
            className="flex flex-col gap-2 border-t border-border-subtle p-3 first:border-t-0 sm:border-t-0 sm:border-l sm:first:border-l-0"
          >
            <div className="flex items-center justify-between gap-2">
              <Skeleton className="h-3.5 w-24" />
              <Skeleton className="h-3.5 w-16" />
            </div>
            <Skeleton className="h-1 w-full" />
          </div>
        ))}
      </div>
    </PanelSkeleton>
  );
}

function AddOnsSkeleton({ rows }: { rows: number }) {
  return (
    <PanelSkeleton>
      <div className="grid grid-cols-1 sm:grid-cols-2">
        {Array.from({ length: rows }, (_, index) => (
          <div
            key={index}
            className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1.5 border-t border-border-subtle px-3 py-2.5 max-sm:first:border-t-0 sm:odd:border-r sm:nth-[-n+2]:border-t-0"
          >
            <Skeleton className="h-4 w-36" />
            <Skeleton className="col-start-2 row-span-2 row-start-1 h-5 w-12" />
            <Skeleton className="col-start-1 h-3 w-full max-w-72" />
          </div>
        ))}
      </div>
    </PanelSkeleton>
  );
}

/**
 * The plan page as it lands: usage, the add-ons on the plan and those that can be added, what
 * every plan includes, and recent changes.
 */
export function PlanSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Loading your plan")}>
      <div aria-hidden className="flex flex-col gap-4">
        <UsageSkeleton />
        <AddOnsSkeleton rows={2} />
        <AddOnsSkeleton rows={6} />
        <PanelSkeleton>
          <div className="flex flex-col gap-1.5 px-3 py-3">
            <Skeleton className="h-3 w-full max-w-xl" />
            <Skeleton className="h-3 w-64" />
          </div>
        </PanelSkeleton>
        <PanelSkeleton>
          <div className="flex flex-col divide-y divide-border-subtle">
            {Array.from({ length: 3 }, (_, index) => (
              <div key={index} className="grid grid-cols-[96px_minmax(0,1fr)] gap-x-3 px-3 py-2.5">
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-3 w-72 max-w-full" />
              </div>
            ))}
          </div>
        </PanelSkeleton>
      </div>
    </div>
  );
}
