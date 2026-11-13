import { tourTarget } from "@/components/tour/tour-model";
import { OUTLOOK_TOUR_TARGET } from "./outlook-tour";
import {
  HoverCardHeading,
  HoverCardRow,
  PointerHoverCard,
  usePointerHoverTarget,
} from "@/components/pointer-hover-card";
import type {
  FreightArea,
  FreightMove,
  FreightOutlook,
  FreightSlot,
} from "@/lib/graphql/freight-outlook";
import { ChevronDownIcon } from "@trenova/shared/components/icons";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import { formatNumber } from "@trenova/shared/i18n/format";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn, formatCentsWhole } from "@trenova/shared/lib/utils";
import { useLayoutEffect, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { BoardTide } from "./board-tide";
import { TIDE_HEIGHT, type BoardColumns } from "./board-tide-model";
import {
  balanceFigure,
  balanceOf,
  laneGroups,
  movesIn,
  moveTrail,
  paceReadings,
  startsDay,
  type Lane,
  type Peak,
} from "./outlook-model";
import {
  Strong,
  balanceStyle,
  balanceTextClass,
  dayKey,
  dayLabel,
  hourLabel,
  slotLabel,
  slotRange,
} from "./outlook-parts";

/** The area column and the one-more-driver column; the windows share what is left. */
/**
 * The name column and the value column, either side of the windows. They are as narrow as
 * what they hold allows (an area's name, the chart's line names, a price), because every
 * pixel they give up goes to the twelve windows between them.
 */
const LABEL_WIDTH = 192;
const VALUE_WIDTH = 100;
/** The network is short when it is three or more loads past every driver it has. */
const NETWORK_SHORT = 3;

const LANE_LABELS = defineLabels({
  short: "Short of trucks",
  tight: "Could run short",
  spare: "Trucks to spare",
  even: "Balanced",
});

const LANE_DOT: Record<Lane, string> = {
  short: "bg-warning",
  tight: "bg-warning/50",
  spare: "bg-accent-sky",
  even: "bg-border-strong",
};

const NOTE_TONE = {
  short: "text-warning-foreground",
  spare: "text-accent-sky-on-subtle",
  even: "text-foreground-subtle",
} as const;

function gridStyle(count: number): CSSProperties {
  return {
    gridTemplateColumns: `${LABEL_WIDTH}px repeat(${count}, minmax(0, 1fr)) ${VALUE_WIDTH}px`,
  };
}

type Cell = { area: FreightArea; slot: FreightSlot; index: number };

function CellMark({
  cell,
  beyond,
  pinned,
  dayStart,
  mark,
}: {
  cell: Cell;
  beyond: boolean;
  pinned: boolean;
  dayStart: boolean;
  mark: "from" | "to" | null;
}) {
  const hover = usePointerHoverTarget(cell, `${cell.area.cell}:${cell.index}`);
  const balance = balanceOf(cell.slot);
  return (
    <span
      data-cell={`${cell.area.cell}:${cell.index}`}
      className={cn(
        "mx-px grid h-5 place-items-center rounded-[3px] font-mono text-2xs font-medium tabular-nums transition-shadow",
        balance.kind === "even" ? "bg-foreground/4 text-transparent" : balanceTextClass(balance),
        dayStart && "ml-0.75",
        beyond && "opacity-35",
        pinned && "ring-[1.5px] ring-brand",
        mark === "from" &&
          "ring-[1.5px] ring-accent-sky-on-subtle ring-offset-2 ring-offset-background",
        mark === "to" &&
          "ring-[1.5px] ring-warning-foreground ring-offset-2 ring-offset-background",
      )}
      style={balanceStyle(balance)}
      {...hover}
    >
      {balanceFigure(balance)}
    </span>
  );
}

function CellDetail({ cell }: { cell: Cell }) {
  const t = useT();
  const { slot } = cell;
  const balance = balanceOf(slot);
  return (
    <>
      <HoverCardHeading>{t("{0} · {1}", cell.area.label, slotRange(slot))}</HoverCardHeading>
      <HoverCardRow
        label={t("Loads likely")}
        value={
          slot.likelyLow === slot.likelyHigh
            ? formatNumber(slot.likelyLow)
            : t("{0}–{1}", formatNumber(slot.likelyLow), formatNumber(slot.likelyHigh))
        }
      />
      <HoverCardRow label={t("Already booked")} value={formatNumber(slot.booked)} />
      <HoverCardRow label={t("Drivers expected")} value={formatNumber(slot.drivers)} />
      <p className="mt-1 text-2xs text-muted-foreground">
        {balance.kind === "short"
          ? t(
              "About {0, plural, one {# load} other {# loads}} more than drivers.",
              Math.round(balance.amount),
            )
          : balance.kind === "spare"
            ? t(
                "About {0, plural, one {# driver} other {# drivers}} more than freight.",
                Math.round(balance.amount),
              )
            : t("Drivers and freight roughly match.")}
      </p>
    </>
  );
}

function noteOf(
  area: FreightArea,
  peak: Peak,
  outlook: FreightOutlook,
  indexes: readonly number[],
  t: TranslateFn,
): { tone: "short" | "spare" | "even"; text: string } {
  const when = slotLabel(outlook.slotStarts[peak.index] ?? 0);
  const moves = movesIn(outlook.moves, outlook.slotStarts, indexes);
  if (peak.kind === "short") {
    const lead = t("{0} short {1}", Math.round(peak.amount), when);
    const behind = paceReadings(outlook.pace, indexes).find(
      (reading) => reading.pace.cell === area.cell && reading.standing === "Behind",
    );
    if (behind) {
      return { tone: "short", text: t("{0} · {1} behind", lead, behind.pace.customerName) };
    }
    const customer = area.customers[0];
    return { tone: "short", text: customer ? t("{0} · {1}", lead, customer.name) : lead };
  }
  if (peak.kind === "spare") {
    const lead = t("{0} to spare {1}", Math.round(peak.amount), when);
    const outgoing = moves.find(
      (move) => move.fromCell === area.cell && move.kind !== "Call" && move.toCell !== area.cell,
    );
    return {
      tone: "spare",
      text: outgoing
        ? t("{0} · {1} → {2}", lead, outgoing.workerName.split(" ")[0] ?? "", outgoing.toLabel)
        : t("{0} · nothing short in reach", lead),
    };
  }
  return { tone: "even", text: t("Drivers match the freight") };
}

function BoardKey() {
  const t = useT();
  const ramp = (kind: "short" | "spare") =>
    ([1, 2, 3] as const).map((level) => (
      <span
        key={level}
        aria-hidden
        className="h-2.5 w-3 rounded-xs"
        style={balanceStyle({ kind, level, amount: level })}
      />
    ));
  return (
    <span className="flex flex-wrap items-center gap-3 text-2xs text-foreground-subtle">
      <span className="inline-flex items-center gap-1.25">
        <span className="inline-flex gap-0.5">{ramp("short")}</span>
        {t("Short")}
      </span>
      <span className="inline-flex items-center gap-1.25">
        <span className="inline-flex gap-0.5">{ramp("spare")}</span>
        {t("Spare")}
      </span>
    </span>
  );
}

/**
 * Why one more driver is worth nothing here, rather than a blank: the forecast prices a driver by
 * the chance freight outruns the drivers already there, so an area with drivers to spare (or
 * exactly enough) prices at nothing, and an area it has no revenue for cannot be priced at all.
 */
function UnpricedValue({ area, kind }: { area: FreightArea; kind: "short" | "spare" | "even" }) {
  const t = useT();
  const label = kind === "short" ? t("No price") : t("Not needed");
  const why =
    kind === "spare"
      ? t(
          "{0} has more drivers than freight here, so one more driver ending here would likely sit idle.",
          area.label,
        )
      : kind === "even"
        ? t(
            "Drivers already match the freight in {0}, so one more would likely sit idle.",
            area.label,
          )
        : t(
            "The forecast has no revenue history for {0} yet, so it can't price one more driver there.",
            area.label,
          );
  return (
    <Tooltip>
      <TooltipTrigger
        render={<span />}
        className="cursor-help text-xs font-normal text-foreground-subtle underline decoration-border-strong decoration-dotted underline-offset-4"
      >
        {label}
      </TooltipTrigger>
      <TooltipContent side="left" className="max-w-64">
        {why}
      </TooltipContent>
    </Tooltip>
  );
}

function AreaRow({
  area,
  note,
  count,
  horizonSlots,
  pinned,
  slotStarts,
  selected,
  dimmed,
  marks,
  onOpen,
}: {
  area: FreightArea;
  note: { tone: "short" | "spare" | "even"; text: string };
  count: number;
  horizonSlots: number;
  pinned: number | null;
  slotStarts: readonly number[];
  selected: boolean;
  dimmed: boolean;
  marks: Map<number, "from" | "to">;
  onOpen: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onOpen}
      aria-pressed={selected}
      className={cn(
        "relative grid min-h-7 w-full cursor-pointer items-center border-t border-border-subtle text-left ui-inset-focus-ring transition-[opacity,background-color] duration-150",
        selected ? "bg-brand/6" : "hover:bg-surface-hover",
        dimmed && "opacity-30",
        selected &&
          "before:absolute before:inset-y-1.5 before:left-0 before:w-0.75 before:rounded-r-sm before:bg-brand",
      )}
      style={gridStyle(count)}
    >
      <span className="flex min-w-0 items-baseline gap-2 py-0.75 pr-2.5 pl-3.5">
        <span className="shrink-0 text-sm font-medium">{area.label}</span>
        <span title={note.text} className={cn("min-w-0 truncate text-2xs", NOTE_TONE[note.tone])}>
          {note.text}
        </span>
      </span>
      {area.slots.map((slot, index) => (
        <CellMark
          key={slot.start}
          cell={{ area, slot, index }}
          beyond={index >= horizonSlots}
          pinned={pinned === index}
          dayStart={index > 0 && startsDay(slotStarts, index, dayKey)}
          mark={marks.get(index) ?? null}
        />
      ))}
      <span className="pr-3.5 text-right text-sm font-medium tabular-nums">
        {area.valueCents > 0 ? (
          formatCentsWhole(area.valueCents)
        ) : (
          <UnpricedValue area={area} kind={note.tone} />
        )}
      </span>
    </button>
  );
}

type Wire = { path: string; head: string; x: number; y: number; label: string };

function wireBetween(
  content: HTMLElement,
  from: HTMLElement,
  to: HTMLElement,
  label: string,
): Wire {
  const box = content.getBoundingClientRect();
  const a = from.getBoundingClientRect();
  const b = to.getBoundingClientRect();
  const fx = a.left + a.width / 2 - box.left;
  const fy = a.top + a.height / 2 - box.top;
  const tx = b.left + b.width / 2 - box.left;
  const ty = b.top + b.height / 2 - box.top;
  const up = ty < fy ? -1 : 1;
  const sy = fy + up * (a.height / 2 + 3);
  const ey = ty - up * (b.height / 2 + 5);
  const bend = Math.max(36, Math.abs(tx - fx) * 0.5);
  return {
    path: `M${fx},${sy} C${fx + bend * 0.2},${sy + up * 30} ${tx - bend * 0.2},${ey - up * 30} ${tx},${ey}`,
    head: `M${tx - 4.5},${ey - up * 6} L${tx},${ey} L${tx + 4.5},${ey - up * 6}`,
    x: (fx + tx) / 2,
    y: (sy + ey) / 2,
    label,
  };
}

/**
 * The board: the network's tide and every area's next twelve windows on one time axis, so a
 * window is a column read top to bottom. Areas group by what they need; a move pointed at in
 * the brief is drawn from where its driver frees up to the window it fills.
 */
export function OutlookBoard({
  outlook,
  indexes,
  horizonSlots,
  pinned,
  onPin,
  selectedCell,
  hotMove,
  onOpenArea,
}: {
  outlook: FreightOutlook;
  indexes: readonly number[];
  horizonSlots: number;
  pinned: number | null;
  onPin: (index: number | null) => void;
  selectedCell: number | null;
  hotMove: FreightMove | null;
  onOpenArea: (cell: number, slotStart: number) => void;
}) {
  const t = useT();
  const rt = useRichT();
  const [tideRef, width] = useElementWidth<HTMLDivElement>();
  const contentRef = useRef<HTMLDivElement>(null);
  const [folded, setFolded] = useState<Record<Lane, boolean>>({
    short: false,
    tight: false,
    spare: false,
    even: true,
  });
  const [wire, setWire] = useState<Wire | null>(null);
  const [settled, setSettled] = useState(0);
  const count = outlook.slotStarts.length;
  const columns: BoardColumns = { width, label: LABEL_WIDTH, value: VALUE_WIDTH, count };
  const groups = laneGroups(outlook.areas, indexes);
  const first = outlook.slotStarts[0] ?? 0;
  const trail = hotMove ? moveTrail(hotMove, outlook.network) : null;
  const wireLabel =
    hotMove && hotMove.kind !== "Call" ? t("+{0}", formatCentsWhole(hotMove.netCents)) : "";

  const fromCell = trail?.fromCell ?? null;
  const fromIndex = trail?.fromIndex ?? null;
  const toCell = trail?.toCell ?? null;
  const toIndex = trail?.toIndex ?? null;
  const layoutKey = `${width}|${JSON.stringify(folded)}|${settled}|${indexes.join(",")}|${outlook.generatedAt}`;

  useLayoutEffect(() => {
    const content = contentRef.current;
    if (!content || fromCell === null || toCell === null || toIndex === null) {
      setWire(null);
      return;
    }
    const shown = (selector: string) => {
      const cell = content.querySelector<HTMLElement>(selector);
      return cell && !cell.closest("[inert]") ? cell : null;
    };
    const from = shown(`[data-cell="${fromCell}:${fromIndex ?? 0}"]`);
    const to = shown(`[data-cell="${toCell}:${toIndex}"]`);
    setWire(from && to ? wireBetween(content, from, to, wireLabel) : null);
  }, [fromCell, fromIndex, toCell, toIndex, wireLabel, layoutKey]);

  const network = outlook.network.filter((_, index) => indexes.includes(index));
  const worstNetwork = Math.max(0, ...network.map((slot) => slot.short));
  const worstPlaced = Math.max(0, ...network.map((slot) => Math.round(slot.uncovered)));
  const marksFor = (cell: number) => {
    const marks = new Map<number, "from" | "to">();
    if (!trail) return marks;
    if (trail.fromCell === cell && trail.fromIndex !== null) marks.set(trail.fromIndex, "from");
    if (trail.toCell === cell && trail.toIndex !== null) marks.set(trail.toIndex, "to");
    return marks;
  };
  const strong = (chunks: ReactNode) => <Strong>{chunks}</Strong>;

  return (
    <section
      aria-label={t("Loads against drivers")}
      className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-border-subtle bg-background"
    >
      <div className="flex min-h-0 flex-1 flex-col max-md:overflow-x-auto">
        <div className="flex min-h-0 flex-1 flex-col max-md:min-w-180">
          <div className="shrink-0 border-b border-border-subtle">
            <header className="flex flex-wrap items-center gap-x-3.5 gap-y-1.5 px-3.5 pt-2.5">
              <h2 className="text-sm font-semibold">{t("Loads against drivers")}</h2>
              <p className="mr-auto text-xs text-muted-foreground">
                {worstNetwork < NETWORK_SHORT
                  ? rt(
                      "<b>Enough drivers overall</b>, yet at the worst hour {0, plural, one {# load has} other {# loads have}} no truck nearby: where drivers end up, not how many.",
                      { b: strong },
                      worstPlaced,
                    )
                  : rt(
                      "Freight outruns the fleet by about <b>{0, plural, one {# load} other {# loads}}</b> at the peak.",
                      { b: strong },
                      Math.round(worstNetwork),
                    )}
              </p>
              <BoardKey />
            </header>
            <div
              ref={tideRef}
              style={{ height: TIDE_HEIGHT }}
              {...tourTarget(OUTLOOK_TOUR_TARGET.network)}
            >
              <BoardTide
                outlook={outlook}
                columns={columns}
                horizonSlots={horizonSlots}
                pinned={pinned}
                onPin={onPin}
              />
            </div>
            <div
              className="grid h-7.5 items-center border-t border-border-subtle bg-sunken text-2xs text-foreground-subtle"
              style={gridStyle(count)}
              {...tourTarget(OUTLOOK_TOUR_TARGET.windows)}
            >
              <span className="flex min-w-0 items-center gap-2 pl-3.5">
                {pinned === null ? (
                  t("Area")
                ) : (
                  <>
                    <span className="truncate text-foreground">
                      {slotRange(outlook.network[pinned])}
                    </span>
                    <button
                      type="button"
                      onClick={() => onPin(null)}
                      className="shrink-0 cursor-pointer rounded-sm ui-focus-ring font-medium text-brand"
                    >
                      {t("Show all")}
                    </button>
                  </>
                )}
              </span>
              {outlook.slotStarts.map((start, index) => {
                const dayStart = startsDay(outlook.slotStarts, index, dayKey);
                return (
                  <button
                    key={start}
                    type="button"
                    title={slotRange(outlook.network[index])}
                    aria-pressed={pinned === index}
                    onClick={() => onPin(pinned === index ? null : index)}
                    className={cn(
                      "grid h-full cursor-pointer place-items-center truncate whitespace-nowrap ui-inset-focus-ring",
                      "hover:bg-surface-hover hover:text-foreground",
                      dayStart &&
                        index > 0 &&
                        "text-foreground shadow-[inset_1px_0_0_var(--color-border)]",
                      dayStart && "font-medium text-foreground",
                      pinned === index && "bg-brand-subtle font-medium text-brand",
                      index >= horizonSlots && pinned !== index && "text-foreground-faint",
                    )}
                  >
                    {dayStart ? t("{0} {1}", dayLabel(start), hourLabel(start)) : hourLabel(start)}
                  </button>
                );
              })}
              <span
                className="truncate pr-3.5 text-right"
                title={t("What one more driver ending in the area is worth to you there")}
              >
                {t("One more driver")}
              </span>
            </div>
          </div>

          <ScrollArea
            maskVariant="background"
            maskHeight={18}
            className="min-h-0 flex-1 max-xl:max-h-140"
            viewportClassName="overscroll-contain"
          >
            <PointerHoverCard<Cell>
              itemId={(cell) => `${cell.area.cell}:${cell.index}`}
              render={(cell) => <CellDetail cell={cell} />}
            >
              <div ref={contentRef} className="relative" {...tourTarget(OUTLOOK_TOUR_TARGET.areas)}>
                {groups.map(({ lane, rows }) => {
                  const open = !folded[lane];
                  return (
                    <div key={lane} role="group" aria-label={LANE_LABELS[lane]}>
                      <button
                        type="button"
                        aria-expanded={open}
                        onClick={() =>
                          setFolded((current) => ({ ...current, [lane]: !current[lane] }))
                        }
                        className={cn(
                          "sticky top-0 z-2 flex h-6.5 w-full cursor-pointer items-center gap-2 bg-background/92 px-3.5 text-left text-xs ui-inset-focus-ring backdrop-blur-sm transition-opacity",
                          "after:h-px after:flex-1 after:bg-border-subtle",
                          trail && "opacity-50",
                        )}
                      >
                        <ChevronDownIcon
                          aria-hidden
                          className={cn(
                            "size-3 text-foreground-subtle transition-transform",
                            !open && "-rotate-90",
                          )}
                        />
                        <span aria-hidden className={cn("size-2 rounded-xs", LANE_DOT[lane])} />
                        <span className="font-medium text-foreground">{LANE_LABELS[lane]}</span>
                        <span className="text-foreground-subtle">{formatNumber(rows.length)}</span>
                      </button>
                      {/* Rows stay mounted and fold by their track height, so a group opens and
                          closes in one motion both ways. */}
                      <div
                        inert={!open}
                        onTransitionEnd={(event) => {
                          if (event.target === event.currentTarget) setSettled((n) => n + 1);
                        }}
                        className={cn(
                          "grid transition-[grid-template-rows,opacity] duration-320 ease-settle motion-reduce:transition-none",
                          open ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0",
                        )}
                      >
                        <div className="min-h-0 overflow-hidden">
                          {rows.map(({ area, peak }) => {
                            const marks = marksFor(area.cell);
                            return (
                              <AreaRow
                                key={area.cell}
                                area={area}
                                note={noteOf(area, peak, outlook, indexes, t)}
                                count={count}
                                horizonSlots={horizonSlots}
                                pinned={pinned}
                                slotStarts={outlook.slotStarts}
                                selected={selectedCell === area.cell}
                                dimmed={trail !== null && marks.size === 0}
                                marks={marks}
                                onOpen={() =>
                                  onOpenArea(area.cell, outlook.slotStarts[peak.index] ?? first)
                                }
                              />
                            );
                          })}
                        </div>
                      </div>
                    </div>
                  );
                })}
                {wire ? (
                  <svg
                    aria-hidden
                    className="pointer-events-none absolute inset-0 z-3 size-full overflow-visible"
                  >
                    <path
                      d={wire.path}
                      className="fill-none stroke-foreground"
                      strokeWidth={1.75}
                      strokeLinecap="round"
                    />
                    <path
                      d={wire.head}
                      className="fill-none stroke-foreground"
                      strokeWidth={1.75}
                      strokeLinecap="round"
                    />
                    {wire.label ? (
                      <g>
                        <rect
                          x={wire.x - (wire.label.length * 6.6 + 14) / 2}
                          y={wire.y - 10}
                          width={wire.label.length * 6.6 + 14}
                          height={20}
                          rx={5}
                          className="fill-ink"
                        />
                        <text
                          x={wire.x}
                          y={wire.y + 4}
                          textAnchor="middle"
                          className="fill-ink-foreground text-xs font-medium"
                        >
                          {wire.label}
                        </text>
                      </g>
                    ) : null}
                  </svg>
                ) : null}
              </div>
            </PointerHoverCard>
          </ScrollArea>
          <footer className="shrink-0 border-t border-border-subtle px-3.5 py-1.75 text-xs text-foreground-subtle">
            {t(
              "A cell is loads likely minus drivers expected in that area and window. Click a window to read everything at that hour; click an area to open it.",
            )}
          </footer>
        </div>
      </div>
    </section>
  );
}
