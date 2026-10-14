import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useShipmentBoardUrl } from "../url-state";

/**
 * The floating panel's loading shapes. Each part is drawn by its own component while its query
 * answers, and the panel skeleton puts the same parts in the same frame, so the panel looks one
 * way however it is waiting.
 */

/** The panel's frame: it floats over the board with a stronger border and no shadow. */
export const SIDE_PANEL_FRAME_CLASS =
  "animate-panel-in bg-raised border-border-strong absolute top-2 right-3 bottom-3 z-30 flex w-[min(372px,calc(100%-28px))] flex-col overflow-hidden rounded-lg border @max-[820px]/board:inset-x-2 @max-[820px]/board:w-auto";

const UP_NEXT_WIDTHS = ["w-3/5", "w-2/3", "w-1/2"] as const;
const DELIVERY_BARS = [20, 35, 55, 80, 65, 90, 70, 45, 60, 30, 25, 15] as const;
const DETENTION_WIDTHS = ["w-2/3", "w-1/2"] as const;
const BILLING_WIDTHS = ["w-24", "w-32", "w-20"] as const;
const ACTIVITY_WIDTHS = ["w-11/12", "w-3/4", "w-5/6", "w-2/3", "w-4/5", "w-3/5"] as const;

/** The suggested actions: the heading, the progress ticks, the card in front and what is next. */
export function ActionQueueSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-3" aria-busy aria-label={t("Loading suggested actions")}>
      <div className="contents" aria-hidden>
        <div className="flex items-center justify-between text-xs">
          <Skeleton className="inline-block h-2.5 w-28 align-middle" />
          <Skeleton className="inline-block h-2.5 w-10 align-middle" />
        </div>
        <div className="flex gap-1">
          {UP_NEXT_WIDTHS.map((_, index) => (
            <Skeleton key={index} className="h-0.75 flex-1 rounded-full" />
          ))}
        </div>
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-1.5 text-xs">
            <Skeleton className="size-1.5 rounded-full" />
            <Skeleton className="inline-block h-2.5 w-20 align-middle" />
          </div>
          <div className="text-lg leading-snug">
            <Skeleton className="inline-block h-4 w-4/5 align-middle" />
          </div>
          <div className="flex flex-col text-sm">
            <span>
              <Skeleton className="inline-block h-3 w-full align-middle" />
            </span>
            <span>
              <Skeleton className="inline-block h-3 w-2/3 align-middle" />
            </span>
          </div>
          <div className="flex items-center gap-1.5 pt-1">
            <Skeleton className="h-7 w-24" />
            <Skeleton className="h-7 w-16" />
          </div>
        </div>
        <div className="flex flex-col">
          <div className="pb-1 text-xs">
            <Skeleton className="inline-block h-2.5 w-14 align-middle" />
          </div>
          {UP_NEXT_WIDTHS.map((width, index) => (
            <div key={index} className="flex h-7 items-center gap-2">
              <Skeleton className="size-1.5 shrink-0 rounded-full" />
              <Skeleton className={cn("h-3", width)} />
              <Skeleton className="ml-auto h-2.5 w-10" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function WatchSectionSkeleton({ children }: { children: React.ReactNode }) {
  return (
    <div className="border-border flex flex-col gap-2.5 border-t pt-3 first:border-t-0 first:pt-0">
      <div className="flex items-baseline justify-between gap-2">
        <div className="text-xs">
          <Skeleton className="inline-block h-2.5 w-24 align-middle" />
        </div>
        <div className="text-sm">
          <Skeleton className="inline-block h-3 w-20 align-middle" />
        </div>
      </div>
      {children}
    </div>
  );
}

/** The day at a glance: deliveries by hour, uncovered pickups, detention and billing. */
export function WatchlistSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-3" aria-busy aria-label={t("Loading the watchlist")}>
      <div className="contents" aria-hidden>
        <WatchSectionSkeleton>
          <div className="flex flex-col gap-1">
            <div className="flex h-14 items-end gap-0.5">
              {DELIVERY_BARS.map((height, index) => (
                <Skeleton
                  key={index}
                  className="flex-1 rounded-none rounded-t-xs"
                  style={{ height: `${height}%` }}
                />
              ))}
            </div>
            <div className="text-2xs flex justify-between">
              <Skeleton className="inline-block h-2 w-6 align-middle" />
              <Skeleton className="inline-block h-2 w-6 align-middle" />
              <Skeleton className="inline-block h-2 w-6 align-middle" />
            </div>
          </div>
        </WatchSectionSkeleton>
        <WatchSectionSkeleton>
          <div className="grid grid-cols-4 gap-1">
            {[0, 1, 2, 3].map((index) => (
              <Skeleton key={index} className="h-14 rounded-md" />
            ))}
          </div>
        </WatchSectionSkeleton>
        <WatchSectionSkeleton>
          <div className="flex flex-col gap-1.5">
            {DETENTION_WIDTHS.map((width, index) => (
              <div key={index} className="flex flex-col gap-1">
                <div className="flex items-center gap-2 text-sm">
                  <Skeleton className={cn("inline-block h-3 align-middle", width)} />
                  <Skeleton className="ml-auto inline-block h-2.5 w-10 align-middle" />
                </div>
                <Skeleton className="h-1 w-full rounded-full" />
              </div>
            ))}
          </div>
        </WatchSectionSkeleton>
        <WatchSectionSkeleton>
          <div className="flex flex-col gap-1.5">
            {BILLING_WIDTHS.map((width, index) => (
              <div key={index} className="flex items-center gap-2 text-sm">
                <Skeleton className={cn("inline-block h-3 align-middle", width)} />
                <span className="flex-1" />
                <Skeleton className="h-1.5 w-20 rounded-full" />
                <Skeleton className="h-2.5 w-16" />
              </div>
            ))}
          </div>
        </WatchSectionSkeleton>
      </div>
    </div>
  );
}

/** The activity feed: the rule down the left and a dot, a line and a time per event. */
export function ActivityTimelineSkeleton() {
  const t = useT();

  return (
    <div className="relative flex flex-col px-4 py-3" aria-busy aria-label={t("Loading activity")}>
      <div className="contents" aria-hidden>
        <span className="bg-border absolute top-5 bottom-5 left-[19px] w-px" />
        {ACTIVITY_WIDTHS.map((width, index) => (
          <div key={index} className="relative flex items-start gap-3 py-1.5">
            <Skeleton className="relative mt-1.5 size-[7px] shrink-0 rounded-full" />
            <div className="min-w-0 flex-1 text-sm leading-snug">
              <Skeleton className={cn("inline-block h-3 align-middle", width)} />
            </div>
            <Skeleton className="mt-1 h-2.5 w-10 shrink-0" />
          </div>
        ))}
      </div>
    </div>
  );
}

/** The whole panel, open on whichever tab the address names. */
export function ShipmentSidePanelSkeleton() {
  const t = useT();
  const [{ tab }] = useShipmentBoardUrl();

  return (
    <div
      className={SIDE_PANEL_FRAME_CLASS}
      aria-busy
      aria-label={t("Loading the brief and activity")}
    >
      <div className="contents" aria-hidden>
        <div className="border-border flex items-center gap-2 border-b px-3 py-2">
          <Skeleton className="h-9 flex-1 rounded-lg" />
          <Skeleton className="size-7" />
        </div>
        <div className="min-h-0 flex-1 overflow-hidden">
          {tab === "activity" ? (
            <ActivityTimelineSkeleton />
          ) : (
            <div className="flex flex-col gap-5 p-4">
              <ActionQueueSkeleton />
              <WatchlistSkeleton />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
