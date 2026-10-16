import { useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { PANEL_CHART_HEIGHT } from "./dashboard-panel";

/**
 * The performance panel's body before its figures answer: the two headline
 * figures, the collected bar and the three rates, each where it lands.
 */
export function CollectionsPerformanceBodySkeleton() {
  const t = useT();

  return (
    <div
      className={cn("flex flex-col justify-between gap-3 p-3", PANEL_CHART_HEIGHT)}
      aria-busy
      aria-label={t("Loading collections performance")}
    >
      <div className="grid grid-cols-2 gap-4" aria-hidden>
        {Array.from({ length: 2 }, (_, index) => (
          <div key={index} className="flex flex-col gap-2">
            <Skeleton className="h-2.5 w-28" />
            <Skeleton className="h-6 w-16" />
            <Skeleton className="h-1.5 w-full" />
          </div>
        ))}
      </div>
      <div className="flex flex-col gap-1.5" aria-hidden>
        <div className="flex justify-between">
          <Skeleton className="h-2.5 w-32" />
          <Skeleton className="h-2.5 w-36" />
        </div>
        <Skeleton className="h-1.5 w-full" />
      </div>
      <div className="grid grid-cols-3 gap-3 rounded-md border px-3 py-2.5" aria-hidden>
        {Array.from({ length: 3 }, (_, index) => (
          <div key={index} className="flex flex-col gap-1.5">
            <Skeleton className="h-2.5 w-16" />
            <Skeleton className="h-4 w-12" />
            <Skeleton className="h-2 w-14" />
          </div>
        ))}
      </div>
    </div>
  );
}
