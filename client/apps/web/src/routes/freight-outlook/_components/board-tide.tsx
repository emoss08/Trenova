import {
  HoverCardHeading,
  HoverCardRow,
  PointerHoverCard,
  usePointerHoverTarget,
} from "@/components/pointer-hover-card";
import type { FreightNetworkSlot, FreightOutlook } from "@/lib/graphql/freight-outlook";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { area as areaPath, curveMonotoneX, line as linePath } from "d3-shape";
import { useId } from "react";
import {
  BARS_BASE,
  DRIVERS_LANE,
  LOADS_LANE,
  NOW_LABEL_Y,
  TIDE_HEIGHT,
  barHeights,
  columnStep,
  laneScale,
  laneTop,
  nowLabelAnchor,
  type BoardColumns,
  type Lane,
} from "./board-tide-model";
import { positionAmong } from "./outlook-model";
import { slotRange } from "./outlook-parts";

/** The space between the plot's left edge and the text beside it. */
const GUTTER_GAP = 10;
/** The least a column is wide before its bars carry their numbers. */
const NUMBERED_STEP = 24;
const BAR_WIDTH = 22;
/** How far the shading of a window and of the hours past the range reach around the lanes. */
const LANE_MARGIN = 6;
/** The count at which the whole network is short of drivers in a window. */
const NETWORK_SHORT = 0.5;

type TideSlot = { index: number; slot: FreightNetworkSlot };
/** A point on a line: a window's value, held from the plot's edge to the window's middle. */
type TidePoint = { px: number; slot: FreightNetworkSlot };

function TideColumn({
  item,
  x,
  width,
  pinned,
  onPin,
}: {
  item: TideSlot;
  x: number;
  width: number;
  pinned: boolean;
  onPin: (index: number) => void;
}) {
  const hover = usePointerHoverTarget(item, String(item.index));
  return (
    <rect
      x={x}
      y={0}
      width={width}
      height={TIDE_HEIGHT}
      role="button"
      tabIndex={0}
      aria-pressed={pinned}
      aria-label={slotRange(item.slot)}
      className={cn(
        "cursor-pointer ui-focus-ring outline-none",
        pinned ? "fill-brand/8" : "fill-transparent hover:fill-foreground/4",
      )}
      onClick={() => onPin(item.index)}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onPin(item.index);
        }
      }}
      {...hover}
    />
  );
}

function TideDetail({ item, pinned }: { item: TideSlot; pinned: boolean }) {
  const t = useT();
  const { slot } = item;
  return (
    <>
      <HoverCardHeading>{slotRange(slot)}</HoverCardHeading>
      <HoverCardRow
        label={t("Still need a driver")}
        value={t("{0}–{1}", formatNumber(slot.needLow), formatNumber(slot.needHigh))}
        swatchClass="bg-warning-foreground"
      />
      <HoverCardRow
        label={t("Loads likely, with those covered")}
        value={t("{0}–{1}", formatNumber(slot.likelyLow), formatNumber(slot.likelyHigh))}
      />
      <HoverCardRow
        label={t("Drivers expected")}
        value={formatNumber(Math.round(slot.drivers))}
        swatchClass="bg-foreground"
      />
      {slot.short >= NETWORK_SHORT ? (
        <HoverCardRow
          label={t("More loads than drivers")}
          value={formatNumber(Math.round(slot.short))}
          swatchClass="bg-danger"
        />
      ) : null}
      <HoverCardRow
        label={t("No truck nearby")}
        value={formatNumber(Math.round(slot.uncovered))}
        swatchClass="bg-warning"
      />
      <HoverCardRow
        label={t("Idle drivers")}
        value={formatNumber(Math.round(slot.idle))}
        swatchClass="bg-accent-sky"
      />
      <p className="mt-1 text-2xs text-muted-foreground">
        {pinned
          ? t("Click again for the whole range.")
          : t("Click to read the board at this hour.")}
      </p>
    </>
  );
}

/**
 * The network across the next two days, drawn on the board's own columns: loads still needing
 * a driver (with their usual range) and drivers free, each in a lane with its own scale, and
 * underneath, loads with no truck nearby and idle drivers once each area is counted alone. A
 * window where the loads outrun the drivers is shaded across both lanes, which is what the
 * two lines crossing would have said on one scale. A column pins its window.
 *
 * Everything that names a mark sits in the column to the left of the plot, beside the mark it
 * names; what each lane's top line stands for is written at its right end, with its unit;
 * and the "now" label has a strip of its own above the lanes. No text is drawn over the data
 * or under the shading of the hours past the chosen range.
 */
export function BoardTide({
  outlook,
  columns,
  horizonSlots,
  pinned,
  onPin,
}: {
  outlook: FreightOutlook;
  columns: BoardColumns;
  horizonSlots: number;
  pinned: number | null;
  onPin: (index: number | null) => void;
}) {
  const t = useT();
  const now = useNow("minute");
  const hatch = useId();
  const slots = outlook.network;
  const count = slots.length;
  const { width, label, value } = columns;
  const first = slots[0];
  const last = slots[count - 1];
  if (!first || !last || width === 0) {
    return <div style={{ height: TIDE_HEIGHT }} />;
  }

  const step = columnStep(columns);
  const x = (index: number) => label + step * (index + 0.5);
  const right = width - value;
  const loadsTop = laneTop(Math.max(...slots.map((s) => Math.max(s.needHigh, s.need))));
  const driversTop = laneTop(Math.max(...slots.map((s) => s.drivers)));
  const loadsAt = laneScale(LOADS_LANE, loadsTop);
  const driversAt = laneScale(DRIVERS_LANE, driversTop);
  // Each lane's top line says what it stands for, with its unit, and its bottom line is
  // zero. Bare numbers down the column read as a list, not as two scales.
  const lanes: { lane: Lane; scale: string }[] = [
    {
      lane: LOADS_LANE,
      scale: t("{0, plural, one {# load} other {# loads}}", loadsTop),
    },
    {
      lane: DRIVERS_LANE,
      scale: t("{0, plural, one {# driver} other {# drivers}}", driversTop),
    },
  ];
  const shade = { y: LOADS_LANE.top - LANE_MARGIN, bottom: DRIVERS_LANE.bottom + LANE_MARGIN };
  const shortBar = barHeights(slots.map((s) => s.uncovered));
  const idleBar = barHeights(slots.map((s) => s.idle));
  const barWidth = Math.min(BAR_WIDTH, step * 0.4);
  const numbered = step >= NUMBERED_STEP;

  const items: TideSlot[] = slots.map((slot, index) => ({ index, slot }));
  // Each value stands for a whole window, so the lines hold it out to the plot's edges
  // instead of stopping half a column short of them.
  const points: TidePoint[] = [
    { px: label, slot: first },
    ...items.map(({ index, slot }) => ({ px: x(index), slot })),
    { px: right, slot: last },
  ];
  const band = areaPath<TidePoint>()
    .x((d) => d.px)
    .y0((d) => loadsAt(d.slot.needLow))
    .y1((d) => loadsAt(d.slot.needHigh))
    .curve(curveMonotoneX)(points);
  const loads = linePath<TidePoint>()
    .x((d) => d.px)
    .y((d) => loadsAt(d.slot.need))
    .curve(curveMonotoneX)(points);
  const drivers = linePath<TidePoint>()
    .x((d) => d.px)
    .y((d) => driversAt(d.slot.drivers))
    .curve(curveMonotoneX)(points);

  const gutter = label - GUTTER_GAP;
  const nowX = label + step * Math.min(positionAmong(slots, now), count);
  const nowAnchor = nowLabelAnchor(nowX, right);

  return (
    <PointerHoverCard<TideSlot>
      itemId={(item) => String(item.index)}
      render={(item) => <TideDetail item={item} pinned={pinned === item.index} />}
    >
      <svg
        width={width}
        height={TIDE_HEIGHT}
        className="block overflow-visible"
        role="img"
        aria-label={t(
          "Network loads still needing a driver against drivers free, with loads that have no truck nearby and idle drivers, every four hours",
        )}
      >
        <defs>
          <pattern
            id={hatch}
            width={6}
            height={6}
            patternUnits="userSpaceOnUse"
            patternTransform="rotate(45)"
          >
            <rect width={6} height={6} className="fill-background" opacity={0.6} />
            <line x1={0} y1={0} x2={0} y2={6} className="stroke-border" strokeWidth={1.5} />
          </pattern>
        </defs>
        <text x={14} y={NOW_LABEL_Y} className="fill-muted-foreground text-2xs">
          {t("Network")}
        </text>
        {lanes.map(({ lane, scale }) => (
          <g key={lane.top}>
            <line
              x1={label}
              x2={right}
              y1={lane.top}
              y2={lane.top}
              className="stroke-border-subtle"
            />
            <line
              x1={label}
              x2={right}
              y1={lane.bottom}
              y2={lane.bottom}
              className="stroke-border-subtle"
            />
            <text
              x={right + GUTTER_GAP}
              y={lane.top + 3.5}
              className="fill-foreground-subtle text-2xs tabular-nums"
            >
              {scale}
            </text>
          </g>
        ))}
        {items.map(({ index, slot }) =>
          slot.short >= NETWORK_SHORT ? (
            <rect
              key={`short-${slot.start}`}
              x={label + step * index}
              y={shade.y}
              width={step}
              height={shade.bottom - shade.y}
              className="fill-danger/10"
            />
          ) : null,
        )}
        <text
          x={gutter}
          y={driversAt(first.drivers) + 3.5}
          textAnchor="end"
          className="fill-foreground text-2xs font-medium"
        >
          {t("Drivers free")}
        </text>
        <text
          x={gutter}
          y={loadsAt(first.need) + 3.5}
          textAnchor="end"
          className="fill-warning-foreground text-2xs font-medium"
        >
          {t("Loads needing a driver")}
        </text>
        <text
          x={gutter}
          y={BARS_BASE - 6}
          textAnchor="end"
          className="fill-warning-foreground text-2xs"
        >
          {t("No truck nearby")}
        </text>
        <text
          x={gutter}
          y={BARS_BASE + 13}
          textAnchor="end"
          className="fill-accent-sky-on-subtle text-2xs"
        >
          {t("Idle drivers")}
        </text>
        <line x1={label} x2={right} y1={BARS_BASE} y2={BARS_BASE} className="stroke-border" />
        {band ? <path d={band} className="fill-warning/20" /> : null}
        {loads ? (
          <path d={loads} className="fill-none stroke-warning-foreground" strokeWidth={2} />
        ) : null}
        {drivers ? (
          <path d={drivers} className="fill-none stroke-foreground" strokeWidth={2} />
        ) : null}
        {items.map(({ index, slot }) => {
          const short = shortBar(slot.uncovered);
          const idle = idleBar(slot.idle);
          return (
            <g key={slot.start}>
              {short > 0 ? (
                <>
                  <rect
                    x={x(index) - barWidth / 2}
                    y={BARS_BASE - 1 - short}
                    width={barWidth}
                    height={short}
                    rx={2}
                    className="fill-warning"
                  />
                  {numbered ? (
                    <text
                      x={x(index)}
                      y={BARS_BASE - 4 - short}
                      textAnchor="middle"
                      className="fill-warning-foreground text-2xs font-medium tabular-nums"
                    >
                      {formatNumber(Math.round(slot.uncovered))}
                    </text>
                  ) : null}
                </>
              ) : null}
              {idle > 0 ? (
                <>
                  <rect
                    x={x(index) - barWidth / 2}
                    y={BARS_BASE + 1}
                    width={barWidth}
                    height={idle}
                    rx={2}
                    className="fill-accent-sky"
                  />
                  {numbered ? (
                    <text
                      x={x(index)}
                      y={BARS_BASE + 11 + idle}
                      textAnchor="middle"
                      className="fill-accent-sky-on-subtle text-2xs tabular-nums"
                    >
                      {formatNumber(Math.round(slot.idle))}
                    </text>
                  ) : null}
                </>
              ) : null}
            </g>
          );
        })}
        {horizonSlots < count ? (
          <rect
            x={label + step * horizonSlots}
            y={shade.y}
            width={step * (count - horizonSlots)}
            height={TIDE_HEIGHT - shade.y}
            fill={`url(#${hatch})`}
            className="pointer-events-none"
          />
        ) : null}
        <line
          x1={nowX}
          x2={nowX}
          y1={NOW_LABEL_Y + 4}
          y2={TIDE_HEIGHT}
          className="stroke-brand"
          strokeWidth={1.5}
        />
        <text
          x={nowX}
          y={NOW_LABEL_Y}
          textAnchor={nowAnchor}
          className="fill-brand text-2xs font-medium"
        >
          {t("Now {0}", formatUnixInUserTimezone(now, { hour: "numeric", minute: "2-digit" }))}
        </text>
        {items.map((item) => (
          <TideColumn
            key={`hit-${item.slot.start}`}
            item={item}
            x={label + step * item.index}
            width={step}
            pinned={pinned === item.index}
            onPin={(index) => onPin(pinned === index ? null : index)}
          />
        ))}
      </svg>
    </PointerHoverCard>
  );
}
