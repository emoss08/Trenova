import { AmountDisplay } from "@/components/accounting/amount-display";
import { Badge } from "@/components/ui/badge";
import { formatRange } from "@/lib/date";
import { recordMyStopAction, type PortalLoad, type PortalStop } from "@/lib/graphql/driver-portal";
import { cn } from "@/lib/utils";
import type { PortalStopAction } from "@trenova/graphql/generated/graphql";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowRightIcon } from "lucide-react";
import { m } from "motion/react";
import { Link } from "react-router";
import { toast } from "sonner";
import { LoadStatusBadge } from "./portal-badges";
import {
  destinationStop,
  directionsUrl,
  formatMiles,
  formatPieces,
  formatWeight,
  originStop,
  stopPlace,
} from "./use-loads";

export function stopTimeLabel(stop: PortalStop): string {
  if (stop.actualDeparture) {
    return `Departed ${formatRange(stop.actualDeparture, stop.actualDeparture)}`;
  }
  if (stop.actualArrival) {
    return `Arrived ${formatRange(stop.actualArrival, stop.actualArrival)}`;
  }
  if (stop.scheduledWindowStart) {
    const day = formatRange(stop.scheduledWindowStart, stop.scheduledWindowStart);
    const time = new Intl.DateTimeFormat(undefined, {
      hour: "numeric",
      minute: "2-digit",
    }).format(new Date(stop.scheduledWindowStart * 1000));
    return `${day}, ${time}`;
  }
  return "Not scheduled";
}

export const stopTypeLabels: Record<string, string> = {
  Pickup: "Pickup",
  Delivery: "Delivery",
  SplitPickup: "Split pickup",
  SplitDelivery: "Split delivery",
};

export function LoadPayChip({ load }: { load: PortalLoad }) {
  if (load.payGrossMinor == null) {
    return null;
  }
  if (load.payOnHold) {
    return (
      <Badge variant="warning">
        <AmountDisplay value={load.payGrossMinor} /> held
      </Badge>
    );
  }
  return (
    <Badge variant={load.payStatus === "Settled" ? "active" : "teal"}>
      <AmountDisplay value={load.payGrossMinor} />
      {load.payStatus === "Settled" ? " paid" : ""}
    </Badge>
  );
}

export function LoadCard({ load, index = 0 }: { load: PortalLoad; index?: number }) {
  const origin = originStop(load);
  const destination = destinationStop(load);
  const meta = [
    formatMiles(load.distanceMiles),
    formatWeight(load.weight),
    formatPieces(load.pieces),
    load.tractorCode ? `Truck ${load.tractorCode}` : null,
    load.isPrimary ? null : "Co-driver",
  ].filter(Boolean);

  return (
    <m.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.22, ease: "easeOut", delay: Math.min(index * 0.04, 0.24) }}
    >
      <Link to={`/dash/loads/${load.assignmentId}`} className="block">
        <m.div
          whileTap={{ scale: 0.98 }}
          className="rounded-2xl border border-border bg-card p-4 transition-colors hover:border-foreground/20"
        >
          <div className="flex items-center justify-between gap-2">
            <p className="truncate font-mono text-sm font-semibold">
              {load.proNumber || "Pending pro #"}
            </p>
            <div className="flex shrink-0 items-center gap-1.5">
              <LoadPayChip load={load} />
              <LoadStatusBadge status={load.status} />
            </div>
          </div>

          <div className="mt-3 flex items-center gap-2">
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium">{stopPlace(origin)}</p>
              {origin ? (
                <p className="truncate text-xs text-muted-foreground">{stopTimeLabel(origin)}</p>
              ) : null}
            </div>
            <div className="flex shrink-0 flex-col items-center px-1">
              <ArrowRightIcon className="size-4 text-muted-foreground" />
              {load.stops.length > 2 ? (
                <span className="text-2xs text-muted-foreground">
                  +{load.stops.length - 2} stop{load.stops.length - 2 === 1 ? "" : "s"}
                </span>
              ) : null}
            </div>
            <div className="min-w-0 flex-1 text-right">
              <p className="truncate text-sm font-medium">{stopPlace(destination)}</p>
              {destination ? (
                <p className="truncate text-xs text-muted-foreground">
                  {stopTimeLabel(destination)}
                </p>
              ) : null}
            </div>
          </div>

          {meta.length > 0 ? (
            <p className="mt-3 border-t border-border pt-2.5 text-xs text-muted-foreground">
              {meta.join(" · ")}
            </p>
          ) : null}
        </m.div>
      </Link>
    </m.div>
  );
}

export function StopTimeline({
  stops,
  showDirections = false,
  moveId,
}: {
  stops: PortalStop[];
  showDirections?: boolean;
  moveId?: string;
}) {
  const queryClient = useQueryClient();
  const checkIn = useMutation({
    mutationFn: ({ stopId, action }: { stopId: string; action: PortalStopAction }) =>
      recordMyStopAction({ moveId: moveId ?? "", stopId, action }),
    onSuccess: async (_, variables) => {
      toast.success(variables.action === "Arrive" ? "Arrival recorded" : "Departure recorded");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["dash-loads"] }),
        queryClient.invalidateQueries({ queryKey: ["dash-period-summary"] }),
        queryClient.invalidateQueries({ queryKey: ["dash-recent-pay-events"] }),
      ]);
    },
    onError: (error: Error) => toast.error(error.message || "We couldn't record that. Try again."),
  });

  const activeStops = stops.filter((stop) => stop.status !== "Canceled");
  const actionableStopId = moveId
    ? activeStops.find((stop) => !stop.actualDeparture)?.id
    : undefined;

  return (
    <ol className="flex flex-col">
      {stops.map((stop, index) => {
        const isDone = Boolean(stop.actualDeparture);
        const isCurrent = Boolean(stop.actualArrival) && !isDone;
        const isLast = index === stops.length - 1;
        const isActionable = stop.id === actionableStopId;
        const action: PortalStopAction = stop.actualArrival ? "Depart" : "Arrive";
        return (
          <li key={stop.id} className="relative flex gap-3 pb-4 last:pb-0">
            {!isLast ? (
              <span
                className={cn(
                  "absolute top-4 left-[5px] h-full border-l border-border",
                  isDone && "border-green-600/60",
                )}
              />
            ) : null}
            <span
              className={cn(
                "z-10 mt-1.5 size-3 shrink-0 rounded-full border-2 border-muted-foreground/50 bg-background",
                isDone && "border-green-600 bg-green-600",
                isCurrent && "border-blue-600 bg-blue-600",
              )}
            />
            <div className="min-w-0 flex-1">
              <p className="text-xs font-medium">
                {stopTypeLabels[stop.type] ?? stop.type}
                <span className="font-normal text-muted-foreground"> · {stopTimeLabel(stop)}</span>
              </p>
              <p className="truncate text-sm font-medium">
                {stop.locationName || stop.addressLine}
              </p>
              {stop.locationName && stop.addressLine ? (
                <p className="truncate text-xs text-muted-foreground">{stop.addressLine}</p>
              ) : null}
            </div>
            <div className="flex shrink-0 flex-col items-end gap-1.5">
              {isActionable ? (
                <button
                  type="button"
                  disabled={checkIn.isPending}
                  onClick={() => checkIn.mutate({ stopId: stop.id, action })}
                  className={cn(
                    "mt-1 rounded-full px-2.5 py-1 text-xs font-semibold transition-colors disabled:opacity-60",
                    action === "Arrive"
                      ? "bg-primary text-primary-foreground"
                      : "bg-green-600 text-white",
                  )}
                >
                  {checkIn.isPending ? "Saving..." : action === "Arrive" ? "Arrive" : "Depart"}
                </button>
              ) : null}
              {showDirections && (stop.addressLine || stop.locationName) && !isDone ? (
                <a
                  href={directionsUrl(stop)}
                  target="_blank"
                  rel="noreferrer"
                  onClick={(event) => event.stopPropagation()}
                  className="mt-1 shrink-0 rounded-full border border-border px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-accent"
                >
                  Directions
                </a>
              ) : null}
            </div>
          </li>
        );
      })}
    </ol>
  );
}
