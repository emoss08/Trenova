import { shipmentBoard } from "@/lib/queries/shipment-board";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";
import { STAGE_META } from "@/lib/shipment-board/stage";
import { type BarDragHandle } from "@/components/timeline/timeline-bar-frame";
import {
  TimelineCanvas,
  TimelineGroupRow,
  TimelineRow,
  type TimelineScrollMemory,
  type TimelineSheetItem,
} from "@/components/timeline/timeline-canvas";
import {
  TimelineConflictFlag,
  TimelineLaneBars,
  TimelineLaneGaps,
  TimelineLaneLabel,
} from "@/components/timeline/timeline-lane-parts";
import {
  BAR_HEIGHT,
  GROUP_HEIGHT,
  laneHeight,
  type TimelineScale,
} from "@/components/timeline/timeline-scale";
import {
  buildCoverageLanes,
  type CarrierLane,
  type DriverLane,
  type TimelineLoad,
} from "@/lib/shipment-board/timeline";
import type { PackedLane } from "@/lib/timeline/lanes";
import {
  DndContext,
  DragOverlay,
  KeyboardSensor,
  MouseSensor,
  TouchSensor,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatClockDurationMs } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";
import { IdentityAvatar } from "../../identity-avatar";
import { useBoardActions } from "../../use-board-actions";
import { LOW_HOS_MS, useDriverHos } from "../../use-driver-hos";
import { TimelineAssignDialog, type PendingAssignment } from "./timeline-assign-dialog";
import { TimelineBar } from "./timeline-bar";

/** Drive time below this reads as out. */
const OUT_OF_HOS_MS = 3_600_000;

type LaneSelection = { selectedId: string | null; onSelect: (id: string) => void };

/**
 * One lane per driver and carrier, with the loads still to place in a lane of
 * their own on top. Dragging an uncovered load onto a driver asks to assign it,
 * after checking it against the driver's other loads and drive time.
 */
export function TimelineCoverageLanes({
  loads,
  scale,
  today,
  selectedId,
  onSelect,
  scrollMemory,
}: {
  loads: readonly TimelineLoad[];
  scale: TimelineScale;
  today: number;
  scrollMemory: TimelineScrollMemory;
} & LaneSelection) {
  const t = useT();
  const { runsAssets } = useShipmentCapabilities();
  const actions = useBoardActions();
  const { data: capacity } = useQuery({
    ...shipmentBoard.capacity("Driver"),
    enabled: runsAssets,
    staleTime: 60_000,
  });
  const lanes = buildCoverageLanes(loads, runsAssets ? (capacity?.units ?? []) : []);
  const [dragging, setDragging] = useState<TimelineLoad | null>(null);
  const [pending, setPending] = useState<PendingAssignment | null>(null);
  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 6 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 220, tolerance: 6 } }),
    // Space picks a load up; Enter stays the bar's own click, which opens it.
    useSensor(KeyboardSensor, {
      keyboardCodes: { start: ["Space"], cancel: ["Escape"], end: ["Space", "Enter"] },
    }),
  );

  const open = lanes.pool.filter((load) => load.coverage.kind === "uncovered");
  const openMoveIds = open.map((load) => load.moveId).filter((id): id is string => !!id);
  const loadById = new Map(loads.map((load) => [load.id, load]));
  const laneById = new Map(lanes.drivers.map((lane) => [lane.id, lane]));

  // The sheet's lines: each group's header, then its lanes at their packed heights.
  const items: TimelineSheetItem[] = [
    {
      key: "group:pool",
      height: GROUP_HEIGHT,
      header: true,
      node: (
        <TimelineGroupRow
          scale={scale}
          label={t("Needs coverage")}
          count={lanes.pool.length}
          swatchClassName={STAGE_META.NeedsCoverage.swatchClassName}
          tail={
            runsAssets && open.length > 0 ? (
              <>
                <span className="truncate">{t("Drag a load onto a driver to assign it")}</span>
                <Button
                  variant="outline"
                  size="xxs"
                  isLoading={actions.autoAssign.isPending}
                  loadingText={t("Assigning")}
                  onClick={() => actions.autoAssign.mutate(openMoveIds)}
                >
                  {t(
                    "{0, plural, one {Auto-assign # load} other {Auto-assign # loads}}",
                    openMoveIds.length,
                  )}
                </Button>
              </>
            ) : null
          }
        />
      ),
    },
    {
      key: "lane:pool",
      height: laneHeight(lanes.poolPacked.rows),
      node: (
        <LaneRow
          scale={scale}
          packed={lanes.poolPacked}
          draggable={runsAssets}
          selectedId={selectedId}
          onSelect={onSelect}
          label={
            <>
              <span
                aria-hidden
                className="border-warning size-5.5 shrink-0 rounded-full border-[1.5px] border-dashed"
              />
              <TimelineLaneLabel
                title={t("Uncovered")}
                meta={t("{0} open · {1} tendered", open.length, lanes.pool.length - open.length)}
              />
            </>
          }
        />
      ),
    },
  ];
  if (runsAssets) {
    items.push({
      key: "group:drivers",
      height: GROUP_HEIGHT,
      header: true,
      node: (
        <TimelineGroupRow
          scale={scale}
          label={t("Drivers")}
          count={lanes.drivers.length}
          tail={
            <span className="truncate">
              {t("Free time is labeled; a red outline is an overlap")}
            </span>
          }
        />
      ),
    });
    for (const lane of lanes.drivers) {
      items.push({
        key: `driver:${lane.id}`,
        height: laneHeight(lane.packed.rows),
        node: (
          <DriverLaneRow lane={lane} scale={scale} selectedId={selectedId} onSelect={onSelect} />
        ),
      });
    }
  }
  if (lanes.carriers.length > 0) {
    items.push({
      key: "group:carriers",
      height: GROUP_HEIGHT,
      header: true,
      node: <TimelineGroupRow scale={scale} label={t("Carriers")} count={lanes.carriers.length} />,
    });
    for (const lane of lanes.carriers) {
      items.push({
        key: `carrier:${lane.id}`,
        height: laneHeight(lane.packed.rows),
        node: (
          <CarrierLaneRow lane={lane} scale={scale} selectedId={selectedId} onSelect={onSelect} />
        ),
      });
    }
  }

  // The selected load is revealed by the lane that holds it.
  let revealKey: string | null = null;
  if (selectedId) {
    if (lanes.pool.some((load) => load.id === selectedId)) revealKey = "lane:pool";
    const driver = lanes.drivers.find((lane) => lane.loads.some((load) => load.id === selectedId));
    if (driver) revealKey = `driver:${driver.id}`;
    const carrier = lanes.carriers.find((lane) =>
      lane.loads.some((load) => load.id === selectedId),
    );
    if (carrier) revealKey = `carrier:${carrier.id}`;
  }

  const onDragStart = ({ active }: DragStartEvent) =>
    setDragging(loadById.get(String(active.id)) ?? null);
  const onDragEnd = ({ active, over }: DragEndEvent) => {
    setDragging(null);
    const load = loadById.get(String(active.id));
    const lane = over ? laneById.get(String(over.data.current?.laneId)) : undefined;
    if (load && lane) setPending({ load, lane });
  };

  return (
    <DndContext
      sensors={sensors}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
      onDragCancel={() => setDragging(null)}
    >
      <TimelineCanvas
        scale={scale}
        today={today}
        labelHeading={t("Driver or carrier")}
        items={items}
        revealKey={revealKey}
        scrollMemory={scrollMemory}
      />

      <DragOverlay dropAnimation={null}>
        {dragging ? (
          <span className="bg-card ui-lift-float border-warning inline-flex items-center gap-1.5 rounded-md border-[1.5px] border-dashed px-2 py-1 text-xs whitespace-nowrap">
            <span className="font-mono font-semibold">{dragging.shipment.proNumber ?? "—"}</span>
            {[dragging.stops[0].city, dragging.stops[dragging.stops.length - 1].city].join(" → ")}
          </span>
        ) : null}
      </DragOverlay>

      <TimelineAssignDialog pending={pending} onClose={() => setPending(null)} />
    </DndContext>
  );
}

/** A lane's loads at their packed rows; uncovered loads in the pool can be picked up. */
function LaneBars({
  packed,
  scale,
  draggable,
  selectedId,
  onSelect,
}: {
  packed: PackedLane<TimelineLoad>;
  scale: TimelineScale;
  draggable: boolean;
} & LaneSelection) {
  return (
    <TimelineLaneBars
      packed={packed}
      scale={scale}
      renderBar={(load, place) => {
        const props = {
          load,
          scale,
          top: place.top,
          height: BAR_HEIGHT,
          caption: "lane" as const,
          nextLeft: place.nextLeft,
          conflict: place.conflict,
          selected: load.id === selectedId,
          onSelect,
        };
        return draggable && load.coverage.kind === "uncovered" ? (
          <DraggableBar {...props} />
        ) : (
          <TimelineBar {...props} />
        );
      }}
    />
  );
}

function DraggableBar(props: Parameters<typeof TimelineBar>[0]) {
  const { setNodeRef, attributes, listeners, isDragging } = useDraggable({ id: props.load.id });
  const drag: BarDragHandle = {
    ref: setNodeRef,
    props: { ...attributes, ...listeners },
    dragging: isDragging,
  };
  return <TimelineBar {...props} drag={drag} />;
}

function LaneRow({
  scale,
  packed,
  label,
  draggable = false,
  selectedId,
  onSelect,
}: {
  scale: TimelineScale;
  packed: PackedLane<TimelineLoad>;
  label: ReactNode;
  draggable?: boolean;
} & LaneSelection) {
  return (
    <TimelineRow
      scale={scale}
      height={laneHeight(packed.rows)}
      label={label}
      lane={
        <LaneBars
          packed={packed}
          scale={scale}
          draggable={draggable}
          selectedId={selectedId}
          onSelect={onSelect}
        />
      }
    />
  );
}

function DriverLaneRow({
  lane,
  scale,
  selectedId,
  onSelect,
}: {
  lane: DriverLane;
  scale: TimelineScale;
} & LaneSelection) {
  const t = useT();
  // `active` is the load being dragged; only the lane under it redraws as the pointer moves.
  const { setNodeRef, isOver, active } = useDroppable({
    id: `driver:${lane.id}`,
    data: { laneId: lane.id },
  });
  const readHos = useDriverHos(lane.driveRemainingMs == null ? lane.id : null);
  const remaining = lane.driveRemainingMs ?? readHos;
  const { rows, conflicts } = lane.packed;
  const height = laneHeight(rows);

  return (
    <TimelineRow
      scale={scale}
      height={height}
      laneRef={setNodeRef}
      dropState={active && isOver ? (lane.tractorId ? "ok" : "refused") : null}
      labelTitle={lane.name}
      label={
        <>
          <IdentityAvatar
            id={lane.id}
            initials={lane.initials}
            shape="circle"
            className="size-5.5 text-2xs"
          />
          <TimelineLaneLabel
            title={lane.name}
            flag={conflicts.size > 0 ? <TimelineConflictFlag /> : null}
            meta={
              <>
                <span className="truncate">{lane.tractorCode ?? t("No tractor")}</span>
                {remaining != null ? (
                  <>
                    <span aria-hidden>·</span>
                    <span
                      title={t("Drive time left today")}
                      className={cn(
                        "shrink-0 font-mono tabular-nums",
                        remaining < OUT_OF_HOS_MS
                          ? "text-danger"
                          : remaining < LOW_HOS_MS
                            ? "text-warning"
                            : undefined,
                      )}
                    >
                      {t("{0} left", formatClockDurationMs(remaining))}
                    </span>
                  </>
                ) : null}
              </>
            }
          />
        </>
      }
      lane={
        <>
          <TimelineLaneGaps items={lane.loads} scale={scale} />
          <LaneBars
            packed={lane.packed}
            scale={scale}
            draggable={false}
            selectedId={selectedId}
            onSelect={onSelect}
          />
        </>
      }
    />
  );
}

function CarrierLaneRow({
  lane,
  scale,
  selectedId,
  onSelect,
}: {
  lane: CarrierLane;
  scale: TimelineScale;
} & LaneSelection) {
  const t = useT();
  return (
    <LaneRow
      scale={scale}
      packed={lane.packed}
      selectedId={selectedId}
      onSelect={onSelect}
      label={
        <>
          <IdentityAvatar
            id={lane.id}
            initials={lane.initials}
            shape="square"
            className="size-5.5 text-2xs"
          />
          <TimelineLaneLabel
            title={lane.name}
            meta={
              <>
                {lane.detail ? <span className="font-mono">{lane.detail}</span> : null}
                <span className="truncate">
                  {t("{0, plural, one {# load} other {# loads}}", lane.loads.length)}
                </span>
              </>
            }
          />
        </>
      }
    />
  );
}
