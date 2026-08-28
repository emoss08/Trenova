import type {
  DispatchBoardDriver,
  DispatchBoardMove,
  DispatchCommitment,
  DispatchTimeOff,
} from "@/lib/graphql/dispatch-console";
import {
  buildDayColumnsForRange,
  buildHourTicksForRange,
  canvasWidthForRange,
  packLanes,
  secondsToXForRange,
  type TimeRange,
} from "@/lib/timeline/time-scale";
import { useDraggable, useDroppable } from "@dnd-kit/core";
import { Avatar, AvatarFallback, AvatarImage } from "@trenova/shared/components/ui/avatar";
import { useVirtualizer } from "@tanstack/react-virtual";
import { cn } from "@trenova/shared/lib/utils";
import { CalendarClockIcon, InboxIcon, MoonIcon } from "lucide-react";
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { availabilityMeta, workerInitials, type UrgencyBucket } from "./dispatch-vocabulary";
import {
  BAR_HEIGHT_PX,
  DAY_LABEL_HEIGHT_PX,
  HOUR_TICK_HEIGHT_PX,
  LANE_HEIGHT_PX,
  RAIL_WIDTH_PX,
  ROW_PADDING_PX,
  timelineZoomConfig,
  type TimelineZoomDays,
} from "./timeline-metrics";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";

const MIN_SPAN_SECONDS = 1800;
const NOW_TICK_MS = 60_000;

const UNCOVERED_TONE: Record<UrgencyBucket, string> = {
  Late: "bg-destructive/75 text-white hover:bg-destructive/90",
  Now: "bg-warning/80 text-warning-foreground hover:bg-warning",
  Today: "bg-blue-500/70 text-white hover:bg-blue-500/85",
  Tomorrow: "bg-purple-500/65 text-white hover:bg-purple-500/80",
  Planned: "bg-muted-foreground/40 text-white hover:bg-muted-foreground/55",
};

type MoveSpan = {
  move: DispatchBoardMove;
  start: number;
  end: number;
};

type CommitmentSpan = {
  commitment: DispatchCommitment;
  start: number;
  end: number;
};

function moveSpan(move: DispatchBoardMove): MoveSpan | null {
  if (move.originWindowStart <= 0) return null;
  const start = move.originWindowStart;
  const candidates = [
    move.destinationWindowEnd ?? 0,
    move.destinationWindowStart,
    move.originWindowEnd ?? 0,
  ];
  const end = Math.max(start + MIN_SPAN_SECONDS, ...candidates);
  return { move, start, end };
}

function commitmentSpan(commitment: DispatchCommitment): CommitmentSpan | null {
  if (commitment.windowStart <= 0) return null;
  const end = Math.max(commitment.windowStart + MIN_SPAN_SECONDS, commitment.windowEnd);
  return { commitment, start: commitment.windowStart, end };
}

function overlapsRange(start: number, end: number, range: TimeRange): boolean {
  return end > range.start && start < range.end;
}

type UnassignedRowModel = {
  kind: "unassigned";
  spans: MoveSpan[];
  laneByIndex: number[];
  laneCount: number;
};

type DriverRowModel = {
  kind: "driver";
  driver: DispatchBoardDriver;
  commitments: CommitmentSpan[];
  laneByIndex: number[];
  laneCount: number;
  timeOff: DispatchTimeOff[];
};

type RowModel = UnassignedRowModel | DriverRowModel;

function rowKey(row: RowModel): string {
  return row.kind === "unassigned" ? "unassigned" : row.driver.workerId;
}

function rowHeight(row: RowModel): number {
  return row.laneCount * LANE_HEIGHT_PX + ROW_PADDING_PX;
}

function buildRows(
  moves: readonly DispatchBoardMove[],
  drivers: readonly DispatchBoardDriver[],
  range: TimeRange,
): RowModel[] {
  const uncoveredSpans = moves
    .filter((move) => !move.isCovered)
    .map(moveSpan)
    .filter((span): span is MoveSpan => span !== null && overlapsRange(span.start, span.end, range))
    .sort((a, b) => a.start - b.start);
  const uncoveredPacking = packLanes(uncoveredSpans);

  const rows: RowModel[] = [
    {
      kind: "unassigned",
      spans: uncoveredSpans,
      laneByIndex: uncoveredPacking.laneByIndex,
      laneCount: uncoveredSpans.length > 0 ? uncoveredPacking.laneCount : 1,
    },
  ];

  for (const driver of drivers) {
    const commitments = driver.commitments
      .map(commitmentSpan)
      .filter(
        (span): span is CommitmentSpan =>
          span !== null && overlapsRange(span.start, span.end, range),
      )
      .sort((a, b) => a.start - b.start);
    const packing = packLanes(commitments);
    rows.push({
      kind: "driver",
      driver,
      commitments,
      laneByIndex: packing.laneByIndex,
      laneCount: commitments.length > 0 ? packing.laneCount : 1,
      timeOff: driver.timeOff.filter((pto) => overlapsRange(pto.startDate, pto.endDate, range)),
    });
  }

  return rows;
}

function barGeometry(start: number, end: number, range: TimeRange, pxPerHour: number) {
  const left = secondsToXForRange(start, range, pxPerHour);
  const right = secondsToXForRange(end, range, pxPerHour);
  return { left, width: Math.max(right - left, 14) };
}

function UncoveredBar({
  span,
  lane,
  range,
  pxPerHour,
  isSelected,
  onSelect,
}: {
  span: MoveSpan;
  lane: number;
  range: TimeRange;
  pxPerHour: number;
  isSelected: boolean;
  onSelect: (moveId: string) => void;
}) {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `move:${span.move.moveId}`,
    data: { type: "move", move: span.move },
  });
  const geometry = barGeometry(span.start, span.end, range, pxPerHour);
  const tone = UNCOVERED_TONE[(span.move.urgency as UrgencyBucket) ?? "Planned"];

  return (
    <button
      ref={setNodeRef}
      {...attributes}
      {...listeners}
      type="button"
      title={`${span.move.proNumber} · ${span.move.originCity} → ${span.move.destinationCity}`}
      onClick={() => onSelect(span.move.moveId)}
      className={cn(
        "focus-visible:ring-brand absolute flex cursor-grab items-center overflow-hidden rounded px-1.5 transition-[opacity,box-shadow] outline-none focus-visible:ring-2 active:cursor-grabbing",
        tone,
        isDragging && "opacity-40",
        isSelected && "shadow-[0_0_0_2px_var(--brand)]",
      )}
      style={{
        left: geometry.left,
        width: geometry.width,
        height: BAR_HEIGHT_PX,
        top: ROW_PADDING_PX / 2 + lane * LANE_HEIGHT_PX + (LANE_HEIGHT_PX - BAR_HEIGHT_PX) / 2,
      }}
    >
      <span className="truncate font-mono text-[10px] font-semibold tabular-nums">
        {span.move.proNumber}
      </span>
    </button>
  );
}

function CommitmentBar({
  span,
  lane,
  range,
  pxPerHour,
  onSelect,
}: {
  span: CommitmentSpan;
  lane: number;
  range: TimeRange;
  pxPerHour: number;
  onSelect: (moveId: string) => void;
}) {
  const geometry = barGeometry(span.start, span.end, range, pxPerHour);
  const inTransit = span.commitment.moveStatus === "InTransit";

  return (
    <button
      type="button"
      title={`${span.commitment.proNumber} → ${span.commitment.destinationCity}, ${span.commitment.destinationState}`}
      onClick={() => onSelect(span.commitment.moveId)}
      className={cn(
        "focus-visible:ring-brand absolute flex cursor-pointer items-center overflow-hidden rounded px-1.5 transition-colors outline-none focus-visible:ring-2",
        inTransit
          ? "bg-brand text-brand-foreground hover:bg-brand/85"
          : "bg-brand/60 text-brand-foreground hover:bg-brand/75",
      )}
      style={{
        left: geometry.left,
        width: geometry.width,
        height: BAR_HEIGHT_PX,
        top: ROW_PADDING_PX / 2 + lane * LANE_HEIGHT_PX + (LANE_HEIGHT_PX - BAR_HEIGHT_PX) / 2,
      }}
    >
      <span className="truncate font-mono text-[10px] font-semibold tabular-nums">
        {span.commitment.proNumber}
      </span>
    </button>
  );
}

function DriverLaneRow({
  row,
  range,
  pxPerHour,
  canvasWidth,
  isSelected,
  onSelectDriver,
  onSelectMove,
}: {
  row: DriverRowModel;
  range: TimeRange;
  pxPerHour: number;
  canvasWidth: number;
  isSelected: boolean;
  onSelectDriver: (workerId: string) => void;
  onSelectMove: (moveId: string) => void;
}) {
  const { driver } = row;
  const isBlocked = driver.availability === "Blocked";
  const { setNodeRef, isOver, active } = useDroppable({
    id: `driver-lane:${driver.workerId}`,
    disabled: isBlocked,
    data: { type: "driver-target", driver },
  });
  const availability = availabilityMeta(driver.availability);
  const height = rowHeight(row);
  const showDropHint = isOver && active?.data.current?.type === "move";

  const availableFrom = Math.max(driver.projectedTimeAvailable, range.start);
  const availableGeometry =
    !isBlocked && availableFrom < range.end
      ? barGeometry(availableFrom, range.end, range, pxPerHour)
      : null;

  return (
    <div
      ref={setNodeRef}
      className={cn(
        "border-border/70 flex border-b transition-colors",
        showDropHint && "bg-brand/10",
      )}
      style={{ height }}
    >
      <button
        type="button"
        onClick={() => onSelectDriver(driver.workerId)}
        className={cn(
          "border-border bg-card hover:bg-muted/60 sticky left-0 z-30 flex shrink-0 items-center gap-1.5 border-r px-2 text-left transition-colors",
          isSelected && "bg-muted",
          showDropHint && "bg-[color-mix(in_oklch,var(--brand)_8%,var(--card))]",
        )}
        style={{ width: RAIL_WIDTH_PX }}
      >
        <Avatar className="size-6 shrink-0">
          {driver.profilePicUrl && (
            <AvatarImage
              src={driver.profilePicUrl}
              alt={`${driver.firstName} ${driver.lastName}`}
            />
          )}
          <AvatarFallback className="text-[9px]">
            {workerInitials(driver.firstName, driver.lastName)}
          </AvatarFallback>
        </Avatar>
        <div className="flex min-w-0 flex-col">
          <span className="truncate text-[11.5px] font-medium">
            {driver.firstName} {driver.lastName}
          </span>
          <span className="text-muted-foreground truncate text-[9.5px] tabular-nums">
            {driver.tractorCode || "No tractor"} · {availability.label}
          </span>
        </div>
      </button>

      <div className="relative shrink-0" style={{ width: canvasWidth }}>
        {availableGeometry && (
          <div
            aria-hidden
            className="bg-success/[7%] absolute inset-y-0"
            style={{ left: availableGeometry.left, width: availableGeometry.width }}
          />
        )}
        {row.timeOff.map((pto) => {
          const geometry = barGeometry(pto.startDate, pto.endDate, range, pxPerHour);
          return (
            <div
              key={`${pto.startDate}-${pto.endDate}-${pto.type}`}
              title={`${pto.type} time off`}
              className="absolute inset-y-0 flex items-center justify-center bg-purple-500/15"
              style={{ left: geometry.left, width: geometry.width }}
            >
              <MoonIcon className="size-3 text-purple-500/70" aria-hidden />
            </div>
          );
        })}
        {row.commitments.map((span, index) => (
          <CommitmentBar
            key={span.commitment.moveId}
            span={span}
            lane={row.laneByIndex[index]}
            range={range}
            pxPerHour={pxPerHour}
            onSelect={onSelectMove}
          />
        ))}
      </div>
    </div>
  );
}

function UnassignedLaneRow({
  row,
  range,
  pxPerHour,
  canvasWidth,
  selectedMoveId,
  onSelectMove,
}: {
  row: UnassignedRowModel;
  range: TimeRange;
  pxPerHour: number;
  canvasWidth: number;
  selectedMoveId: string | null;
  onSelectMove: (moveId: string) => void;
}) {
  const height = rowHeight(row);

  return (
    <div className="border-border/70 bg-warning/[4%] flex border-b" style={{ height }}>
      <div
        className="border-border sticky left-0 z-30 flex shrink-0 items-center gap-1.5 border-r bg-[color-mix(in_oklch,var(--warning)_5%,var(--card))] px-2"
        style={{ width: RAIL_WIDTH_PX }}
      >
        <span className="bg-warning/15 text-warning flex size-6 shrink-0 items-center justify-center rounded-full">
          <InboxIcon className="size-3.5" />
        </span>
        <div className="flex min-w-0 flex-col">
          <span className="text-warning truncate text-[11.5px] font-medium">Uncovered</span>
          <span className="text-muted-foreground truncate text-[9.5px] tabular-nums">
            {row.spans.length} {row.spans.length === 1 ? "move" : "moves"} · drag onto a driver
          </span>
        </div>
      </div>
      <div className="relative shrink-0" style={{ width: canvasWidth }}>
        {row.spans.map((span, index) => (
          <UncoveredBar
            key={span.move.moveId}
            span={span}
            lane={row.laneByIndex[index]}
            range={range}
            pxPerHour={pxPerHour}
            isSelected={selectedMoveId === span.move.moveId}
            onSelect={onSelectMove}
          />
        ))}
      </div>
    </div>
  );
}

/**
 * The planning board — committed work, approved time off, and projected availability on
 * one time axis, with the uncovered queue as a draggable top lane. This is how relay and
 * multi-move planning actually gets done.
 */
export function DispatchTimeline({
  moves,
  drivers,
  range,
  zoom,
  selectedMoveId,
  selectedWorkerId,
  onSelectMove,
  onSelectDriver,
}: {
  moves: readonly DispatchBoardMove[];
  drivers: readonly DispatchBoardDriver[];
  range: TimeRange;
  zoom: TimelineZoomDays;
  selectedMoveId: string | null;
  selectedWorkerId: string | null;
  onSelectMove: (moveId: string) => void;
  onSelectDriver: (workerId: string) => void;
}) {
  const { pxPerHour, hourTickStep } = timelineZoomConfig(zoom);
  const canvasWidth = canvasWidthForRange(range, pxPerHour);
  const dayColumns = useMemo(() => buildDayColumnsForRange(range, pxPerHour), [range, pxPerHour]);
  const hourTicks = useMemo(
    () => buildHourTicksForRange(range, pxPerHour, hourTickStep),
    [range, pxPerHour, hourTickStep],
  );

  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));
  useEffect(() => {
    const interval = setInterval(() => setNow(Math.floor(Date.now() / 1000)), NOW_TICK_MS);
    return () => clearInterval(interval);
  }, []);
  const nowInRange = now >= range.start && now < range.end;
  const nowX = secondsToXForRange(now, range, pxPerHour);

  const rows = useMemo(() => buildRows(moves, drivers, range), [moves, drivers, range]);

  const scrollRef = useRef<HTMLDivElement>(null);
  const rowVirtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: (index) => rowHeight(rows[index]),
    getItemKey: (index) => rowKey(rows[index]),
    overscan: 6,
  });

  useEffect(() => {
    rowVirtualizer.measure();
  }, [rows, rowVirtualizer]);

  const rangeKey = `${range.start}:${zoom}`;
  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const nowSeconds = Math.floor(Date.now() / 1000);
    if (nowSeconds >= range.start && nowSeconds < range.end) {
      const x = secondsToXForRange(nowSeconds, range, pxPerHour);
      el.scrollLeft = Math.max(0, RAIL_WIDTH_PX + x - el.clientWidth / 2);
    } else {
      el.scrollLeft = 0;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- re-center only when the window moves
  }, [rangeKey]);

  const hasHourRow = hourTicks.length > 0;
  const headerHeight = DAY_LABEL_HEIGHT_PX + (hasHourRow ? HOUR_TICK_HEIGHT_PX : 0);

  if (drivers.length === 0) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 text-center">
        <CalendarClockIcon className="text-muted-foreground size-6" />
        <p className="text-sm font-semibold">No drivers to plan</p>
        <p className="text-muted-foreground max-w-sm text-xs">
          No active, assignable drivers match the current filters.
        </p>
      </div>
    );
  }

  return (
    <div ref={scrollRef} className="relative h-full overflow-auto overscroll-x-contain">
      <div style={{ width: RAIL_WIDTH_PX + canvasWidth }}>
        <div
          className="border-border sticky top-0 z-40 flex border-b"
          style={{ height: headerHeight }}
        >
          <div
            className="border-border bg-muted sticky left-0 z-50 flex shrink-0 items-center border-r px-2.5"
            style={{ width: RAIL_WIDTH_PX }}
          >
            <span className="text-muted-foreground text-[9.5px] font-semibold tracking-wide uppercase">
              Drivers · {drivers.length}
            </span>
          </div>
          <div className="bg-muted relative shrink-0" style={{ width: canvasWidth }}>
            {dayColumns.map((day) => (
              <div
                key={day.start}
                className={cn(
                  "border-border/70 absolute top-0 flex items-center border-l px-1.5 first:border-l-0",
                  day.isToday && "text-brand",
                )}
                style={{ left: day.x, width: day.width, height: DAY_LABEL_HEIGHT_PX }}
              >
                <span className="truncate text-[10px] font-semibold tracking-wide uppercase">
                  {day.label}
                </span>
              </div>
            ))}
            {hasHourRow &&
              hourTicks.map((tick) => (
                <span
                  key={tick.time}
                  className="font-table text-muted-foreground absolute -translate-x-1/2 text-[9px] tabular-nums"
                  style={{ left: tick.x, top: DAY_LABEL_HEIGHT_PX + 3 }}
                >
                  {tick.label}
                </span>
              ))}
            {nowInRange && (
              <span
                className="bg-brand font-table text-brand-foreground absolute bottom-0 z-10 -translate-x-1/2 rounded-t px-1 py-px text-[8.5px] font-semibold tabular-nums"
                style={{ left: nowX }}
              >
                {formatUnixInUserTimezone(now, {
                  hour: "2-digit",
                  minute: "2-digit",
                  hour12: false,
                })}
              </span>
            )}
          </div>
        </div>

        <div className="relative">
          <div
            aria-hidden
            className="pointer-events-none absolute inset-y-0 z-0"
            style={{ left: RAIL_WIDTH_PX, width: canvasWidth }}
          >
            {dayColumns.map((day) => (
              <div
                key={day.start}
                className={cn(
                  "border-border/60 absolute inset-y-0 border-l first:border-l-0",
                  day.isWeekend && "bg-muted/40",
                  day.isToday && "bg-brand/[3%]",
                )}
                style={{ left: day.x, width: day.width }}
              />
            ))}
            {hourTicks.map((tick) => (
              <div
                key={tick.time}
                className="border-border/25 absolute inset-y-0 border-l"
                style={{ left: tick.x }}
              />
            ))}
          </div>

          <div className="relative" style={{ height: rowVirtualizer.getTotalSize() }}>
            {rowVirtualizer.getVirtualItems().map((virtualRow) => {
              const row = rows[virtualRow.index];
              return (
                <div
                  key={virtualRow.key}
                  className="absolute inset-x-0"
                  style={{ transform: `translateY(${virtualRow.start}px)` }}
                >
                  {row.kind === "unassigned" ? (
                    <UnassignedLaneRow
                      row={row}
                      range={range}
                      pxPerHour={pxPerHour}
                      canvasWidth={canvasWidth}
                      selectedMoveId={selectedMoveId}
                      onSelectMove={onSelectMove}
                    />
                  ) : (
                    <DriverLaneRow
                      row={row}
                      range={range}
                      pxPerHour={pxPerHour}
                      canvasWidth={canvasWidth}
                      isSelected={selectedWorkerId === row.driver.workerId}
                      onSelectDriver={onSelectDriver}
                      onSelectMove={onSelectMove}
                    />
                  )}
                </div>
              );
            })}
          </div>

          {nowInRange && (
            <div
              aria-hidden
              className="bg-brand pointer-events-none absolute inset-y-0 z-20 w-px"
              style={{ left: RAIL_WIDTH_PX + nowX }}
            >
              <span className="bg-brand absolute -top-0 -left-[3px] size-[7px] rounded-full" />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
