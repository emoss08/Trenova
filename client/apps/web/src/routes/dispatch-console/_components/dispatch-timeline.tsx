import { PointerHoverCard } from "@/components/pointer-hover-card";
import {
  barGeometry,
  TimelineBarFrame,
  TimelineDrivenFill,
  type BarDragHandle,
} from "@/components/timeline/timeline-bar-frame";
import {
  TimelineCanvas,
  TimelineGroupRow,
  TimelineRow,
  type TimelineSheetItem,
} from "@/components/timeline/timeline-canvas";
import { captionWidth, formatTimelineTime } from "@/components/timeline/timeline-format";
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
  ROW_HEIGHT,
  type TimelineScale,
} from "@/components/timeline/timeline-scale";
import type {
  DispatchBoardDriver,
  DispatchBoardMove,
  DispatchCommitment,
  DispatchTimeOff,
} from "@/lib/graphql/dispatch-console";
import { packLane, type PackedLane, type TimelineSpan } from "@/lib/timeline/lanes";
import { useDraggable, useDroppable } from "@dnd-kit/core";
import { Avatar, AvatarFallback, AvatarImage } from "@trenova/shared/components/ui/avatar";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatClockDurationMs } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  AVAILABILITY_SORT_RANK,
  availabilityMeta,
  groupMovesByUrgency,
  urgencyMeta,
  workerInitials,
  type UrgencyBucket,
} from "./dispatch-vocabulary";

/** A bar is never drawn shorter than this, so a move with one appointment still reads. */
const MIN_SPAN_SECONDS = 1_800;
/** Drive time below this reads as out, below LOW as running short. */
const OUT_OF_HOS_MS = 3_600_000;
const LOW_HOS_MS = 4 * 3_600_000;
const HOVER_CARD_WIDTH = 288;

export type MoveSpan = TimelineSpan & { move: DispatchBoardMove };
export type CommitmentSpan = TimelineSpan & { commitment: DispatchCommitment };

export type ConsoleDriverLane = {
  driver: DispatchBoardDriver;
  packed: PackedLane<CommitmentSpan>;
  commitments: CommitmentSpan[];
  timeOff: DispatchTimeOff[];
};

export type ConsoleLanes = {
  /** Moves still needing a driver, in the queue's urgency groups, the earliest pickup first. */
  groups: { bucket: UrgencyBucket; moves: MoveSpan[] }[];
  moves: MoveSpan[];
  drivers: ConsoleDriverLane[];
};

type HoverItem =
  | { id: string; kind: "move"; move: DispatchBoardMove }
  | { id: string; kind: "commitment"; commitment: DispatchCommitment; driver: DispatchBoardDriver };

function moveSpan(move: DispatchBoardMove): MoveSpan | null {
  if (move.originWindowStart <= 0) return null;
  const start = move.originWindowStart;
  const end = Math.max(
    start + MIN_SPAN_SECONDS,
    move.destinationWindowEnd ?? 0,
    move.destinationWindowStart,
    move.originWindowEnd ?? 0,
  );
  return { id: move.moveId, start, end, move };
}

function commitmentSpan(commitment: DispatchCommitment): CommitmentSpan | null {
  if (commitment.windowStart <= 0) return null;
  const end = Math.max(commitment.windowStart + MIN_SPAN_SECONDS, commitment.windowEnd);
  return { id: commitment.moveId, start: commitment.windowStart, end, commitment };
}

function inRange(span: { start: number; end: number }, scale: TimelineScale): boolean {
  return span.end > scale.start && span.start < scale.end;
}

/**
 * The console's lanes for the window: the moves still needing a driver in a pool, then one
 * lane per driver with the work they hold and their time off. Drivers read as the Drivers
 * tab does, open capacity first and the soonest free within each availability.
 */
export function buildConsoleLanes(
  moves: readonly DispatchBoardMove[],
  drivers: readonly DispatchBoardDriver[],
  scale: TimelineScale,
): ConsoleLanes {
  const open = moves
    .filter((move) => !move.isCovered)
    .map(moveSpan)
    .filter((span): span is MoveSpan => span !== null && inRange(span, scale))
    .sort((a, b) => a.start - b.start);
  const groups = [
    ...groupMovesByUrgency(open.map((span) => ({ urgency: span.move.urgency, span }))),
  ]
    .filter(([, inBucket]) => inBucket.length > 0)
    .map(([bucket, inBucket]) => ({ bucket, moves: inBucket.map((entry) => entry.span) }));

  const lanes = drivers.map((driver) => {
    const commitments = driver.commitments
      .map(commitmentSpan)
      .filter((span): span is CommitmentSpan => span !== null && inRange(span, scale));
    return {
      driver,
      commitments,
      packed: packLane(commitments),
      timeOff: driver.timeOff.filter(
        (pto) => pto.endDate > scale.start && pto.startDate < scale.end,
      ),
    };
  });
  lanes.sort(
    (a, b) =>
      (AVAILABILITY_SORT_RANK[a.driver.availability] ?? 5) -
        (AVAILABILITY_SORT_RANK[b.driver.availability] ?? 5) ||
      a.driver.projectedTimeAvailable - b.driver.projectedTimeAvailable ||
      `${a.driver.firstName} ${a.driver.lastName}`.localeCompare(
        `${b.driver.firstName} ${b.driver.lastName}`,
      ),
  );

  return { groups, moves: open, drivers: lanes };
}

/** Where the selected move or driver is drawn, so the sheet can bring it into view. */
export function revealKeyFor(
  lanes: ConsoleLanes,
  selectedMoveId: string | null,
  selectedWorkerId: string | null,
): string | null {
  if (selectedWorkerId) return `driver:${selectedWorkerId}`;
  if (!selectedMoveId) return null;
  if (lanes.moves.some((span) => span.id === selectedMoveId)) return `move:${selectedMoveId}`;
  const holder = lanes.drivers.find((lane) =>
    lane.commitments.some((span) => span.id === selectedMoveId),
  );
  return holder ? `driver:${holder.driver.workerId}` : null;
}

function timeOffTitle(t: TranslateFn, type: string): string {
  switch (type) {
    case "Personal":
      return t("Personal time off");
    case "Vacation":
      return t("Vacation time off");
    case "Sick":
      return t("Sick time off");
    case "Holiday":
      return t("Holiday time off");
    case "Bereavement":
      return t("Bereavement time off");
    case "Maternity":
      return t("Maternity time off");
    case "Paternity":
      return t("Paternity time off");
    default:
      return t("Time off ({0})", type);
  }
}

type Selection = {
  selectedMoveId: string | null;
  onSelectMove: (moveId: string) => void;
};

/**
 * The planning sheet: moves still needing a driver in a pool on top, then each driver's
 * committed work and time off on the same axis, with their free time labeled. Dragging a
 * move from the pool onto a driver opens the pre-flight for that pairing; nothing is
 * assigned until the dispatcher confirms it there. It is the shipments board's timeline,
 * fed the console's moves and drivers.
 */
export function DispatchTimeline({
  moves,
  drivers,
  scale,
  today,
  selectedMoveId,
  selectedWorkerId,
  onSelectMove,
  onSelectDriver,
}: {
  moves: readonly DispatchBoardMove[];
  drivers: readonly DispatchBoardDriver[];
  scale: TimelineScale;
  today: number;
  selectedWorkerId: string | null;
  onSelectDriver: (workerId: string) => void;
} & Selection) {
  const t = useT();
  const lanes = buildConsoleLanes(moves, drivers, scale);

  const items: TimelineSheetItem[] = [];
  if (lanes.moves.length === 0) {
    items.push({
      key: "group:pool",
      height: GROUP_HEIGHT,
      header: true,
      node: (
        <TimelineGroupRow
          scale={scale}
          label={t("Needs a driver")}
          count={0}
          swatchClassName="bg-warning"
          tail={<span className="truncate">{t("Nothing to cover in this window")}</span>}
        />
      ),
    });
  }
  lanes.groups.forEach((group, index) => {
    const meta = urgencyMeta(group.bucket);
    items.push({
      key: `group:${group.bucket}`,
      height: GROUP_HEIGHT,
      header: true,
      node: (
        <TimelineGroupRow
          scale={scale}
          label={t(meta.label)}
          count={group.moves.length}
          swatchClassName={meta.dotClass}
          tail={
            index === 0 && drivers.length > 0 ? (
              <span className="truncate">
                {t("Drag a move onto a driver to check the pairing before it's assigned")}
              </span>
            ) : (
              <span className="truncate">{t(meta.description)}</span>
            )
          }
        />
      ),
    });
    for (const span of group.moves) {
      items.push({
        key: `move:${span.id}`,
        height: ROW_HEIGHT,
        node: (
          <MoveRow
            span={span}
            scale={scale}
            selected={span.id === selectedMoveId}
            onSelectMove={onSelectMove}
          />
        ),
      });
    }
  });
  items.push({
    key: "group:drivers",
    height: GROUP_HEIGHT,
    header: true,
    node: (
      <TimelineGroupRow
        scale={scale}
        label={t("Drivers")}
        count={drivers.length}
        tail={
          drivers.length > 0 ? (
            <span className="truncate">
              {t("Free time is labeled; a red outline is an overlap")}
            </span>
          ) : null
        }
      />
    ),
  });
  if (lanes.drivers.length === 0) {
    items.push({
      key: "driver:none",
      height: ROW_HEIGHT,
      node: (
        <TimelineRow
          scale={scale}
          height={ROW_HEIGHT}
          label={
            <TimelineLaneLabel
              title={t("No drivers to plan")}
              meta={
                <span className="truncate">{t("No assignable driver matches the filters")}</span>
              }
            />
          }
          lane={null}
        />
      ),
    });
  }
  for (const lane of lanes.drivers) {
    items.push({
      key: `driver:${lane.driver.workerId}`,
      height: laneHeight(lane.packed.rows),
      node: (
        <DriverLaneRow
          lane={lane}
          scale={scale}
          selected={lane.driver.workerId === selectedWorkerId}
          onSelectDriver={onSelectDriver}
          selectedMoveId={selectedMoveId}
          onSelectMove={onSelectMove}
        />
      ),
    });
  }

  return (
    <div className="relative min-h-0 flex-1">
      <PointerHoverCard<HoverItem>
        itemId={(item) => item.id}
        width={HOVER_CARD_WIDTH}
        render={(item) => <BarDetail item={item} scale={scale} />}
      >
        <TimelineCanvas
          scale={scale}
          today={today}
          labelHeading={t("Move or driver")}
          items={items}
          revealKey={revealKeyFor(lanes, selectedMoveId, selectedWorkerId)}
        />
      </PointerHoverCard>
    </div>
  );
}

/** A move still needing a driver: its lane and customer beside a bar it is dragged from. */
function MoveRow({
  span,
  scale,
  selected,
  onSelectMove,
}: {
  span: MoveSpan;
  scale: TimelineScale;
  selected: boolean;
  onSelectMove: (moveId: string) => void;
}) {
  const t = useT();
  const { move } = span;
  const route = t("{0} → {1}", move.originCity, move.destinationCity);

  return (
    <TimelineRow
      height={ROW_HEIGHT}
      scale={scale}
      selected={selected}
      onLabelClick={() => onSelectMove(move.moveId)}
      labelTitle={t(
        "{0}, {1} → {2}, {3}",
        move.originCity,
        move.originState,
        move.destinationCity,
        move.destinationState,
      )}
      label={
        <>
          <TimelineLaneLabel
            title={route}
            meta={
              <>
                <span className="shrink-0 font-mono tabular-nums">{move.proNumber}</span>
                <span className="truncate">{move.customerName}</span>
              </>
            }
          />
          <span
            aria-hidden
            className="border-warning size-5.5 shrink-0 rounded-full border-[1.5px] border-dashed"
          />
        </>
      }
      lane={
        <MoveBar
          span={span}
          scale={scale}
          route={route}
          selected={selected}
          onSelectMove={onSelectMove}
        />
      }
    />
  );
}

/**
 * The move's bar, dashed in its urgency's tone and picked up to be placed on a driver. Its
 * caption says when the pickup is, or that its window has closed.
 */
function MoveBar({
  span,
  scale,
  route,
  selected,
  onSelectMove,
}: {
  span: MoveSpan;
  scale: TimelineScale;
  route: string;
  selected: boolean;
  onSelectMove: (moveId: string) => void;
}) {
  const t = useT();
  const { move } = span;
  const { setNodeRef, attributes, listeners, isDragging } = useDraggable({
    id: `move:${move.moveId}`,
    data: { type: "move", move },
  });
  const drag: BarDragHandle = {
    ref: setNodeRef,
    props: { ...attributes, ...listeners },
    dragging: isDragging,
  };
  const geometry = barGeometry(span.start, span.end, scale);
  if (!geometry) return null;

  const late = move.urgency === "Late";
  const pickup = formatTimelineTime(move.originWindowStart, scale);
  const caption = late
    ? t("Pickup window closed · was {0}", pickup)
    : move.destinationWindowStart > 0
      ? t(
          "Pick up {0} · deliver {1}",
          pickup,
          formatTimelineTime(move.destinationWindowStart, scale),
        )
      : t("Pick up {0}", pickup);

  return (
    <TimelineBarFrame
      id={move.moveId}
      geometry={geometry}
      scale={scale}
      top={(ROW_HEIGHT - BAR_HEIGHT) / 2}
      height={BAR_HEIGHT}
      toneVar={urgencyMeta(move.urgency).colorVar}
      caption={caption}
      captionTone={late ? "late" : "muted"}
      open
      selected={selected}
      label={t("{0}, {1}", move.proNumber, route)}
      onSelect={() => onSelectMove(move.moveId)}
      drag={drag}
      hoverItem={{ id: move.moveId, kind: "move", move } satisfies HoverItem}
    />
  );
}

/** Work a driver holds: tinted in the brand, deeper up to now once it is under way. */
function CommitmentBar({
  span,
  driver,
  scale,
  top,
  nextLeft,
  conflict,
  selected,
  onSelectMove,
}: {
  span: CommitmentSpan;
  driver: DispatchBoardDriver;
  scale: TimelineScale;
  top: number;
  nextLeft?: number;
  conflict: boolean;
  selected: boolean;
  onSelectMove: (moveId: string) => void;
}) {
  const t = useT();
  const { commitment } = span;
  const geometry = barGeometry(span.start, span.end, scale);
  if (!geometry) return null;
  const destination = t("to {0}", commitment.destinationCity);

  return (
    <TimelineBarFrame
      id={commitment.moveId}
      geometry={geometry}
      scale={scale}
      top={top}
      height={BAR_HEIGHT}
      toneVar="--brand"
      caption={[commitment.proNumber, destination].join(" ")}
      captionInside={
        <>
          <span className="mr-1 font-mono font-semibold">{commitment.proNumber}</span>
          {destination}
        </>
      }
      captionTone="plain"
      nextLeft={nextLeft}
      conflict={conflict}
      selected={selected}
      label={t("{0}, {1}", commitment.proNumber, destination)}
      onSelect={() => onSelectMove(commitment.moveId)}
      hoverItem={
        { id: commitment.moveId, kind: "commitment", commitment, driver } satisfies HoverItem
      }
    >
      {commitment.moveStatus === "InTransit" ? (
        <TimelineDrivenFill start={span.start} end={span.end} scale={scale} geometry={geometry} />
      ) : null}
    </TimelineBarFrame>
  );
}

function DriverLaneRow({
  lane,
  scale,
  selected,
  onSelectDriver,
  selectedMoveId,
  onSelectMove,
}: {
  lane: ConsoleDriverLane;
  scale: TimelineScale;
  selected: boolean;
  onSelectDriver: (workerId: string) => void;
} & Selection) {
  const t = useT();
  const { driver, packed } = lane;
  // `active` is what is being dragged; only the lane under it redraws as the pointer moves.
  const { setNodeRef, isOver, active } = useDroppable({
    id: `driver-lane:${driver.workerId}`,
    data: { type: "driver-target", driver },
  });
  const blocked = driver.availability === "Blocked" || !driver.tractorId;
  const hovering = isOver && active?.data.current?.type === "move";
  const name = t("{0} {1}", driver.firstName, driver.lastName);
  const availability = availabilityMeta(driver.availability);
  const hasHours = driver.hosRecordedAt > 0 && !driver.hosIsStale;
  const away = lane.timeOff.map((pto) => ({ start: pto.startDate, end: pto.endDate }));

  return (
    <TimelineRow
      scale={scale}
      height={laneHeight(packed.rows)}
      laneRef={setNodeRef}
      dropState={hovering ? (blocked ? "refused" : "ok") : null}
      selected={selected}
      onLabelClick={() => onSelectDriver(driver.workerId)}
      labelTitle={name}
      label={
        <>
          <Avatar className="size-5.5 shrink-0">
            {driver.profilePicUrl ? <AvatarImage src={driver.profilePicUrl} alt={name} /> : null}
            <AvatarFallback className="text-3xs">
              {workerInitials(driver.firstName, driver.lastName)}
            </AvatarFallback>
          </Avatar>
          <TimelineLaneLabel
            title={name}
            flag={packed.conflicts.size > 0 ? <TimelineConflictFlag /> : null}
            meta={
              <>
                <span className="truncate">{driver.tractorCode || t("No tractor")}</span>
                <span aria-hidden>·</span>
                {driver.availability === "Blocked" || driver.availability === "TimeOff" ? (
                  <span className={cn("shrink-0", availability.labelClass)}>
                    {t(availability.label)}
                  </span>
                ) : hasHours ? (
                  <span
                    title={t("Drive time left today")}
                    className={cn(
                      "shrink-0 font-mono tabular-nums",
                      driver.driveRemainingMs < OUT_OF_HOS_MS
                        ? "text-danger"
                        : driver.driveRemainingMs < LOW_HOS_MS
                          ? "text-warning"
                          : undefined,
                    )}
                  >
                    {t("{0} left", formatClockDurationMs(driver.driveRemainingMs))}
                  </span>
                ) : (
                  <span className="shrink-0">{t(availability.label)}</span>
                )}
              </>
            }
          />
        </>
      }
      lane={
        <>
          {lane.timeOff.map((pto) => {
            const left = scale.x(Math.max(pto.startDate, scale.start));
            const width = scale.x(Math.min(pto.endDate, scale.end)) - left;
            const title = timeOffTitle(t, pto.type);
            return (
              <div
                key={`${pto.startDate}-${pto.endDate}-${pto.type}`}
                title={title}
                className="bg-accent-violet-subtle absolute inset-y-0 flex items-center justify-center overflow-hidden"
                style={{ left, width }}
              >
                {captionWidth(title) <= width ? (
                  <span className="text-accent-violet-on-subtle truncate px-1.5 text-xs">
                    {title}
                  </span>
                ) : null}
              </div>
            );
          })}
          {driver.availability === "Blocked" ? null : (
            <TimelineLaneGaps items={[...lane.commitments, ...away]} scale={scale} />
          )}
          <TimelineLaneBars
            packed={packed}
            scale={scale}
            renderBar={(span, place) => (
              <CommitmentBar
                span={span}
                driver={driver}
                scale={scale}
                top={place.top}
                nextLeft={place.nextLeft}
                conflict={place.conflict}
                selected={span.id === selectedMoveId}
                onSelectMove={onSelectMove}
              />
            )}
          />
        </>
      }
    />
  );
}

/** A bar's hover: which move it is, where it runs and when, and who has it or what to do. */
function BarDetail({ item, scale }: { item: HoverItem; scale: TimelineScale }) {
  const t = useT();
  if (item.kind === "move") {
    const { move } = item;
    const urgency = urgencyMeta(move.urgency);
    return (
      <div className="flex flex-col gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <span className="font-mono font-semibold tabular-nums">{move.proNumber}</span>
          <span className="text-muted-foreground truncate">{move.customerName}</span>
        </div>
        <ol className="flex flex-col gap-1">
          <li className="grid grid-cols-[1rem_minmax(0,1fr)_auto] gap-2">
            <span className="text-muted-foreground font-mono tabular-nums">1</span>
            <span className="truncate">{t("{0}, {1}", move.originCity, move.originState)}</span>
            <span className="text-muted-foreground font-mono tabular-nums">
              {formatTimelineTime(move.originWindowStart, scale)}
            </span>
          </li>
          <li className="grid grid-cols-[1rem_minmax(0,1fr)_auto] gap-2">
            <span className="text-muted-foreground font-mono tabular-nums">2</span>
            <span className="truncate">
              {t("{0}, {1}", move.destinationCity, move.destinationState)}
            </span>
            <span className="text-muted-foreground font-mono tabular-nums">
              {move.destinationWindowStart > 0
                ? formatTimelineTime(move.destinationWindowStart, scale)
                : "—"}
            </span>
          </li>
        </ol>
        <span className="border-subtle flex items-center gap-1.5 border-t pt-1.5">
          <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full", urgency.dotClass)} />
          <span className="font-medium">{t(urgency.label)}</span>
          <span className="text-muted-foreground truncate">{t(urgency.description)}</span>
        </span>
      </div>
    );
  }
  const { commitment, driver } = item;
  return (
    <div className="flex flex-col gap-2">
      <div className="flex min-w-0 items-center gap-2">
        <span className="font-mono font-semibold tabular-nums">{commitment.proNumber}</span>
        <span className="text-muted-foreground truncate">
          {commitment.moveStatus === "InTransit" ? t("In transit") : t("Planned")}
        </span>
      </div>
      <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
        <span className="truncate">
          {t("To {0}, {1}", commitment.destinationCity, commitment.destinationState)}
        </span>
        <span className="text-muted-foreground font-mono tabular-nums">
          {t(
            "{0} to {1}",
            formatTimelineTime(commitment.windowStart, scale),
            formatTimelineTime(commitment.windowEnd, scale),
          )}
        </span>
      </div>
      <span className="text-muted-foreground border-subtle border-t pt-1.5">
        {[t("{0} {1}", driver.firstName, driver.lastName), driver.tractorCode]
          .filter(Boolean)
          .join(" · ")}
      </span>
    </div>
  );
}
