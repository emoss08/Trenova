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
import { SLOT_SECONDS } from "./outlook-model";
import { slotRange } from "./outlook-parts";

export const TIDE_HEIGHT = 148;
const TOP = 16;
const LINES_BOTTOM = 92;
const BASE = 116;
const BAR_MAX = 26;
const BAR_WIDTH = 22;

export type BoardColumns = { width: number; label: number; value: number; count: number };

export function columnStep({ width, label, value, count }: BoardColumns): number {
  return Math.max(0, width - label - value) / Math.max(count, 1);
}

type TideSlot = { index: number; slot: FreightNetworkSlot };

function niceTicks(min: number, max: number, count: number): number[] {
  const span = Math.max(max - min, 1);
  const raw = span / count;
  const magnitude = 10 ** Math.floor(Math.log10(raw));
  const step = [1, 2, 5, 10].map((m) => m * magnitude).find((s) => s >= raw) ?? raw;
  const ticks: number[] = [];
  for (let value = Math.ceil(min / step) * step; value <= max; value += step) ticks.push(value);
  return ticks;
}

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
      aria-label={slotRange(item.slot.start)}
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
      <HoverCardHeading>{slotRange(slot.start)}</HoverCardHeading>
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
 * The network across the next two days, drawn on the board's own columns: loads likely (with
 * their usual range) against drivers expected, and underneath, loads with no truck nearby and
 * idle drivers once each area is counted alone. A column pins its window.
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
  if (count === 0 || width === 0) {
    return <div style={{ height: TIDE_HEIGHT }} />;
  }

  const step = columnStep(columns);
  const x = (index: number) => label + step * (index + 0.5);
  const right = width - value;
  const lowest = Math.min(...slots.map((s) => Math.min(s.needLow, s.need, s.drivers)));
  const highest = Math.max(...slots.map((s) => Math.max(s.needHigh, s.need, s.drivers)));
  const ticks = niceTicks(Math.floor(lowest), Math.ceil(highest), 3);
  const yMin = Math.min(ticks[0] ?? lowest, lowest);
  const yMax = Math.max(ticks[ticks.length - 1] ?? highest, highest);
  const y = (v: number) => TOP + (1 - (v - yMin) / Math.max(yMax - yMin, 1)) * (LINES_BOTTOM - TOP);
  const barTop = Math.max(1, ...slots.map((s) => Math.max(s.uncovered, s.idle)));
  const bar = (v: number) => (v / barTop) * BAR_MAX;
  const barWidth = Math.min(BAR_WIDTH, step * 0.4);

  const items: TideSlot[] = slots.map((slot, index) => ({ index, slot }));
  const band = areaPath<TideSlot>()
    .x((d) => x(d.index))
    .y0((d) => y(d.slot.needLow))
    .y1((d) => y(d.slot.needHigh))
    .curve(curveMonotoneX)(items);
  const loads = linePath<TideSlot>()
    .x((d) => x(d.index))
    .y((d) => y(d.slot.need))
    .curve(curveMonotoneX)(items);
  const drivers = linePath<TideSlot>()
    .x((d) => x(d.index))
    .y((d) => y(d.slot.drivers))
    .curve(curveMonotoneX)(items);

  const first = outlook.slotStarts[0] ?? now;
  const nowX = label + step * Math.min(Math.max((now - first) / SLOT_SECONDS, 0), count);
  const last = items[count - 1];
  const loadsY = y(last.slot.need);
  const driversY = y(last.slot.drivers);
  const apart = Math.abs(loadsY - driversY) < 14 ? (loadsY < driversY ? -7 : 7) : 0;

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
        {ticks.map((tick) => (
          <g key={tick}>
            <line
              x1={label}
              x2={right}
              y1={y(tick)}
              y2={y(tick)}
              className="stroke-border-subtle"
            />
            <text
              x={label - 8}
              y={y(tick) + 3.5}
              textAnchor="end"
              className="fill-foreground-subtle text-2xs"
            >
              {formatNumber(tick)}
            </text>
          </g>
        ))}
        <text x={14} y={TOP + 4} className="fill-muted-foreground text-2xs">
          {t("Network")}
        </text>
        <text x={14} y={BASE - 8} className="fill-warning-foreground text-2xs">
          {t("No truck nearby")}
        </text>
        <text x={14} y={BASE + 14} className="fill-accent-sky-on-subtle text-2xs">
          {t("Idle drivers")}
        </text>
        <line x1={label} x2={right} y1={BASE} y2={BASE} className="stroke-border" />
        {band ? <path d={band} className="fill-warning/20" /> : null}
        {loads ? (
          <path
            d={loads}
            className="fill-none stroke-warning-foreground"
            strokeWidth={2}
            strokeLinecap="round"
          />
        ) : null}
        {drivers ? (
          <path
            d={drivers}
            className="fill-none stroke-foreground"
            strokeWidth={2}
            strokeLinecap="round"
          />
        ) : null}
        <text
          x={x(count - 1) + 14}
          y={loadsY + 4 + apart}
          className="fill-warning-foreground text-xs font-medium"
        >
          {t("Loads")}
        </text>
        <text
          x={x(count - 1) + 14}
          y={driversY + 4 - apart}
          className="fill-foreground text-xs font-medium"
        >
          {t("Drivers")}
        </text>
        {items.map(({ index, slot }) => (
          <g key={slot.start}>
            {slot.uncovered > 0.2 ? (
              <rect
                x={x(index) - barWidth / 2}
                y={BASE - 1 - bar(slot.uncovered)}
                width={barWidth}
                height={bar(slot.uncovered)}
                rx={2}
                className="fill-warning"
              />
            ) : null}
            {slot.idle > 0.2 ? (
              <rect
                x={x(index) - barWidth / 2}
                y={BASE + 1}
                width={barWidth}
                height={bar(slot.idle)}
                rx={2}
                className="fill-accent-sky"
              />
            ) : null}
          </g>
        ))}
        {horizonSlots < count ? (
          <rect
            x={label + step * horizonSlots}
            y={0}
            width={step * (count - horizonSlots)}
            height={TIDE_HEIGHT}
            fill={`url(#${hatch})`}
            className="pointer-events-none"
          />
        ) : null}
        <line
          x1={nowX}
          x2={nowX}
          y1={TOP - 6}
          y2={TIDE_HEIGHT}
          className="stroke-brand"
          strokeWidth={1.5}
        />
        <text x={nowX + 5} y={TOP + 2} className="fill-brand text-2xs font-medium">
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
