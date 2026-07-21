import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { getDestinationLocation, getOriginLocation } from "@/lib/shipment-utils";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import { usePermissionStore } from "@/stores/permission-store";
import type { FieldFilter, FilterGroup } from "@/types/data-table";
import { Operation, Resource } from "@/types/permission";
import type { AssignmentPayload } from "@/types/shipment";
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useVirtualizer } from "@tanstack/react-virtual";
import { CalendarClockIcon, CircleAlertIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { AssignmentDialog } from "../../assignment-dialog";
import { useCommandCenterStore } from "../store";
import { useCommandCenterUrl, type TimelineZoom } from "../url-state";
import type { CommandCenterTableSummary } from "../command-center-table";
import { RAIL_WIDTH_PX, rowHeightPx } from "./constants";
import {
  buildDayColumns,
  buildHourTicks,
  canvasWidthPx,
  getTimelineRange,
  parseAnchorDate,
  secondsToX,
  serializeAnchorDate,
  shiftAnchor,
} from "./time-scale";
import { TimelineHeader } from "./timeline-header";
import { TimelineRowItem } from "./timeline-row";
import { TimelineToolbar } from "./timeline-toolbar";
import { BarDetailPopover } from "./bar-detail-popover";
import {
  UNASSIGNED_ROW_KEY,
  useTimelineData,
  type TimelineBar,
  type TimelineRow,
} from "./use-timeline-data";

const NOW_TICK_MS = 60_000;

type PendingAssignment = {
  moveId: string;
  existingAssignment: TimelineBar["assignment"];
  prefill: Partial<AssignmentPayload> | null;
};

type CommandCenterTimelineProps = {
  fieldFilters: FieldFilter[];
  filterGroups: FilterGroup[] | undefined;
  query: string;
  onSummaryChange?: (summary: CommandCenterTableSummary) => void;
};

export default function CommandCenterTimeline({
  fieldFilters,
  filterGroups,
  query,
  onSummaryChange,
}: CommandCenterTimelineProps) {
  const [{ at, zoom }, setUrl] = useCommandCenterUrl();
  const [, setPanelParams] = useQueryStates(panelSearchParamsParser);
  const queryClient = useQueryClient();

  const highlightId = useCommandCenterStore.use.highlightId();
  const setHighlightId = useCommandCenterStore.use.setHighlightId();
  const canAssign = usePermissionStore((state) =>
    state.hasPermission(Resource.Shipment, Operation.Update),
  );

  const anchor = useMemo(() => parseAnchorDate(at), [at]);
  const range = useMemo(() => getTimelineRange(anchor, zoom), [anchor, zoom]);
  const canvasWidth = canvasWidthPx(range, zoom);
  const dayColumns = useMemo(() => buildDayColumns(range, zoom), [range, zoom]);
  const hourTicks = useMemo(() => buildHourTicks(range, zoom), [range, zoom]);

  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));
  useEffect(() => {
    const interval = setInterval(() => setNow(Math.floor(Date.now() / 1000)), NOW_TICK_MS);
    return () => clearInterval(interval);
  }, []);
  const nowInRange = now >= range.start && now < range.end;

  const { dataQuery, data } = useTimelineData({
    range,
    fieldFilters,
    filterGroups,
    query,
    enabled: true,
  });

  useEffect(() => {
    if (!dataQuery.data) return;
    onSummaryChange?.({
      totalCount: data.totalCount,
      dataUpdatedAt: dataQuery.dataUpdatedAt,
      backgroundQueriesEnabled: dataQuery.isSuccess && !dataQuery.isFetching,
    });
  }, [
    data.totalCount,
    dataQuery.data,
    dataQuery.dataUpdatedAt,
    dataQuery.isFetching,
    dataQuery.isSuccess,
    onSummaryChange,
  ]);

  const displayRows = useMemo<TimelineRow[]>(
    () => (data.unassignedRow ? [data.unassignedRow, ...data.rows] : data.rows),
    [data.unassignedRow, data.rows],
  );

  const scrollRef = useRef<HTMLDivElement>(null);
  const rowVirtualizer = useVirtualizer({
    count: displayRows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: (index) => rowHeightPx(displayRows[index].laneCount),
    getItemKey: (index) => displayRows[index].key,
    overscan: 6,
  });

  useEffect(() => {
    rowVirtualizer.measure();
  }, [displayRows, rowVirtualizer]);

  const rangeKey = `${range.start}:${zoom}`;
  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const nowSeconds = Math.floor(Date.now() / 1000);
    if (nowSeconds >= range.start && nowSeconds < range.end) {
      const x = secondsToX(nowSeconds, range, zoom);
      el.scrollLeft = Math.max(0, RAIL_WIDTH_PX + x - el.clientWidth / 2);
    } else {
      el.scrollLeft = 0;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- re-center only when the window moves
  }, [rangeKey]);

  const [selectedBar, setSelectedBar] = useState<{
    bar: TimelineBar;
    anchor: HTMLElement;
  } | null>(null);
  const [pendingAssignment, setPendingAssignment] = useState<PendingAssignment | null>(null);
  const [pendingUnassign, setPendingUnassign] = useState<TimelineBar | null>(null);
  const [activeDragBar, setActiveDragBar] = useState<TimelineBar | null>(null);
  const suppressClickRef = useRef(false);

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));

  const handleSelectBar = useCallback((bar: TimelineBar, anchorEl: HTMLElement) => {
    if (suppressClickRef.current) {
      suppressClickRef.current = false;
      return;
    }
    setSelectedBar((prev) => (prev?.bar.moveId === bar.moveId ? null : { bar, anchor: anchorEl }));
  }, []);

  const handleDragStart = useCallback((event: DragStartEvent) => {
    setSelectedBar(null);
    setActiveDragBar((event.active.data.current?.bar as TimelineBar | undefined) ?? null);
  }, []);

  const handleDragEnd = useCallback((event: DragEndEvent) => {
    setActiveDragBar(null);
    suppressClickRef.current = true;
    setTimeout(() => {
      suppressClickRef.current = false;
    }, 0);

    const bar = event.active.data.current?.bar as TimelineBar | undefined;
    const row = event.over?.data.current?.row as TimelineRow | undefined;
    if (!bar || !row) return;

    if (row.key === UNASSIGNED_ROW_KEY) {
      if (bar.assignment) setPendingUnassign(bar);
      return;
    }
    if (bar.assignment?.primaryWorker?.id === row.key) return;

    setPendingAssignment({
      moveId: bar.moveId,
      existingAssignment: bar.assignment,
      prefill: { primaryWorkerId: row.key },
    });
  }, []);

  const { mutate: unassignMove, isPending: isUnassigning } = useMutation({
    mutationFn: (moveId: string) => apiService.assignmentService.unassign(moveId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      toast.success("Move unassigned", {
        description: "The load is back in the unassigned lane.",
      });
      setPendingUnassign(null);
    },
    onError: () => {
      toast.error("Failed to unassign move");
    },
  });

  const handleEdit = useCallback(
    (bar: TimelineBar) => {
      setSelectedBar(null);
      if (bar.shipment.id) {
        void setPanelParams({ panelType: "edit", panelEntityId: bar.shipment.id });
      }
    },
    [setPanelParams],
  );

  const handleViewInTable = useCallback(
    (bar: TimelineBar) => {
      setSelectedBar(null);
      if (bar.shipment.id) {
        void setUrl({ mode: "table", expanded: bar.shipment.id });
      }
    },
    [setUrl],
  );

  const handleReassignFromPopover = useCallback((bar: TimelineBar) => {
    setSelectedBar(null);
    setPendingAssignment({
      moveId: bar.moveId,
      existingAssignment: bar.assignment,
      prefill: null,
    });
  }, []);

  const handleShift = useCallback(
    (direction: 1 | -1) =>
      void setUrl({ at: serializeAnchorDate(shiftAnchor(anchor, zoom, direction)) }),
    [anchor, zoom, setUrl],
  );
  const handleToday = useCallback(() => void setUrl({ at: null }), [setUrl]);
  const handleAnchorSelect = useCallback(
    (date: Date) => void setUrl({ at: serializeAnchorDate(date) }),
    [setUrl],
  );
  const handleZoomChange = useCallback(
    (next: TimelineZoom) => void setUrl({ zoom: next === "day" ? null : next }),
    [setUrl],
  );

  const isInitialLoading = dataQuery.isLoading;
  const isEmpty = !isInitialLoading && !dataQuery.isError && displayRows.length === 0;
  const nowX = secondsToX(now, range, zoom);

  return (
    <div className="flex flex-col">
      <TimelineToolbar
        anchor={anchor}
        zoom={zoom}
        barCount={data.barCount}
        shipmentCount={data.shipmentCount}
        totalCount={data.totalCount}
        truncated={data.truncated}
        isFetching={dataQuery.isFetching && !dataQuery.isLoading}
        onShift={handleShift}
        onToday={handleToday}
        onAnchorSelect={handleAnchorSelect}
        onZoomChange={handleZoomChange}
      />

      <DndContext sensors={sensors} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
        <div
          ref={scrollRef}
          className="relative h-[clamp(420px,58vh,640px)] overflow-auto overscroll-x-contain"
        >
          {isInitialLoading ? (
            <TimelineSkeleton />
          ) : dataQuery.isError ? (
            <TimelineErrorState onRetry={() => void dataQuery.refetch()} />
          ) : isEmpty ? (
            <TimelineEmptyState />
          ) : (
            <div style={{ width: RAIL_WIDTH_PX + canvasWidth }}>
              <TimelineHeader
                range={range}
                zoom={zoom}
                canvasWidth={canvasWidth}
                dayColumns={dayColumns}
                hourTicks={hourTicks}
                now={now}
                driverCount={data.rows.length}
              />
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
                        "absolute inset-y-0 border-l border-border/60 first:border-l-0",
                        day.isWeekend && "bg-muted/40",
                        day.isToday && "bg-brand/[3%]",
                      )}
                      style={{ left: day.x, width: day.width }}
                    />
                  ))}
                  {hourTicks.map((tick) => (
                    <div
                      key={tick.time}
                      className="absolute inset-y-0 border-l border-border/25"
                      style={{ left: tick.x }}
                    />
                  ))}
                </div>

                <div className="relative" style={{ height: rowVirtualizer.getTotalSize() }}>
                  {rowVirtualizer.getVirtualItems().map((virtualRow) => {
                    const row = displayRows[virtualRow.index];
                    return (
                      <div
                        key={virtualRow.key}
                        className="absolute inset-x-0"
                        style={{ transform: `translateY(${virtualRow.start}px)` }}
                      >
                        <TimelineRowItem
                          row={row}
                          range={range}
                          zoom={zoom}
                          canvasWidth={canvasWidth}
                          highlightId={highlightId}
                          draggable={canAssign}
                          droppable={canAssign}
                          onHoverChange={setHighlightId}
                          onSelectBar={handleSelectBar}
                        />
                      </div>
                    );
                  })}
                </div>

                {nowInRange && (
                  <div
                    aria-hidden
                    className="pointer-events-none absolute inset-y-0 z-20 w-px bg-brand"
                    style={{ left: RAIL_WIDTH_PX + nowX }}
                  >
                    <span className="absolute -top-0 -left-[3px] size-[7px] rounded-full bg-brand" />
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        <DragOverlay dropAnimation={null}>
          {activeDragBar && <DragGhost bar={activeDragBar} />}
        </DragOverlay>
      </DndContext>

      <BarDetailPopover
        bar={selectedBar?.bar ?? null}
        anchor={selectedBar?.anchor ?? null}
        onOpenChange={(open) => !open && setSelectedBar(null)}
        onEdit={handleEdit}
        onViewInTable={handleViewInTable}
        onReassign={canAssign ? handleReassignFromPopover : null}
      />

      {pendingAssignment && (
        <AssignmentDialog
          open
          onOpenChange={(open) => !open && setPendingAssignment(null)}
          moveId={pendingAssignment.moveId}
          existingAssignment={pendingAssignment.existingAssignment}
          prefill={pendingAssignment.prefill}
          onAssigned={() => setPendingAssignment(null)}
        />
      )}

      <AlertDialog
        open={!!pendingUnassign}
        onOpenChange={(open) => !open && setPendingUnassign(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Unassign this move?</AlertDialogTitle>
            <AlertDialogDescription>
              {pendingUnassign
                ? `${pendingUnassign.shipment.proNumber ?? "This shipment"} will lose its driver and equipment and return to the unassigned lane.`
                : ""}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={isUnassigning}
              onClick={() => pendingUnassign && unassignMove(pendingUnassign.moveId)}
            >
              Unassign
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function DragGhost({ bar }: { bar: TimelineBar }) {
  const originCode = getOriginLocation(bar.shipment)?.code ?? "—";
  const destCode = getDestinationLocation(bar.shipment)?.code ?? "—";
  return (
    <div className="flex h-6.5 cursor-grabbing items-center rounded border border-brand/50 bg-brand/15 px-2 shadow-lg backdrop-blur-sm">
      <span className="font-table text-[10px] font-semibold tabular-nums">
        {bar.shipment.proNumber ?? "—"}
        <span className="ml-1.5 font-normal text-muted-foreground">
          {originCode} → {destCode}
        </span>
      </span>
    </div>
  );
}

function TimelineSkeleton() {
  return (
    <div className="flex flex-col gap-px p-3">
      <div className="mb-2 flex gap-2">
        <Skeleton className="h-6 w-52" />
        <Skeleton className="h-6 flex-1" />
      </div>
      {Array.from({ length: 8 }).map((_, index) => (
        <div key={index} className="flex items-center gap-2 py-1.5">
          <Skeleton className="h-8 w-52 shrink-0" />
          <Skeleton
            className="h-6.5"
            style={{ width: `${28 + ((index * 17) % 40)}%`, marginLeft: `${(index * 13) % 30}%` }}
          />
        </div>
      ))}
    </div>
  );
}

function TimelineEmptyState() {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2 text-center">
      <CalendarClockIcon className="size-6 text-muted-foreground" />
      <p className="text-sm font-semibold">No scheduled activity in this window</p>
      <p className="max-w-sm text-xs text-muted-foreground">
        No shipments have stops scheduled in the visible range with the current filters. Move the
        window, widen the zoom, or clear filters to see more.
      </p>
    </div>
  );
}

function TimelineErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2 text-center">
      <CircleAlertIcon className="size-6 text-destructive" />
      <p className="text-sm font-semibold">Couldn&apos;t load the timeline</p>
      <p className="max-w-sm text-xs text-muted-foreground">
        Something went wrong while fetching shipments for this window.
      </p>
      <Button type="button" variant="outline" size="xs" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}
