import { Spinner } from "@trenova/shared/components/ui/spinner";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import {
  getDestinationLocation,
  getOriginLocation,
  getOriginStop,
  getTotalMiles,
} from "@/lib/shipment-utils";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { Shipment } from "@trenova/shared/types/shipment";
import { GripVerticalIcon } from "lucide-react";
import { useEffect, useMemo, useRef } from "react";
import { useUnassignedShipments } from "./use-work-queues";

function parseDecimal(value: string | number | null | undefined): number {
  if (value === null || value === undefined) return 0;
  if (typeof value === "number") return value;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function pickupDisplay(shipment: Shipment): string {
  const stop = getOriginStop(shipment);
  if (!stop?.scheduledWindowStart) return "—";
  return formatToUserTimezone(stop.scheduledWindowStart, {
    showTimeZone: false,
    showSeconds: false,
  });
}

function priorityFor(s: Shipment): {
  label: "HOT" | "MED" | "LOW";
  tone: "danger" | "warning" | "muted";
} {
  if (s.status === "New") return { label: "HOT", tone: "danger" };
  if (s.status === "PartiallyAssigned") return { label: "MED", tone: "warning" };
  return { label: "LOW", tone: "muted" };
}

const PILL_TONE: Record<"danger" | "warning" | "muted", string> = {
  danger: "bg-destructive/12 text-destructive",
  warning: "bg-warning/15 text-warning",
  muted: "bg-muted text-muted-foreground",
};

export type UnassignedQueueSummary = {
  totalCount: number | undefined;
  pendingRevenue: number;
};

export type UnassignedQueueListProps = {
  enabled?: boolean;
  /** Caps the visible rows; omit for the infinite-scrolling queue. */
  limit?: number;
  onSelect: (shipmentId: string) => void;
  /**
   * Reports the count and waiting revenue upward so whichever chrome wraps this
   * list can put them in its own header.
   */
  onSummary?: (summary: UnassignedQueueSummary) => void;
};

/**
 * The unassigned-load queue, without chrome. The command center wraps it in a
 * right-stack module card and the home screen wraps it in a widget; both render
 * the same rows from the same query.
 */
export function UnassignedQueueList({
  enabled = true,
  limit,
  onSelect,
  onSummary,
}: UnassignedQueueListProps) {
  const { data, isLoading, hasNextPage, isFetchingNextPage, fetchNextPage } =
    useUnassignedShipments(undefined, enabled);

  const shipments = useMemo(
    () => (data?.pages.flatMap((page) => page.results) ?? []) as Shipment[],
    [data?.pages],
  );
  const totalCount = data?.pages[0]?.count;

  const pendingRevenue = useMemo(
    () =>
      shipments.reduce(
        (total, s) => total + parseDecimal(s.totalChargeAmount as unknown as string),
        0,
      ),
    [shipments],
  );

  useEffect(() => {
    onSummary?.({ totalCount, pendingRevenue });
  }, [totalCount, pendingRevenue, onSummary]);

  const visible = limit ? shipments.slice(0, limit) : shipments;
  const observerTarget = useRef<HTMLDivElement>(null);

  // A capped list never pages: the widget that capped it wants a snapshot, not
  // a scroller that quietly loads the whole queue behind a fixed height.
  const paging = !limit;

  useEffect(() => {
    if (!paging) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );

    const currentTarget = observerTarget.current;
    if (currentTarget) {
      observer.observe(currentTarget);
    }

    return () => {
      if (currentTarget) {
        observer.unobserve(currentTarget);
      }
    };
  }, [paging, hasNextPage, isFetchingNextPage, fetchNextPage]);

  return (
    <div className="flex flex-col gap-1.5">
      {isLoading && (
        <div className="text-muted-foreground flex items-center gap-2 text-[10.5px]">
          <Spinner className="size-3" /> Loading…
        </div>
      )}
      {!isLoading && visible.length === 0 && (
        <p className="text-muted-foreground py-3 text-center text-[10.5px]">
          All loads are assigned ✓
        </p>
      )}
      {visible.map((s) => {
        const origin = getOriginLocation(s)?.code ?? "—";
        const dest = getDestinationLocation(s)?.code ?? "—";
        const revenue = parseDecimal(s.totalChargeAmount as unknown as string);
        const miles = getTotalMiles(s);
        const priority = priorityFor(s);

        return (
          <button
            key={s.id}
            type="button"
            onClick={() => s.id && onSelect(s.id)}
            className="border-border bg-muted/30 hover:border-foreground/20 hover:bg-muted/60 flex flex-col gap-1 rounded border px-2 py-1.5 text-left transition-colors"
          >
            <div className="flex items-center justify-between gap-1.5">
              <div className="flex min-w-0 items-center gap-1">
                <GripVerticalIcon className="text-muted-foreground size-2.5 shrink-0" />
                <span className="font-table truncate text-[10.5px] font-semibold tabular-nums">
                  {origin} → {dest}
                </span>
              </div>
              <span
                className={cn(
                  "shrink-0 rounded px-1 py-px text-[8.5px] font-bold tracking-wide uppercase",
                  PILL_TONE[priority.tone],
                )}
              >
                {priority.label}
              </span>
            </div>
            <div className="text-muted-foreground flex items-center justify-between gap-2 text-[10px]">
              <span className="truncate">{s.customer?.name ?? "No Customer Found"}</span>
            </div>
            <div className="font-table flex items-baseline justify-between gap-2 tabular-nums">
              <span className="text-muted-foreground truncate text-[9.5px]">
                pickup {pickupDisplay(s)}
              </span>
              <span className="text-[10.5px] font-semibold">
                {formatCurrency(revenue)}{" "}
                <span className="text-muted-foreground text-[9.5px] font-normal">· {miles}mi</span>
              </span>
            </div>
          </button>
        );
      })}
      {paging && isFetchingNextPage && (
        <div className="flex items-center justify-center py-2">
          <TextShimmer className="font-mono text-[10px]" duration={1}>
            Loading more…
          </TextShimmer>
        </div>
      )}
      {paging && <div ref={observerTarget} aria-hidden className="h-px w-full" />}
    </div>
  );
}
