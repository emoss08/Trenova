import { Avatar, AvatarFallback, AvatarImage } from "@trenova/shared/components/ui/avatar";
import { cn } from "@trenova/shared/lib/utils";
import { useDroppable } from "@dnd-kit/core";
import { ChevronDownIcon, ChevronRightIcon, InboxIcon } from "lucide-react";
import {
  COLLAPSED_BAR_HEIGHT_PX,
  RAIL_WIDTH_PX,
  rowHeightPx,
  type TimelineDensity,
} from "./constants";
import { TimelineBarItem } from "./timeline-bar";
import { getBarGeometry, type TimeRange } from "./time-scale";
import type { TimelineZoom } from "../url-state";
import {
  barMatchesFocus,
  UNASSIGNED_ROW_KEY,
  type TimelineBar,
  type TimelineFocus,
  type TimelineRow,
  type TimelineRowStats,
} from "./use-timeline-data";
import type { ShipmentEtaTone } from "@/lib/shipment-utils";

const STRIP_TONE_CLASS: Record<ShipmentEtaTone, string> = {
  ontime: "bg-brand/50 hover:bg-brand/70",
  watch: "bg-warning/60 hover:bg-warning/80",
  late: "bg-destructive/60 hover:bg-destructive/80",
  delivered: "bg-success/50 hover:bg-success/70",
  pending: "bg-muted-foreground/30 hover:bg-muted-foreground/50",
};

function workerInitials(name: string): string {
  const parts = name.split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return `${parts[0][0]}${parts[parts.length - 1][0]}`.toUpperCase();
}

function alertTitle(stats: TimelineRowStats): string {
  const parts: string[] = [];
  if (stats.late > 0) parts.push(`${stats.late} late`);
  if (stats.watch > 0) parts.push(`${stats.watch} at watch`);
  if (stats.dwelling > 0) parts.push(`${stats.dwelling} dwelling`);
  if (stats.overlaps > 0) parts.push(`${stats.overlaps} overlapping`);
  return parts.join(" · ");
}

type TimelineRowItemProps = {
  row: TimelineRow;
  range: TimeRange;
  zoom: TimelineZoom;
  density: TimelineDensity;
  collapsed: boolean;
  focus: TimelineFocus | null;
  canvasWidth: number;
  highlightId: string | null;
  draggable: boolean;
  droppable: boolean;
  onToggleCollapsed: (key: string) => void;
  onHoverChange: (shipmentId: string | null) => void;
  onSelectBar: (bar: TimelineBar, anchor: HTMLElement) => void;
};

export function TimelineRowItem({
  row,
  range,
  zoom,
  density,
  collapsed,
  focus,
  canvasWidth,
  highlightId,
  draggable,
  droppable,
  onToggleCollapsed,
  onHoverChange,
  onSelectBar,
}: TimelineRowItemProps) {
  const isUnassigned = row.key === UNASSIGNED_ROW_KEY;
  const { setNodeRef, isOver, active } = useDroppable({
    id: `row:${row.key}`,
    data: { row },
    disabled: !droppable,
  });

  const activeBar = active?.data.current?.bar as TimelineBar | undefined;
  const isValidTarget =
    !!activeBar &&
    (isUnassigned
      ? !!activeBar.assignment
      : activeBar.assignment?.primaryWorker?.id !== row.key);
  const showDropHint = isOver && isValidTarget;
  const height = rowHeightPx(row.laneCount, density, collapsed);

  return (
    <div
      ref={setNodeRef}
      className={cn(
        "flex border-b border-border/70 transition-colors",
        isUnassigned && "bg-warning/[4%]",
        showDropHint && "bg-brand/10",
      )}
      style={{ height }}
    >
      <div
        className={cn(
          "sticky left-0 z-30 flex shrink-0 items-center gap-1.5 border-r border-border bg-card pr-2.5 pl-1",
          isUnassigned && "bg-[color-mix(in_oklch,var(--warning)_5%,var(--card))]",
          showDropHint && "bg-[color-mix(in_oklch,var(--brand)_8%,var(--card))]",
        )}
        style={{ width: RAIL_WIDTH_PX }}
      >
        <button
          type="button"
          aria-expanded={!collapsed}
          aria-label={collapsed ? `Expand ${row.workerName}` : `Collapse ${row.workerName}`}
          onClick={() => onToggleCollapsed(row.key)}
          className="flex size-5 shrink-0 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          {collapsed ? (
            <ChevronRightIcon className="size-3.5" />
          ) : (
            <ChevronDownIcon className="size-3.5" />
          )}
        </button>
        {!collapsed &&
          (isUnassigned ? (
            <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-warning/15 text-warning">
              <InboxIcon className="size-3.5" />
            </span>
          ) : (
            <Avatar className={cn("shrink-0", density === "compact" ? "size-5" : "size-6")}>
              {row.workerProfilePicUrl && (
                <AvatarImage src={row.workerProfilePicUrl} alt={row.workerName} />
              )}
              <AvatarFallback className="text-[9px]">
                {workerInitials(row.workerName)}
              </AvatarFallback>
            </Avatar>
          ))}
        <div className="flex min-w-0 flex-col">
          <span className="flex min-w-0 items-center gap-1.5">
            <span
              className={cn(
                "truncate text-[11.5px] font-medium",
                isUnassigned && "text-warning",
              )}
            >
              {isUnassigned ? "Unassigned" : row.workerName}
            </span>
            {row.alert && (
              <span
                title={alertTitle(row.stats)}
                className={cn(
                  "size-1.5 shrink-0 rounded-full",
                  row.alert === "late" ? "animate-pulse bg-destructive" : "bg-warning",
                )}
              />
            )}
            {collapsed && (
              <span className="shrink-0 font-table text-[9.5px] text-muted-foreground tabular-nums">
                {row.bars.length} {row.bars.length === 1 ? "load" : "loads"}
              </span>
            )}
          </span>
          {!collapsed && (
            <span className="truncate font-table text-[9.5px] text-muted-foreground tabular-nums">
              {isUnassigned
                ? "Drop here to unassign"
                : row.equipmentCodes.length > 0
                  ? row.equipmentCodes.join(" · ")
                  : "No tractor"}
              {" · "}
              {row.bars.length} {row.bars.length === 1 ? "load" : "loads"}
            </span>
          )}
        </div>
      </div>
      <div className="relative shrink-0" style={{ width: canvasWidth }}>
        {collapsed
          ? row.bars.map((bar) => (
              <CollapsedBarStrip
                key={bar.moveId}
                bar={bar}
                range={range}
                zoom={zoom}
                rowHeight={height}
                dimmed={!!focus && !barMatchesFocus(bar, focus)}
                isHighlighted={!!highlightId && highlightId === bar.shipment.id}
                onHoverChange={onHoverChange}
                onSelect={onSelectBar}
              />
            ))
          : row.bars.map((bar) => (
              <TimelineBarItem
                key={bar.moveId}
                bar={bar}
                range={range}
                zoom={zoom}
                density={density}
                isHighlighted={!!highlightId && highlightId === bar.shipment.id}
                dimmed={!!focus && !barMatchesFocus(bar, focus)}
                draggable={draggable}
                onHoverChange={onHoverChange}
                onSelect={onSelectBar}
              />
            ))}
      </div>
    </div>
  );
}

/**
 * Collapsed rows keep the shape of the driver's day visible as thin tone
 * strips: cheap to render (no drag, no portal tooltip), still clickable for
 * the detail popover.
 */
function CollapsedBarStrip({
  bar,
  range,
  zoom,
  rowHeight,
  dimmed,
  isHighlighted,
  onHoverChange,
  onSelect,
}: {
  bar: TimelineBar;
  range: TimeRange;
  zoom: TimelineZoom;
  rowHeight: number;
  dimmed: boolean;
  isHighlighted: boolean;
  onHoverChange: (shipmentId: string | null) => void;
  onSelect: (bar: TimelineBar, anchor: HTMLElement) => void;
}) {
  const geometry = getBarGeometry(bar.start, bar.end, range, zoom);
  const proNumber = bar.shipment.proNumber ?? bar.shipment.bol ?? "Shipment";

  return (
    <button
      type="button"
      title={bar.dwell ? `${proNumber} · dwelling` : proNumber}
      aria-label={`Shipment ${proNumber}`}
      onClick={(event) => onSelect(bar, event.currentTarget as HTMLElement)}
      onMouseEnter={() => bar.shipment.id && onHoverChange(bar.shipment.id)}
      onMouseLeave={() => onHoverChange(null)}
      className={cn(
        "absolute cursor-pointer rounded-sm transition-[background-color,opacity] outline-none focus-visible:ring-2 focus-visible:ring-brand",
        STRIP_TONE_CLASS[bar.tone],
        bar.isCanceled && "opacity-40",
        dimmed && "opacity-20",
        isHighlighted && "ring-1 ring-foreground/40",
      )}
      style={{
        left: geometry.left,
        width: geometry.width,
        height: COLLAPSED_BAR_HEIGHT_PX,
        top: (rowHeight - COLLAPSED_BAR_HEIGHT_PX) / 2,
      }}
    >
      {bar.dwell && (
        <span
          aria-hidden
          className={cn(
            "absolute -top-0.5 -right-0.5 size-1.5 animate-pulse rounded-full",
            bar.dwell.severity === "critical" ? "bg-destructive" : "bg-warning",
          )}
        />
      )}
    </button>
  );
}
