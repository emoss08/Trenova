import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";

/**
 * The capacity strip's loading shapes: the summary (the headline figure, its bar and legend)
 * and the dock of drivers or carriers, each at the size of what replaces it. The strip draws the
 * two parts while its capacity is read, and the whole strip stands in for itself around them.
 */

const DOCK_UNITS = 10;
const LEGEND_WIDTHS = ["w-12", "w-16", "w-12"] as const;

/** The summary's headline figure, its bar and its legend. */
export function CapacitySummarySkeleton() {
  return (
    <div className="contents" aria-hidden>
      <div className="flex items-baseline gap-2 text-3xl">
        <Skeleton className="inline-block h-6 w-10 align-middle" />
        <Skeleton className="h-3 w-32" />
      </div>
      <Skeleton className="h-1.25 w-full rounded-full" />
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
        {LEGEND_WIDTHS.map((width, index) => (
          <span key={index} className="inline-flex items-center gap-1.5">
            <Skeleton className="size-2 rounded-xs" />
            <Skeleton className={`inline-block h-2.5 align-middle ${width}`} />
          </span>
        ))}
      </div>
    </div>
  );
}

/** The dock: a row of avatars with a name and a caption under each, and the hint under them. */
export function CapacityDockSkeleton() {
  return (
    <div className="flex min-w-0 flex-1 flex-col gap-1.5" aria-hidden>
      <div className="min-w-0 overflow-hidden pr-12 pl-6">
        <div className="flex h-19 items-center">
          {Array.from({ length: DOCK_UNITS }, (_, index) => (
            <span key={index} className="flex w-17 shrink-0 justify-center">
              <span className="flex w-16 flex-col items-center gap-1 py-1">
                <Skeleton className="size-7.5 rounded-full" />
                <Skeleton className="h-2.5 w-10" />
                <Skeleton className="h-2 w-8" />
              </span>
            </span>
          ))}
        </div>
      </div>
      <div className="flex items-center gap-2 pl-6 text-xs">
        <Skeleton className="inline-block h-2.5 w-56 align-middle" />
        <span className="flex-1" />
        <Skeleton className="size-5.5" />
        <Skeleton className="size-5.5" />
      </div>
    </div>
  );
}

/** The whole strip: the provider switch or its label, the summary and the dock. */
export function CapacityStripSkeleton() {
  const t = useT();
  const { operationType } = useShipmentCapabilities();

  return (
    <div
      className="flex min-w-0 flex-col gap-3 @[760px]/board:flex-row @[760px]/board:items-stretch"
      aria-busy
      aria-label={t("Loading capacity")}
    >
      <div className="contents" aria-hidden>
        <div className="@[760px]/board:border-border flex shrink-0 flex-col gap-2 @[760px]/board:w-[260px] @[760px]/board:border-r @[760px]/board:pr-4">
          {operationType === "both" ? (
            <Skeleton className="h-7 w-44 rounded-md" />
          ) : (
            <div className="text-xs">
              <Skeleton className="inline-block h-2.5 w-24 align-middle" />
            </div>
          )}
          <CapacitySummarySkeleton />
        </div>
        <CapacityDockSkeleton />
      </div>
    </div>
  );
}
