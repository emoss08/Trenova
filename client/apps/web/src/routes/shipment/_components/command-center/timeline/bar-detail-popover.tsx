import { ShipmentStatusBadge } from "@trenova/shared/components/status-badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent } from "@trenova/shared/components/ui/popover";
import { formatDurationFromSeconds, formatToUserTimezone } from "@trenova/shared/lib/date";
import {
  getDestinationLocation,
  getOriginLocation,
  getTotalMiles,
} from "@/lib/shipment-utils";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import {
  ArrowLeftRightIcon,
  CircleCheckIcon,
  CircleDashedIcon,
  CircleDotIcon,
  PencilIcon,
  TableIcon,
  TimerIcon,
} from "lucide-react";
import type { TimelineBar, TimelineStopMarker } from "./use-timeline-data";

function parseDecimal(value: unknown): number {
  if (typeof value === "number") return value;
  const parsed = Number(value ?? 0);
  return Number.isFinite(parsed) ? parsed : 0;
}

function formatTime(seconds: number) {
  return formatToUserTimezone(seconds, { showTimeZone: false, showSeconds: false });
}

function stopTimeDetail(stop: TimelineStopMarker): string {
  if (stop.actualDeparture) return `dep ${formatTime(stop.actualDeparture)}`;
  if (stop.actualArrival) return `arr ${formatTime(stop.actualArrival)}`;
  if (stop.scheduledStart) {
    return stop.scheduledEnd && stop.scheduledEnd !== stop.scheduledStart
      ? `${formatTime(stop.scheduledStart)}–${formatTime(stop.scheduledEnd)}`
      : formatTime(stop.scheduledStart);
  }
  return formatTime(stop.time);
}

type BarDetailPopoverProps = {
  bar: TimelineBar | null;
  anchor: HTMLElement | null;
  onOpenChange: (open: boolean) => void;
  onEdit: (bar: TimelineBar) => void;
  onViewInTable: (bar: TimelineBar) => void;
  onReassign: ((bar: TimelineBar) => void) | null;
};

export function BarDetailPopover({
  bar,
  anchor,
  onOpenChange,
  onEdit,
  onViewInTable,
  onReassign,
}: BarDetailPopoverProps) {
  if (!bar || !anchor) return null;

  const shipment = bar.shipment;
  const origin = getOriginLocation(shipment);
  const dest = getDestinationLocation(shipment);
  const revenue = parseDecimal(shipment.totalChargeAmount);
  const miles = getTotalMiles(shipment);
  const workerName = bar.assignment?.primaryWorker?.wholeName;
  const tractorCode = bar.assignment?.tractor?.code;

  return (
    <Popover open onOpenChange={onOpenChange}>
      <PopoverContent anchor={anchor} side="top" className="w-72 gap-0 p-0">
        <div className="flex items-start justify-between gap-2 border-b border-border px-3 py-2">
          <div className="flex min-w-0 flex-col">
            <span className="truncate font-table text-[12px] font-semibold tabular-nums">
              {shipment.proNumber ?? shipment.bol ?? "Shipment"}
            </span>
            <span className="truncate text-[10.5px] text-muted-foreground">
              {shipment.customer?.name ?? "No customer"}
            </span>
          </div>
          <ShipmentStatusBadge status={shipment.status} />
        </div>

        {bar.dwell && (
          <div
            className={cn(
              "flex items-center gap-1.5 border-b border-border px-3 py-1.5 text-[10.5px] font-semibold",
              bar.dwell.severity === "critical"
                ? "bg-destructive/10 text-destructive"
                : "bg-warning/10 text-warning",
            )}
          >
            <TimerIcon className="size-3 shrink-0" />
            Dwelling {formatDurationFromSeconds(bar.dwell.seconds)} at {bar.dwell.locationName}
            {bar.dwell.severity === "critical" && " · detention risk"}
          </div>
        )}

        <div className="flex flex-col gap-1.5 px-3 py-2">
          <div className="flex items-baseline justify-between gap-2">
            <span className="truncate font-table text-[11px] font-medium tabular-nums">
              {origin?.code ?? "—"} → {dest?.code ?? "—"}
            </span>
            <span className="shrink-0 font-table text-[10.5px] text-muted-foreground tabular-nums">
              {formatCurrency(revenue)} · {miles}mi
            </span>
          </div>
          <ol className="flex flex-col gap-1">
            {bar.stops.map((stop) => {
              const isDone = stop.status === "Completed";
              const isAtStop =
                !isDone && !!stop.actualArrival && !stop.actualDeparture && stop.status !== "Canceled";
              return (
                <li key={stop.id} className="flex items-center gap-1.5 text-[10.5px]">
                  {isDone ? (
                    <CircleCheckIcon className="size-3 shrink-0 text-success" />
                  ) : isAtStop ? (
                    <CircleDotIcon className="size-3 shrink-0 animate-pulse text-brand" />
                  ) : (
                    <CircleDashedIcon className="size-3 shrink-0 text-muted-foreground" />
                  )}
                  <span className={cn("truncate", isDone && "text-muted-foreground")}>
                    {stop.locationName}
                  </span>
                  <span className="ml-auto shrink-0 font-table text-[9.5px] text-muted-foreground tabular-nums">
                    {stopTimeDetail(stop)}
                  </span>
                </li>
              );
            })}
          </ol>
          <p className="font-table text-[10px] text-muted-foreground tabular-nums">
            {workerName ? `${workerName}${tractorCode ? ` · ${tractorCode}` : ""}` : "Unassigned"}
          </p>
        </div>

        <div className="flex items-center gap-1 border-t border-border px-2 py-1.5">
          <Button type="button" variant="ghost" size="xs" onClick={() => onEdit(bar)}>
            <PencilIcon className="size-3" />
            Open
          </Button>
          <Button type="button" variant="ghost" size="xs" onClick={() => onViewInTable(bar)}>
            <TableIcon className="size-3" />
            View in table
          </Button>
          {onReassign && (
            <Button
              type="button"
              variant="ghost"
              size="xs"
              className="ml-auto"
              onClick={() => onReassign(bar)}
            >
              <ArrowLeftRightIcon className="size-3" />
              {bar.assignment ? "Reassign" : "Assign"}
            </Button>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}
