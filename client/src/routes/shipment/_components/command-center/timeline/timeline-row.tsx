import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { cn } from "@/lib/utils";
import { useDroppable } from "@dnd-kit/core";
import { InboxIcon } from "lucide-react";
import { RAIL_WIDTH_PX, rowHeightPx } from "./constants";
import { TimelineBarItem } from "./timeline-bar";
import type { TimeRange } from "./time-scale";
import type { TimelineZoom } from "../url-state";
import { UNASSIGNED_ROW_KEY, type TimelineBar, type TimelineRow } from "./use-timeline-data";

function workerInitials(name: string): string {
  const parts = name.split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return `${parts[0][0]}${parts[parts.length - 1][0]}`.toUpperCase();
}

type TimelineRowItemProps = {
  row: TimelineRow;
  range: TimeRange;
  zoom: TimelineZoom;
  canvasWidth: number;
  highlightId: string | null;
  draggable: boolean;
  droppable: boolean;
  onHoverChange: (shipmentId: string | null) => void;
  onSelectBar: (bar: TimelineBar, anchor: HTMLElement) => void;
};

export function TimelineRowItem({
  row,
  range,
  zoom,
  canvasWidth,
  highlightId,
  draggable,
  droppable,
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

  return (
    <div
      ref={setNodeRef}
      className={cn(
        "flex border-b border-border/70 transition-colors",
        isUnassigned && "bg-warning/[4%]",
        showDropHint && "bg-brand/10",
      )}
      style={{ height: rowHeightPx(row.laneCount) }}
    >
      <div
        className={cn(
          "sticky left-0 z-30 flex shrink-0 items-center gap-2 border-r border-border bg-card px-2.5",
          isUnassigned && "bg-[color-mix(in_oklch,var(--warning)_5%,var(--card))]",
          showDropHint && "bg-[color-mix(in_oklch,var(--brand)_8%,var(--card))]",
        )}
        style={{ width: RAIL_WIDTH_PX }}
      >
        {isUnassigned ? (
          <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-warning/15 text-warning">
            <InboxIcon className="size-3.5" />
          </span>
        ) : (
          <Avatar className="size-6 shrink-0">
            {row.workerProfilePicUrl && (
              <AvatarImage src={row.workerProfilePicUrl} alt={row.workerName} />
            )}
            <AvatarFallback className="text-[9px]">{workerInitials(row.workerName)}</AvatarFallback>
          </Avatar>
        )}
        <div className="flex min-w-0 flex-col">
          <span
            className={cn(
              "truncate text-[11.5px] font-medium",
              isUnassigned && "text-warning",
            )}
          >
            {isUnassigned ? "Unassigned" : row.workerName}
          </span>
          <span className="truncate font-table text-[9.5px] text-muted-foreground tabular-nums">
            {isUnassigned
              ? "Drop here to unassign"
              : row.equipmentCodes.length > 0
                ? row.equipmentCodes.join(" · ")
                : "No tractor"}
            {" · "}
            {row.bars.length} {row.bars.length === 1 ? "load" : "loads"}
          </span>
        </div>
      </div>
      <div className="relative shrink-0" style={{ width: canvasWidth }}>
        {row.bars.map((bar) => (
          <TimelineBarItem
            key={bar.moveId}
            bar={bar}
            range={range}
            zoom={zoom}
            isHighlighted={!!highlightId && highlightId === bar.shipment.id}
            draggable={draggable}
            onHoverChange={onHoverChange}
            onSelect={onSelectBar}
          />
        ))}
      </div>
    </div>
  );
}
