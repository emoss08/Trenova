import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatDurationFromSeconds, formatToUserTimezone } from "@trenova/shared/lib/date";
import { getDestinationLocation, getOriginLocation, type ShipmentEtaTone } from "@/lib/shipment-utils";
import { cn } from "@trenova/shared/lib/utils";
import { useDraggable } from "@dnd-kit/core";
import { ChevronLeftIcon, ChevronRightIcon, TimerIcon } from "lucide-react";
import { DENSITY_CONFIGS, type TimelineDensity } from "./constants";
import { getBarGeometry, type TimeRange } from "./time-scale";
import type { TimelineZoom } from "../url-state";
import type { TimelineBar, TimelineStopMarker } from "./use-timeline-data";

const BAR_TONE_CLASS: Record<ShipmentEtaTone, string> = {
  ontime: "border-brand/45 bg-brand/12 hover:bg-brand/20",
  watch: "border-warning/55 bg-warning/15 hover:bg-warning/25",
  late: "border-destructive/55 bg-destructive/12 hover:bg-destructive/20",
  delivered: "border-success/45 bg-success/10 hover:bg-success/20",
  pending: "border-border bg-muted/70 hover:bg-muted",
};

const BAR_PROGRESS_CLASS: Record<ShipmentEtaTone, string> = {
  ontime: "bg-brand",
  watch: "bg-warning",
  late: "bg-destructive",
  delivered: "bg-success",
  pending: "bg-muted-foreground/50",
};

const STOP_LABEL: Record<TimelineBar["stops"][number]["type"], string> = {
  Pickup: "Pickup",
  SplitPickup: "Split pickup",
  Delivery: "Delivery",
  SplitDelivery: "Split delivery",
};

function formatTime(seconds: number) {
  return formatToUserTimezone(seconds, { showTimeZone: false, showSeconds: false });
}

function stopTimesLabel(stop: TimelineStopMarker): string {
  const parts: string[] = [];
  if (stop.scheduledStart) {
    parts.push(
      stop.scheduledEnd && stop.scheduledEnd !== stop.scheduledStart
        ? `win ${formatTime(stop.scheduledStart)}–${formatTime(stop.scheduledEnd)}`
        : formatTime(stop.scheduledStart),
    );
  }
  if (stop.actualArrival) parts.push(`arr ${formatTime(stop.actualArrival)}`);
  if (stop.actualDeparture) parts.push(`dep ${formatTime(stop.actualDeparture)}`);
  return parts.join(" · ");
}

type TimelineBarItemProps = {
  bar: TimelineBar;
  range: TimeRange;
  zoom: TimelineZoom;
  density: TimelineDensity;
  isHighlighted: boolean;
  dimmed: boolean;
  draggable: boolean;
  onHoverChange: (shipmentId: string | null) => void;
  onSelect: (bar: TimelineBar, anchor: HTMLElement) => void;
};

export function TimelineBarItem({
  bar,
  range,
  zoom,
  density,
  isHighlighted,
  dimmed,
  draggable,
  onHoverChange,
  onSelect,
}: TimelineBarItemProps) {
  const geometry = getBarGeometry(bar.start, bar.end, range, zoom);
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `bar:${bar.moveId}`,
    data: { bar },
    disabled: !draggable,
  });

  const { laneHeightPx, barHeightPx, rowPaddingPx } = DENSITY_CONFIGS[density];
  const shipment = bar.shipment;
  const originCode = getOriginLocation(shipment)?.code ?? "—";
  const destCode = getDestinationLocation(shipment)?.code ?? "—";
  const completedStops = bar.stops.filter((s) => s.status === "Completed").length;
  const progress = bar.stops.length > 0 ? completedStops / bar.stops.length : 0;
  const showLabel = geometry.width >= 72;
  const showDwellBadge = !!bar.dwell && geometry.width >= 40;
  const barSpanSeconds = Math.max(bar.end - bar.start, 1);

  return (
    <Tooltip delay={350}>
      <TooltipTrigger
        ref={setNodeRef}
        {...attributes}
        {...listeners}
        onClick={(event) => onSelect(bar, event.currentTarget as HTMLElement)}
        onMouseEnter={() => shipment.id && onHoverChange(shipment.id)}
        onMouseLeave={() => onHoverChange(null)}
        aria-label={`Shipment ${shipment.proNumber ?? ""}, ${originCode} to ${destCode}`}
        className={cn(
          "group/bar absolute flex cursor-pointer items-center overflow-hidden rounded border px-1.5 text-left transition-[background-color,box-shadow,opacity] outline-none focus-visible:ring-2 focus-visible:ring-brand",
          BAR_TONE_CLASS[bar.tone],
          bar.isCanceled && "border-dashed opacity-60",
          isHighlighted && "shadow-md ring-1 ring-foreground/25",
          dimmed && "opacity-25",
          isDragging && "opacity-40",
          draggable && "cursor-grab active:cursor-grabbing",
        )}
        style={{
          left: geometry.left,
          width: geometry.width,
          height: barHeightPx,
          top: rowPaddingPx / 2 + bar.laneIndex * laneHeightPx + (laneHeightPx - barHeightPx) / 2,
        }}
      >
        {geometry.clippedStart && (
          <ChevronLeftIcon className="size-3 shrink-0 text-muted-foreground" aria-hidden />
        )}
        {showLabel && (
          <span
            className={cn(
              "min-w-0 truncate font-table text-[10px] font-semibold tabular-nums",
              bar.isCanceled && "line-through",
            )}
          >
            {shipment.proNumber ?? shipment.bol ?? "—"}
            <span className="ml-1.5 font-normal text-muted-foreground">
              {originCode} → {destCode}
            </span>
          </span>
        )}
        {showDwellBadge && bar.dwell && (
          <span
            aria-hidden
            className={cn(
              "ml-auto flex shrink-0 items-center gap-0.5 rounded-sm px-1 py-px font-table text-[8.5px] font-semibold tabular-nums",
              bar.dwell.severity === "critical"
                ? "bg-destructive/90 text-white"
                : "bg-warning/90 text-white",
            )}
          >
            <TimerIcon className="size-2.5 animate-pulse" />
            {geometry.width >= 96 && formatDurationFromSeconds(bar.dwell.seconds)}
          </span>
        )}
        {geometry.clippedEnd && (
          <ChevronRightIcon
            className={cn("size-3 shrink-0 text-muted-foreground", !showDwellBadge && "ml-auto")}
            aria-hidden
          />
        )}
        {bar.stops.map((stop) => {
          const offset = ((Math.min(Math.max(stop.time, bar.start), bar.end) - bar.start) /
            barSpanSeconds) *
            100;
          return (
            <span
              key={stop.id}
              aria-hidden
              className={cn(
                "absolute bottom-0.5 size-1.5 -translate-x-1/2 rounded-full border border-background",
                stop.status === "Completed" ? BAR_PROGRESS_CLASS[bar.tone] : "bg-muted-foreground/40",
              )}
              style={{ left: `${offset}%` }}
            />
          );
        })}
        {progress > 0 && !bar.isCanceled && (
          <span
            aria-hidden
            className={cn("absolute inset-x-0 top-0 h-0.5 origin-left", BAR_PROGRESS_CLASS[bar.tone])}
            style={{ transform: `scaleX(${progress})` }}
          />
        )}
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-72">
        <div className="flex flex-col gap-1 py-0.5">
          <p className="font-table text-[11px] font-semibold tabular-nums">
            {shipment.proNumber ?? "—"}
            <span className="ml-1.5 font-normal opacity-80">
              {originCode} → {destCode}
            </span>
          </p>
          <p className="text-[10.5px] opacity-80">{shipment.customer?.name ?? "No customer"}</p>
          {bar.dwell && (
            <p
              className={cn(
                "flex items-center gap-1 text-[10.5px] font-semibold",
                bar.dwell.severity === "critical" ? "text-destructive" : "text-warning",
              )}
            >
              <TimerIcon className="size-3" />
              Dwelling {formatDurationFromSeconds(bar.dwell.seconds)} at {bar.dwell.locationCode}
            </p>
          )}
          {bar.hasOverlap && (
            <p className="text-[10.5px] font-semibold text-warning">
              Overlaps another load on this driver
            </p>
          )}
          <div className="mt-0.5 flex flex-col gap-0.5">
            {bar.stops.map((stop) => (
              <p key={stop.id} className="font-table text-[10px] tabular-nums opacity-80">
                {STOP_LABEL[stop.type]} · {stop.locationCode} · {stopTimesLabel(stop)}
              </p>
            ))}
          </div>
        </div>
      </TooltipContent>
    </Tooltip>
  );
}
