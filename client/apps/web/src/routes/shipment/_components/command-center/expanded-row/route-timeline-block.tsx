import { ScrollArea } from "@/components/ui/scroll-area";
import type { Stop, StopType } from "@/types/shipment";

const STOP_KIND: Record<StopType, string> = {
  Pickup: "PICKUP",
  Delivery: "DELIVERY",
  SplitPickup: "SPLIT-PU",
  SplitDelivery: "SPLIT-DL",
};

type StopState = "done" | "current" | "upcoming";

function stopState(stop: Stop): StopState {
  if (stop.status === "Completed") return "done";
  if (stop.status === "InTransit") return "current";
  return "upcoming";
}

const COMPACT_STOP_FORMAT = new Intl.DateTimeFormat("en-US", {
  month: "short",
  day: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

function formatStopTime(timestamp: number | null | undefined): string {
  if (!timestamp) return "—";
  return COMPACT_STOP_FORMAT.format(new Date(timestamp * 1000));
}

function StopDot({ state }: { state: StopState }) {
  if (state === "current") {
    return (
      <span
        aria-hidden
        className="absolute top-0.75 -left-3.25 inline-block size-2.25 rounded-full bg-brand"
        style={{
          boxShadow: "0 0 0 3px color-mix(in oklch, var(--brand) 18%, transparent)",
          border: "1.5px solid var(--card)",
        }}
      />
    );
  }
  if (state === "done") {
    return (
      <span
        aria-hidden
        className="absolute top-0.75 -left-3.25 inline-block size-2.25 rounded-full bg-success"
        style={{ border: "1.5px solid var(--card)" }}
      />
    );
  }
  return (
    <span
      aria-hidden
      className="absolute top-0.75 -left-3.25 inline-block size-2.25 rounded-full bg-muted"
      style={{ border: "1.5px dashed var(--border)" }}
    />
  );
}

function stopNote(stop: Stop): string {
  if (stop.actualArrival) {
    const arrived = formatStopTime(stop.actualArrival);
    if (stop.actualDeparture) {
      return `${arrived} → departed ${formatStopTime(stop.actualDeparture)}`;
    }
    return `Arrived ${arrived}`;
  }
  if (stop.scheduledWindowEnd && stop.scheduledWindowStart) {
    return `Window ${formatStopTime(stop.scheduledWindowStart)} – ${formatStopTime(stop.scheduledWindowEnd)}`;
  }
  if (stop.scheduledWindowStart) {
    return `Scheduled ${formatStopTime(stop.scheduledWindowStart)}`;
  }
  return "—";
}

export function RouteTimelineBlock({ stops }: { stops: Stop[] }) {
  return (
    <ScrollArea className="h-70" viewportClassName="pr-2">
      {stops.length === 0 ? (
        <p className="text-[11px] text-muted-foreground">No stops on this shipment.</p>
      ) : (
        <div className="relative pl-4">
          <div
            aria-hidden
            className="absolute top-2 bottom-2 left-1.25 bg-border"
            style={{ width: "1.5px" }}
          />
          {stops.map((stop, i) => {
            const state = stopState(stop);
            const time = formatStopTime(stop.actualArrival ?? stop.scheduledWindowStart);
            const kind = STOP_KIND[stop.type] ?? stop.type.toUpperCase();
            const loc = stop.location?.name ?? "—";
            return (
              <div
                key={stop.id ?? `${stop.locationId}-${i}`}
                className="relative pb-2 text-[11px] last:pb-0"
              >
                <StopDot state={state} />
                <div className="grid grid-cols-[88px_1fr] gap-x-2 leading-tight">
                  <span className="font-table text-[10px] text-muted-foreground tabular-nums">
                    {time}
                  </span>
                  <div className="flex min-w-0 flex-col gap-0.5">
                    <div className="flex min-w-0 items-baseline gap-2">
                      <span
                        className={`shrink-0 font-table text-[9.5px] font-semibold tracking-wider ${
                          state === "current" ? "text-brand" : "text-muted-foreground"
                        }`}
                      >
                        {kind}
                      </span>
                      <span className="truncate font-medium text-foreground">{loc}</span>
                    </div>
                    <span className="truncate font-table text-[10.5px] text-muted-foreground tabular-nums">
                      {stopNote(stop)}
                    </span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </ScrollArea>
  );
}

export default RouteTimelineBlock;
