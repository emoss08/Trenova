import {
  HoverCardHeading,
  HoverCardRow,
  PointerHoverCard,
  usePointerHoverTarget,
} from "@/components/pointer-hover-card";
import { withoutRecordIds } from "@/lib/record-ids";
import type { WorkerTarget } from "@/routes/worker/use-worker-state";
import type { WorkerClearanceSource } from "@trenova/graphql/generated/graphql";
import { ArrowNarrowRightIcon } from "@trenova/shared/components/icons";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  formatUnixDateMedium,
  formatUnixInUserTimezone,
  formatUnixMonthDay,
} from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useState, type CSSProperties } from "react";
import {
  DEFAULT_RUNWAY_RANGE,
  RUNWAY_LANES,
  RUNWAY_RANGES,
  clearanceItemAction,
  placeRunwayLabels,
  runwayBand,
  runwayLaneCount,
  runwayLater,
  runwayMarkers,
  runwayRangeDays,
  runwayTicks,
  type ClearanceItemAction,
  type RunwayLabel,
  type RunwayLabelSide,
  type RunwayMarker,
  type RunwayRange,
  type RunwayTone,
  type RunwayVariant,
  type WorkerClearance,
  type WorkerClearanceItem,
} from "./runway-model";

export type ClearanceRunwayProps = {
  clearance: WorkerClearance;
  /** The overview's `asOf`, unix seconds: the Today line. */
  asOf: number;
  /** `lanes`: the Overview's four lanes with legend and range; `single`: one 72px lane. */
  variant: RunwayVariant;
  /** Only items from these parts of the record (Credentials passes `["Credential"]`). */
  sources?: readonly WorkerClearanceSource[];
  onOpen: (target: WorkerTarget) => void;
};

type Geometry = {
  flag: number;
  lane: number;
  track: number;
  axis: number;
  sides: readonly RunwayLabelSide[];
};

const GEOMETRY: Record<RunwayVariant, Geometry> = {
  lanes: { flag: 22, lane: 44, track: 26, axis: 22, sides: ["above", "below"] },
  single: { flag: 18, lane: 32, track: 22, axis: 22, sides: ["above"] },
};

const MARKER_BOX = 17;
const SHORT_NAME_MAX = 112;
const LABEL_PART_GAP = 5;
const LABEL_PADDING = 4;
const ENTER_MS = 900;
const LABEL_FONT_PX = 11;
/** Later items a lane holds before the rest are counted: two in a 44px lane, one in 32px. */
const LATER_PER_LANE: Record<RunwayVariant, number> = { lanes: 2, single: 1 };

const DIAMOND_TONE: Record<RunwayTone, string> = {
  far: "border-foreground",
  soon: "border-warning",
  near: "border-warning",
  lapsed: "border-danger",
};

const DIAMOND_FILL: Record<RunwayTone, string> = {
  far: "bg-foreground",
  soon: "bg-warning",
  near: "bg-warning",
  lapsed: "bg-danger",
};

const DAYS_TEXT: Record<RunwayTone, string> = {
  far: "text-muted-foreground",
  soon: "text-warning",
  near: "font-medium text-warning",
  lapsed: "text-danger",
};

const HATCH_STYLE: CSSProperties = {
  backgroundImage:
    "repeating-linear-gradient(135deg, transparent 0, transparent 5px, var(--color-border-subtle) 5px, var(--color-border-subtle) 6px)",
};

function laneLabel(lane: (typeof RUNWAY_LANES)[number], t: TranslateFn): string {
  switch (lane) {
    case "License":
      return t("License and endorsements");
    case "Medical":
      return t("Medical");
    case "ReviewsAndQueries":
      return t("Reviews and queries");
    case "Training":
      return t("Training");
  }
}

/** A day count as the runway says it: "in 17 days", "today", "3 days ago". */
export function runwayDays(daysUntil: number, t: TranslateFn): string {
  if (daysUntil === 0) return t("today");
  if (daysUntil > 0) return t("in {0, plural, one {# day} other {# days}}", daysUntil);
  return t("{0, plural, one {# day} other {# days}} ago", -daysUntil);
}

/** What an item does at its date, in a sentence. */
export function runwayEffect(item: WorkerClearanceItem, t: TranslateFn): string {
  if (item.effect !== "StopsAssignment") return t("Due, does not stop driving.");
  if (item.hazmatOnly) return t("Stops hazmat loads.");
  return t("Stops driving.");
}

/** A marker's action, as its tooltip and the clearance band's button read. */
export function clearanceActionLabel(action: ClearanceItemAction, t: TranslateFn): string {
  switch (action) {
    case "renew":
      return t("Renew");
    case "add-credential":
      return t("Add credential");
    case "log-query":
      return t("Log a query");
    case "open-training":
      return t("Open training");
    case "open-review":
      return t("Open review");
    case "start-review":
      return t("Start review");
  }
}

/**
 * What an item is called on screen. The server names the Clearinghouse query and a performance
 * review generically, so those two are named here; a credential or a course keeps its own name.
 */
export function clearanceItemName(
  item: { source: WorkerClearanceSource; name: string },
  t: TranslateFn,
): string {
  switch (item.source) {
    case "ClearinghouseQuery":
      return t("Clearinghouse annual query");
    case "PerformanceReview":
      return t("Performance review");
    case "Credential":
    case "Training":
      return withoutRecordIds(item.name).trim();
  }
}

let measureContext: CanvasRenderingContext2D | null | undefined;

/**
 * Text width in the label's face (the page's own, which the labels inherit), from a canvas; an
 * estimate where there is none (tests). Read when labels are laid out, after the fonts the
 * page uses have been asked for.
 */
function textWidth(text: string): number {
  if (measureContext === undefined) {
    measureContext =
      typeof document === "undefined" ? null : document.createElement("canvas").getContext("2d");
    if (measureContext) {
      const family = getComputedStyle(document.body).fontFamily || "sans-serif";
      measureContext.font = `${LABEL_FONT_PX}px ${family}`;
    }
  }
  if (!measureContext) return text.length * LABEL_FONT_PX * 0.56;
  return measureContext.measureText(text).width;
}

function MarkerTip({ item, t }: { item: WorkerClearanceItem; t: TranslateFn }) {
  const { action } = clearanceItemAction(item);
  return (
    <>
      <HoverCardHeading>{clearanceItemName(item, t)}</HoverCardHeading>
      <HoverCardRow
        label={formatUnixDateMedium(item.dueAt)}
        value={runwayDays(item.daysUntil, t)}
      />
      <p className="mt-1 text-muted-foreground">{runwayEffect(item, t)}</p>
      {item.regulation ? (
        <HoverCardRow label={t("Regulation")} value={withoutRecordIds(item.regulation)} />
      ) : null}
      <p className="mt-1 inline-flex items-center gap-1 font-medium">
        {clearanceActionLabel(action, t)}
        <ArrowNarrowRightIcon aria-hidden className="size-3" />
      </p>
    </>
  );
}

function Diamond({
  tone,
  solid,
  className,
}: {
  tone: RunwayTone;
  solid: boolean;
  className?: string;
}) {
  return (
    <span
      aria-hidden
      className={cn(
        "block size-[9px] shrink-0 rotate-45 rounded-[1px] border-[1.5px]",
        DIAMOND_TONE[tone],
        solid ? DIAMOND_FILL[tone] : "bg-card",
        className,
      )}
    />
  );
}

function Marker({
  marker,
  label,
  geometry,
  entering,
  onOpen,
}: {
  marker: RunwayMarker;
  label: RunwayLabel | undefined;
  geometry: Geometry;
  entering: boolean;
  onOpen: (target: WorkerTarget) => void;
}) {
  const t = useT();
  const { item, tone, inRange } = marker;
  const hover = usePointerHoverTarget(item, item.key);
  const name = clearanceItemName(item, t);
  const date = formatUnixDateMedium(item.dueAt);
  const days = runwayDays(item.daysUntil, t);
  const short = label?.short ?? false;
  const below = label?.side === "below";
  const top = geometry.flag + marker.lane * geometry.lane + geometry.track;

  return (
    <button
      type="button"
      data-runway-marker={item.key}
      data-tone={tone}
      data-solid={marker.solid ? "" : undefined}
      tabIndex={inRange ? 0 : -1}
      aria-hidden={inRange ? undefined : true}
      aria-label={t("{0}, {1}, {2}. {3}", name, date, days, runwayEffect(item, t))}
      onClick={() => onOpen(clearanceItemAction(item).target)}
      {...(inRange ? hover : {})}
      className={cn(
        "group/marker absolute grid place-items-center rounded-sm ui-focus-ring",
        "transition-[left,opacity,visibility] duration-320 ease-settle motion-reduce:transition-none",
        inRange ? "opacity-100" : "pointer-events-none invisible opacity-0",
        entering && inRange && "animate-rise",
      )}
      style={{
        left: `${(inRange ? marker.x : 1) * 100}%`,
        top,
        width: MARKER_BOX,
        height: MARKER_BOX,
        marginLeft: -MARKER_BOX / 2,
        marginTop: -MARKER_BOX / 2,
        animationDelay: entering ? `${240 + marker.lane * 40}ms` : undefined,
      }}
    >
      <Diamond
        tone={tone}
        solid={marker.solid}
        className="transition-transform group-hover/marker:scale-125 group-focus-visible/marker:scale-125"
      />
      {label && !label.hidden ? (
        <span
          aria-hidden
          className={cn(
            "pointer-events-none absolute flex items-baseline gap-[5px] px-0.5 text-xs whitespace-nowrap tabular-nums",
            label.flip ? "right-[18px]" : "left-[18px]",
            below ? "top-[15px]" : "bottom-[15px]",
          )}
        >
          <span className={cn("text-foreground", short && "max-w-28 truncate")}>{name}</span>
          {short ? null : (
            <span className="text-muted-foreground">{formatUnixMonthDay(item.dueAt)}</span>
          )}
          <span className={DAYS_TEXT[tone]}>{days}</span>
        </span>
      ) : null}
    </button>
  );
}

function LaterItem({
  marker,
  onOpen,
}: {
  marker: RunwayMarker;
  onOpen: (target: WorkerTarget) => void;
}) {
  const t = useT();
  const { item } = marker;
  const hover = usePointerHoverTarget(item, item.key);
  const name = clearanceItemName(item, t);
  const date = formatUnixDateMedium(item.dueAt);
  return (
    <button
      type="button"
      data-runway-later={item.key}
      aria-label={t(
        "{0}, {1}, {2}. {3}",
        name,
        date,
        runwayDays(item.daysUntil, t),
        runwayEffect(item, t),
      )}
      onClick={() => onOpen(clearanceItemAction(item).target)}
      {...hover}
      className="flex h-3.5 min-w-0 animate-rise items-center gap-1.5 rounded-xs ui-focus-ring text-left text-2xs whitespace-nowrap text-muted-foreground transition-colors hover:text-foreground"
    >
      <Diamond tone="far" solid={marker.solid} className="size-1.5 border" />
      <span className="truncate">
        <b className="font-medium text-foreground">{name}</b> {date}
      </span>
    </button>
  );
}

/**
 * The clearance runway: one time axis from today with every dated requirement on it, in four
 * lanes (`lanes`) or one (`single`). A band runs from the Today line to the first item that
 * stops the driver: that band is how long the driver is cleared, and past it the plot is
 * hatched. Solid diamonds stop assignment, hollow ones are only due; a hazmat-only stop is
 * solid and does not end the band. Things past the range wait in the Later notch.
 *
 * It is laid out from its measured width and again whenever the width changes. On first
 * sight the band grows from the Today line and the markers rise lane by lane; a range change
 * moves markers and band to their new places and cross-fades the month labels.
 */
export function ClearanceRunway({
  clearance,
  asOf,
  variant,
  sources,
  onOpen,
}: ClearanceRunwayProps) {
  const t = useT();
  const [range, setRange] = useState<RunwayRange>(DEFAULT_RUNWAY_RANGE);
  const [plotRef, width] = useElementWidth<HTMLDivElement>();
  const [drawn, setDrawn] = useState(false);
  const [entering, setEntering] = useState(false);

  // The first frame it is seen draws the band at nothing; the next grows it to its edge.
  useEffect(() => {
    if (drawn || width <= 0) return;
    const frame = requestAnimationFrame(() => {
      setEntering(true);
      setDrawn(true);
    });
    return () => cancelAnimationFrame(frame);
  }, [drawn, width]);

  useEffect(() => {
    if (!entering) return;
    const timer = setTimeout(() => setEntering(false), ENTER_MS);
    return () => clearTimeout(timer);
  }, [entering]);

  const geometry = GEOMETRY[variant];
  const laneCount = runwayLaneCount(variant);
  const shownRange: RunwayRange = variant === "single" ? DEFAULT_RUNWAY_RANGE : range;
  const rangeDays = runwayRangeDays(asOf, shownRange);
  const markers = runwayMarkers(clearance, { variant, rangeDays, sources });
  const later = runwayLater(markers, laneCount);
  const band = runwayBand(clearance, rangeDays);
  const edge = band ? band.edge : 0;
  const shownEdge = drawn ? edge : 0;
  const labels =
    width > 0
      ? placeRunwayLabels(markers, {
          width,
          sides: geometry.sides,
          measure: (item, short) => {
            const name = textWidth(clearanceItemName(item, t));
            const days = textWidth(runwayDays(item.daysUntil, t));
            if (short)
              return Math.min(name, SHORT_NAME_MAX) + LABEL_PART_GAP + days + LABEL_PADDING;
            const date = textWidth(formatUnixMonthDay(item.dueAt));
            return name + date + days + LABEL_PART_GAP * 2 + LABEL_PADDING;
          },
        })
      : new Map<string, RunwayLabel>();
  const plotHeight = geometry.flag + laneCount * geometry.lane + geometry.axis;
  const tickRanges: readonly RunwayRange[] =
    variant === "single" ? [DEFAULT_RUNWAY_RANGE] : RUNWAY_RANGES;
  const moveClass = entering ? "duration-420" : "duration-320";
  const hasLater = later.some((lane) => lane.length > 0) || clearance.itemsTruncated;
  const showLater = variant === "lanes" || hasLater;

  return (
    <PointerHoverCard<WorkerClearanceItem>
      itemId={(item) => item.key}
      render={(item) => <MarkerTip item={item} t={t} />}
    >
      <div className="flex min-w-0 flex-col gap-2.5" data-runway={variant}>
        {variant === "lanes" ? (
          <div className="flex min-h-7 flex-wrap items-center gap-x-4 gap-y-2">
            <span className="text-xs font-medium text-foreground-subtle">
              {t("What comes due")}
            </span>
            <span className="flex flex-wrap gap-x-3.5 gap-y-1 text-xs text-muted-foreground">
              <span className="inline-flex items-center gap-1.5">
                <Diamond tone="far" solid />
                {t("Stops driving")}
              </span>
              <span className="inline-flex items-center gap-1.5">
                <Diamond tone="far" solid={false} />
                {t("Due, does not stop driving")}
              </span>
            </span>
            <SegmentedControl<`${RunwayRange}`>
              className="ml-auto"
              aria-label={t("How far ahead")}
              value={`${range}`}
              onValueChange={(value) => setRange(Number(value) as RunwayRange)}
              items={RUNWAY_RANGES.map((months) => ({
                value: `${months}` as const,
                label: t("{0, plural, one {# month} other {# months}}", months),
              }))}
            />
          </div>
        ) : null}
        <div className="min-w-0 [scrollbar-width:thin] overflow-x-auto">
          <div
            className={cn(
              "grid min-w-0",
              variant === "lanes"
                ? "min-w-[40rem] grid-cols-[7.5rem_minmax(0,1fr)_9.5rem]"
                : showLater
                  ? "grid-cols-[minmax(0,1fr)_9.5rem]"
                  : "grid-cols-[minmax(0,1fr)]",
            )}
          >
            {variant === "lanes" ? (
              <div
                aria-hidden
                className="grid"
                style={{
                  gridTemplateRows: `${geometry.flag}px repeat(${laneCount}, ${geometry.lane}px) ${geometry.axis}px`,
                }}
              >
                <span />
                {RUNWAY_LANES.map((lane) => (
                  <span
                    key={lane}
                    className="flex items-end pr-2 pb-2.5 text-xs leading-3.5 text-muted-foreground"
                  >
                    {laneLabel(lane, t)}
                  </span>
                ))}
              </div>
            ) : null}
            <div
              ref={plotRef}
              role="group"
              aria-label={t("Dated requirements on a time axis from today")}
              className="relative min-w-0"
              style={{ height: plotHeight }}
            >
              <div
                aria-hidden
                className="absolute inset-x-0"
                style={{ top: geometry.flag, bottom: geometry.axis }}
              >
                {Array.from({ length: laneCount }, (_, lane) => (
                  <span
                    key={lane}
                    className="absolute inset-x-0 h-px bg-border-subtle"
                    style={{ top: lane * geometry.lane + geometry.track }}
                  />
                ))}
              </div>
              {edge < 1 ? (
                <div
                  aria-hidden
                  data-runway-hatch
                  className={cn(
                    "absolute right-0 transition-[left,opacity] ease-settle motion-reduce:transition-none",
                    moveClass,
                    drawn ? "opacity-100" : "opacity-0",
                  )}
                  style={{
                    ...HATCH_STYLE,
                    left: `${edge * 100}%`,
                    top: geometry.flag,
                    bottom: geometry.axis,
                    transitionDelay: entering ? "300ms" : undefined,
                  }}
                />
              ) : null}
              {band ? (
                <>
                  <div
                    aria-hidden
                    data-runway-band
                    data-edge={edge.toFixed(4)}
                    className={cn(
                      "absolute left-0 w-full origin-left bg-success-subtle transition-transform ease-settle motion-reduce:transition-none",
                      moveClass,
                    )}
                    style={{
                      top: geometry.flag,
                      bottom: geometry.axis,
                      transform: `scaleX(${shownEdge})`,
                    }}
                  />
                  {band.stop && !band.beyond ? (
                    <div
                      aria-hidden
                      className={cn(
                        "absolute top-0 transition-[left] ease-settle motion-reduce:transition-none",
                        moveClass,
                      )}
                      style={{ left: `${shownEdge * 100}%`, bottom: geometry.axis }}
                    >
                      <span className="absolute top-1 bottom-0 -left-px w-px bg-success-border" />
                      <span
                        className={cn(
                          "absolute top-[3px] rounded-xs border border-success-border bg-success-subtle px-1 text-2xs font-medium whitespace-nowrap text-success-subtle-foreground tabular-nums",
                          band.flagFlipped ? "right-1" : "left-1",
                        )}
                      >
                        {t(
                          "{0} · {1}",
                          formatUnixMonthDay(band.stop.dueAt),
                          clearanceItemName(band.stop, t),
                        )}
                      </span>
                    </div>
                  ) : null}
                </>
              ) : null}
              <div
                aria-hidden
                className="absolute left-0 w-px bg-foreground"
                style={{ top: 4, bottom: geometry.axis }}
              >
                <span className="absolute top-[-1px] left-[5px] text-2xs font-medium text-foreground">
                  {t("Today")}
                </span>
              </div>
              <div
                aria-hidden
                className="absolute inset-x-0 bottom-0 border-t border-border"
                style={{ height: geometry.axis }}
              >
                {tickRanges.map((months) => (
                  <div
                    key={months}
                    data-runway-axis={months}
                    className={cn(
                      "absolute inset-0 transition-opacity duration-200 motion-reduce:transition-none",
                      months === shownRange ? "opacity-100" : "opacity-0",
                    )}
                  >
                    {runwayTicks(asOf, runwayRangeDays(asOf, months)).map((tick) => (
                      <span
                        key={tick.at}
                        className="absolute -top-px h-[5px] w-px bg-border-strong"
                        style={{ left: `${tick.x * 100}%` }}
                      >
                        <span className="absolute top-[5px] left-1 text-2xs whitespace-nowrap text-foreground-subtle tabular-nums">
                          {formatUnixInUserTimezone(tick.at, {
                            month: "short",
                            year: tick.withYear ? "numeric" : undefined,
                            timezone: "UTC",
                          })}
                        </span>
                      </span>
                    ))}
                  </div>
                ))}
              </div>
              {markers.map((marker) => (
                <Marker
                  key={marker.item.key}
                  marker={marker}
                  label={labels.get(marker.item.key)}
                  geometry={geometry}
                  entering={entering}
                  onOpen={onOpen}
                />
              ))}
            </div>
            {showLater ? (
              <div
                aria-label={t("Later than the range")}
                role="group"
                className="ml-3 grid min-w-0 border-l border-dashed border-border-strong pl-2.5"
                style={{
                  gridTemplateRows: `${geometry.flag}px repeat(${laneCount}, ${geometry.lane}px) ${geometry.axis}px`,
                }}
              >
                <span className="self-start pt-[3px] text-2xs font-medium text-foreground-subtle">
                  {t("Later")}
                </span>
                {later.map((lane, index) => (
                  <div key={index} className="flex min-w-0 flex-col justify-center pt-0.5">
                    {lane.slice(0, LATER_PER_LANE[variant]).map((marker) => (
                      <LaterItem key={marker.item.key} marker={marker} onOpen={onOpen} />
                    ))}
                    {lane.length > LATER_PER_LANE[variant] ? (
                      <span className="h-3.5 text-2xs text-muted-foreground tabular-nums">
                        {t("{0, plural, other {# more}}", lane.length - LATER_PER_LANE[variant])}
                      </span>
                    ) : null}
                  </div>
                ))}
                {clearance.itemsTruncated ? (
                  <span className="self-center text-2xs text-muted-foreground">
                    {t("More past these")}
                  </span>
                ) : null}
              </div>
            ) : null}
          </div>
        </div>
      </div>
    </PointerHoverCard>
  );
}
