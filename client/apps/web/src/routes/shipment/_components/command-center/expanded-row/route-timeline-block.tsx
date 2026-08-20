import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import type { Stop, StopType } from "@trenova/shared/types/shipment";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";

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

function formatStopTime(timestamp: number | null | undefined): string {
  return formatUnixInUserTimezone(
    timestamp,
    { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit", hour12: false },
    "—",
  );
}

function StopDot({ state }: { state: StopState }) {
  if (state === "current") {
    return (
      <span
        aria-hidden
        className="bg-brand absolute top-0.75 -left-3.25 inline-block size-2.25 rounded-full"
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
        className="bg-success absolute top-0.75 -left-3.25 inline-block size-2.25 rounded-full"
        style={{ border: "1.5px solid var(--card)" }}
      />
    );
  }
  return (
    <span
      aria-hidden
      className="bg-muted absolute top-0.75 -left-3.25 inline-block size-2.25 rounded-full"
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
        <p className="text-muted-foreground text-[11px]">No stops on this shipment.</p>
      ) : (
        <div className="relative pl-4">
          <div
            aria-hidden
            className="bg-border absolute top-2 bottom-2 left-1.25"
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
                  <span className="font-table text-muted-foreground text-[10px] tabular-nums">
                    {time}
                  </span>
                  <div className="flex min-w-0 flex-col gap-0.5">
                    <div className="flex min-w-0 items-baseline gap-2">
                      <span
                        className={`font-table shrink-0 text-[9.5px] font-semibold tracking-wider ${
                          state === "current" ? "text-brand" : "text-muted-foreground"
                        }`}
                      >
                        {kind}
                      </span>
                      <span className="text-foreground truncate font-medium">{loc}</span>
                    </div>
                    <span className="font-table text-muted-foreground truncate text-[10.5px] tabular-nums">
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
