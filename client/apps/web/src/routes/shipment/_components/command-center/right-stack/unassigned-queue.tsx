import { Spinner } from "@/components/ui/spinner";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { formatToUserTimezone } from "@/lib/date";
import {
  getDestinationLocation,
  getOriginLocation,
  getOriginStop,
  getTotalMiles,
} from "@/lib/shipment-utils";
import { cn, formatCurrency } from "@/lib/utils";
import type { Shipment } from "@/types/shipment";
import { GripVerticalIcon } from "lucide-react";
import { useEffect, useMemo, useRef } from "react";
import { useCommandCenterUrl } from "../url-state";
import { ModuleCard } from "./module-card";
import { useUnassignedShipments } from "./use-right-stack-data";

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

export function UnassignedQueue({ enabled = true }: { enabled?: boolean }) {
  const { data, isLoading, hasNextPage, isFetchingNextPage, fetchNextPage } =
    useUnassignedShipments(undefined, enabled);
  const [, setUrl] = useCommandCenterUrl();

  const shipments = useMemo(
    () => (data?.pages.flatMap((page) => page.results) ?? []) as Shipment[],
    [data?.pages],
  );
  const totalCount = data?.pages[0]?.count;

  const pendingRevenue = shipments.reduce(
    (total, s) => total + parseDecimal(s.totalChargeAmount as unknown as string),
    0,
  );

  const observerTarget = useRef<HTMLDivElement>(null);

  useEffect(() => {
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
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  return (
    <ModuleCard
      id="unassigned"
      title="Unassigned"
      count={totalCount}
      countTone="warning"
      rightSlot={
        <span className="hidden font-table text-[9.5px] text-muted-foreground tabular-nums sm:inline">
          {formatCurrency(pendingRevenue)} waiting
        </span>
      }
    >
      <div className="flex flex-col gap-1.5">
        {isLoading && (
          <div className="flex items-center gap-2 text-[10.5px] text-muted-foreground">
            <Spinner className="size-3" /> Loading…
          </div>
        )}
        {!isLoading && shipments.length === 0 && (
          <p className="py-3 text-center text-[10.5px] text-muted-foreground">
            All loads are assigned ✓
          </p>
        )}
        {shipments.map((s) => {
          const origin = getOriginLocation(s)?.code ?? "—";
          const dest = getDestinationLocation(s)?.code ?? "—";
          const revenue = parseDecimal(s.totalChargeAmount as unknown as string);
          const miles = getTotalMiles(s);
          const priority = priorityFor(s);

          return (
            <button
              key={s.id}
              type="button"
              onClick={() => s.id && setUrl({ expanded: s.id })}
              className="flex flex-col gap-1 rounded border border-border bg-muted/30 px-2 py-1.5 text-left transition-colors hover:border-foreground/20 hover:bg-muted/60"
            >
              <div className="flex items-center justify-between gap-1.5">
                <div className="flex min-w-0 items-center gap-1">
                  <GripVerticalIcon className="size-2.5 shrink-0 text-muted-foreground" />
                  <span className="truncate font-table text-[10.5px] font-semibold tabular-nums">
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
              <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground">
                <span className="truncate">{s.customer?.name ?? "No Customer Found"}</span>
              </div>
              <div className="flex items-baseline justify-between gap-2 font-table tabular-nums">
                <span className="truncate text-[9.5px] text-muted-foreground">
                  pickup {pickupDisplay(s)}
                </span>
                <span className="text-[10.5px] font-semibold">
                  {formatCurrency(revenue)}{" "}
                  <span className="text-[9.5px] font-normal text-muted-foreground">
                    · {miles}mi
                  </span>
                </span>
              </div>
            </button>
          );
        })}
        {isFetchingNextPage && (
          <div className="flex items-center justify-center py-2">
            <TextShimmer className="font-mono text-[10px]" duration={1}>
              Loading more…
            </TextShimmer>
          </div>
        )}
        <div ref={observerTarget} aria-hidden className="h-px w-full" />
      </div>
    </ModuleCard>
  );
}
