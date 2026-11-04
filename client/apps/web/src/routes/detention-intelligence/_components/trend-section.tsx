import {
  HoverCardHeading,
  HoverCardRow,
  PointerHoverCard,
  usePointerHoverTarget,
} from "@/components/pointer-hover-card";
import { SectionPanel } from "@/components/section-panel";
import type {
  DetentionFacility,
  DetentionIntelligence,
} from "@/lib/graphql/detention-intelligence";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, useState } from "react";
import { hours, shareOf, weekDaysInWindow } from "./intelligence-model";
import {
  BESIDE_PANEL,
  GrowingRule,
  IntelligenceSection,
  percent,
  useFacilityHighlight,
} from "./intelligence-parts";

const strong = (chunk: React.ReactNode) => (
  <span className="text-foreground font-medium">{chunk}</span>
);

const CHART_HEIGHT = 220;
const LEFT = 40;
const RIGHT = 12;
const TOP = 28;
const BOTTOM = 24;
const SECONDS_PER_WEEK = 7 * 86_400;
const MOVERS = 4;

/** A week's label: the date its Monday falls on, which the server writes as if it were UTC. */
function weekLabel(weekStart: number): string {
  return new Date(weekStart * 1000).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  });
}

/** One week as its tooltip reads it: the hours, against the week before's average. */
type WeekPointItem = {
  weekStart: number;
  minutes: number;
  baseline: number | null;
  /** How many of its seven days fall inside the window; fewer reads low through no change. */
  days: number;
};
/** A week with fewer days in the window is drawn as a fainter dotted continuation. */
const FULL_WEEK = 7;
type PolicyChange = DetentionIntelligence["policyChanges"][number];
/** Policy changes close enough on the chart to share one marker. */
type ChangeCluster = { id: string; x: number; changes: PolicyChange[] };
type TrendTip = { kind: "week"; week: WeekPointItem } | { kind: "changes"; cluster: ChangeCluster };

/** Markers closer than this, in pixels, merge into one with a count. */
const MARKER_GAP = 18;
const MARKER_ROWS = 6;

function clusterChanges(
  changes: readonly PolicyChange[],
  place: (change: PolicyChange) => number | null,
): ChangeCluster[] {
  const placed = changes
    .map((change) => ({ change, x: place(change) }))
    .filter((entry): entry is { change: PolicyChange; x: number } => entry.x !== null)
    .sort((a, b) => a.x - b.x);

  const clusters: ChangeCluster[] = [];
  for (const { change, x } of placed) {
    const last = clusters.at(-1);
    if (last && x - last.x < MARKER_GAP) {
      last.changes.push(change);
      continue;
    }
    clusters.push({ id: `${change.locationId}-${change.changedAt}`, x, changes: [change] });
  }

  return clusters;
}

function changeDate(at: number): string {
  return new Date(at * 1000).toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

function ChangesDetail({ cluster }: { cluster: ChangeCluster }) {
  const t = useT();
  const shown = cluster.changes.slice(0, MARKER_ROWS);
  const hidden = cluster.changes.length - shown.length;

  return (
    <>
      <HoverCardHeading>
        {t("{0, plural, one {# policy change} other {# policy changes}}", cluster.changes.length)}
      </HoverCardHeading>
      {shown.map((change) => (
        <div key={`${change.locationId}-${change.changedAt}`} className="mt-1 first-of-type:mt-0">
          <HoverCardRow label={change.locationName} value={changeDate(change.changedAt)} />
          <p className="text-muted-foreground text-pretty">
            {t("Moved to {0}", change.policyName)}
          </p>
        </div>
      ))}
      {hidden > 0 ? (
        <HoverCardRow label={t("{0, plural, one {# more} other {# more}}", hidden)} />
      ) : null}
    </>
  );
}

function ChangeMarker({
  cluster,
  top,
  bottom,
}: {
  cluster: ChangeCluster;
  top: number;
  bottom: number;
}) {
  const t = useT();
  const hover = usePointerHoverTarget<TrendTip>({ kind: "changes", cluster }, cluster.id);
  const count = cluster.changes.length;

  return (
    <g
      tabIndex={0}
      aria-label={t("{0, plural, one {# policy change} other {# policy changes}}", count)}
      className="ui-focus-ring cursor-default outline-none"
      {...hover}
    >
      <line
        x1={cluster.x}
        x2={cluster.x}
        y1={top}
        y2={bottom}
        strokeDasharray="2 3"
        className="stroke-warning pointer-events-none"
      />
      <rect x={cluster.x - 9} y={top - 9} width={18} height={18} className="fill-transparent" />
      {count > 1 ? (
        <>
          <circle cx={cluster.x} cy={top} r={7} className="fill-warning" />
          <text
            x={cluster.x}
            y={top + 3}
            textAnchor="middle"
            className="fill-warning-on-solid text-2xs font-mono font-semibold tabular-nums"
          >
            {count}
          </text>
        </>
      ) : (
        <circle cx={cluster.x} cy={top} r={3.5} className="fill-warning" />
      )}
    </g>
  );
}

function WeekDetail({ week }: { week: WeekPointItem }) {
  const t = useT();
  return (
    <>
      <HoverCardHeading>{t("Week of {0}", weekLabel(week.weekStart))}</HoverCardHeading>
      <HoverCardRow
        label={
          week.baseline === null
            ? t("Past free time")
            : t("{0} h a week the window before", Math.round(week.baseline))
        }
        value={t("{0} h", hours(week.minutes).toLocaleString())}
      />
      {week.days < FULL_WEEK ? (
        <HoverCardRow label={t("{0} of 7 days in this window", week.days)} />
      ) : null}
    </>
  );
}

function WeekPoint({ week, cx, cy }: { week: WeekPointItem; cx: number; cy: number }) {
  const hover = usePointerHoverTarget<TrendTip>({ kind: "week", week }, String(week.weekStart));
  return <circle cx={cx} cy={cy} r={10} className="fill-transparent" {...hover} />;
}

export function WeeklyHoursChart({
  weeks,
  baseline,
  changes,
  window,
  height = CHART_HEIGHT,
}: {
  weeks: DetentionIntelligence["weeks"];
  /** The window the weeks were read over, so its partial first and last weeks are drawn as such. */
  window: { from: number; to: number; timezone: string };
  /** The prior window's weekly average, drawn as a dashed reference. */
  baseline: number | null;
  changes: DetentionIntelligence["policyChanges"];
  height?: number;
}) {
  const t = useT();
  const [ref, width] = useElementWidth<HTMLDivElement>();
  const values = weeks.map((week) => week.pastFreeMinutes / 60);
  const max = Math.max(...values, baseline ?? 0, 1) * 1.1;
  const span = Math.max(weeks.length - 1, 1);
  const x = (index: number) => LEFT + (index / span) * (width - LEFT - RIGHT);
  const y = (value: number) => TOP + (1 - value / max) * (height - TOP - BOTTOM);
  const points = values.map((value, index) => `${x(index)},${y(value)}`).join(" ");
  const days = weeks.map((week) => weekDaysInWindow(week.weekStart, window));
  const firstFull = days.findIndex((count) => count >= FULL_WEEK);
  const fullFromEnd = [...days].reverse().findIndex((count) => count >= FULL_WEEK);
  const lastFull = fullFromEnd === -1 ? -1 : days.length - 1 - fullFromEnd;
  const line = (from: number, to: number) =>
    values
      .slice(from, to + 1)
      .map((value, offset) => `${x(from + offset)},${y(value)}`)
      .join(" ");
  // Full weeks are the solid line; a partial week at either end continues it, fainter and dotted.
  const partial =
    firstFull === -1
      ? [line(0, values.length - 1)]
      : [
          firstFull > 0 ? line(0, firstFull) : null,
          lastFull < values.length - 1 ? line(lastFull, values.length - 1) : null,
        ].filter((segment): segment is string => segment !== null);
  const firstWeek = weeks[0]?.weekStart ?? 0;
  const ticks = [0, max / 2, max];
  // Each policy change is a marker, never a sentence over the chart: changes close together
  // share one marker with their count, and the names are read on hover.
  const clusters =
    width > 0
      ? clusterChanges(changes, (change) => {
          const index = (change.changedAt - firstWeek) / SECONDS_PER_WEEK;
          return index < 0 || index > span ? null : x(index);
        })
      : [];

  return (
    <PointerHoverCard<TrendTip>
      follow="pointer"
      itemId={(tip) => (tip.kind === "week" ? String(tip.week.weekStart) : tip.cluster.id)}
      render={(tip) =>
        tip.kind === "week" ? (
          <WeekDetail week={tip.week} />
        ) : (
          <ChangesDetail cluster={tip.cluster} />
        )
      }
    >
      <div ref={ref} style={{ minHeight: height }}>
        {width > 0 && weeks.length > 0 ? (
          <svg
            width={width}
            height={height}
            viewBox={`0 0 ${width} ${height}`}
            role="img"
            aria-label={t("Hours past free time each week")}
            className="block overflow-visible"
          >
            {ticks.map((tick) => (
              <g key={tick}>
                <line
                  x1={LEFT}
                  x2={width - RIGHT}
                  y1={y(tick)}
                  y2={y(tick)}
                  className="stroke-border-subtle"
                />
                <text
                  x={LEFT - 8}
                  y={y(tick) + 3}
                  textAnchor="end"
                  className="fill-muted-foreground text-2xs font-mono tabular-nums"
                >
                  {t("{0}h", Math.round(tick))}
                </text>
              </g>
            ))}
            {baseline !== null ? (
              <line
                x1={LEFT}
                x2={width - RIGHT}
                y1={y(baseline)}
                y2={y(baseline)}
                strokeDasharray="4 4"
                className="stroke-muted-foreground pointer-events-none"
              />
            ) : null}
            <polygon
              points={`${x(0)},${y(0)} ${points} ${x(values.length - 1)},${y(0)}`}
              className="fill-foreground/5 pointer-events-none"
            />
            {firstFull !== -1 ? (
              <polyline
                points={line(firstFull, lastFull)}
                fill="none"
                strokeWidth={2}
                className="stroke-foreground pointer-events-none"
              />
            ) : null}
            {partial.map((segment) => (
              <polyline
                key={segment}
                points={segment}
                fill="none"
                strokeWidth={2}
                strokeDasharray="1 4"
                strokeLinecap="round"
                className="stroke-muted-foreground pointer-events-none"
              />
            ))}
            {weeks.map((week, index) => (
              <WeekPoint
                key={week.weekStart}
                week={{
                  weekStart: week.weekStart,
                  minutes: week.pastFreeMinutes,
                  baseline,
                  days: days[index] ?? FULL_WEEK,
                }}
                cx={x(index)}
                cy={y(values[index] ?? 0)}
              />
            ))}
            {clusters.map((cluster) => (
              <ChangeMarker
                key={cluster.id}
                cluster={cluster}
                top={TOP - 10}
                bottom={height - BOTTOM}
              />
            ))}
            {weeks.length > 1
              ? [0, Math.floor((weeks.length - 1) / 2), weeks.length - 1].map((index) => (
                  <text
                    key={index}
                    x={x(index)}
                    y={height - 6}
                    textAnchor={
                      index === 0 ? "start" : index === weeks.length - 1 ? "end" : "middle"
                    }
                    className="fill-muted-foreground text-2xs font-mono"
                  >
                    {weekLabel(weeks[index]?.weekStart ?? 0)}
                  </text>
                ))
              : null}
          </svg>
        ) : null}
      </div>
    </PointerHoverCard>
  );
}

function MoverRow({ facility, onOpen }: { facility: DetentionFacility; onOpen: () => void }) {
  const t = useT();
  const { attributes, handlers } = useFacilityHighlight(facility.locationId);
  const before = facility.previousPastFreeMinutes ?? 0;
  const change = facility.pastFreeMinutes - before;

  return (
    <button
      type="button"
      onClick={onOpen}
      className={cn(
        "ui-focus-ring hover:bg-muted/50 grid w-full grid-cols-[minmax(0,1fr)_4rem_3rem] items-center gap-3 border-t py-1.5 text-left text-sm transition-opacity",
      )}
      {...attributes}
      {...handlers}
    >
      <span className="truncate">{facility.locationName}</span>
      <span
        className={cn(
          "text-right font-mono tabular-nums",
          change > 0 ? "text-danger-foreground" : "text-success-foreground",
        )}
      >
        {change > 0 ? t("+{0} h", hours(change)) : t("−{0} h", hours(-change))}
      </span>
      <span className="text-muted-foreground text-right text-xs tabular-nums">
        {before > 0 ? percent(Math.abs(shareOf(change, before))) : "—"}
      </span>
    </button>
  );
}

/**
 * One list of movers under a heading that sticks while its rows scroll beneath it, until the next
 * group's heading takes its place. A marker at the top of the group says when the heading is
 * pinned (the marker has scrolled above the top), and only then is the rule under it drawn.
 */
function MoverGroup({
  title,
  root,
  children,
}: {
  title: string;
  /** The scrolling viewport the heading sticks in. */
  root: HTMLDivElement | null;
  children: React.ReactNode;
}) {
  const marker = useRef<HTMLDivElement>(null);
  const [stuck, setStuck] = useState(false);

  useEffect(() => {
    const node = marker.current;
    if (!node || !root || typeof IntersectionObserver === "undefined") return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (!entry) return;
        const top = entry.rootBounds?.top ?? 0;
        setStuck(!entry.isIntersecting && entry.boundingClientRect.top < top);
      },
      { root, threshold: 0 },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [root]);

  return (
    <section className="mt-1 first:mt-0">
      <div ref={marker} aria-hidden className="h-0" />
      <h4 className="bg-card text-muted-foreground sticky top-0 z-10 pt-3 pb-1.5 text-xs font-medium">
        {title}
        <GrowingRule shown={stuck} className="bottom-0" />
      </h4>
      {children}
    </section>
  );
}

export function TrendSection({
  data,
  onOpenFacility,
}: {
  data: DetentionIntelligence;
  onOpenFacility: (locationId: string) => void;
}) {
  const t = useT();
  const rt = useRichT();
  const [moversViewport, setMoversViewport] = useState<HTMLDivElement | null>(null);
  const { totals, previous } = data;
  const compared = data.facilities.filter(
    (f) => f.previousPastFreeMinutes !== null && f.previousPastFreeMinutes !== undefined,
  );
  const byChange = [...compared].sort(
    (a, b) =>
      b.pastFreeMinutes -
      (b.previousPastFreeMinutes ?? 0) -
      (a.pastFreeMinutes - (a.previousPastFreeMinutes ?? 0)),
  );
  const worse = byChange
    .filter((f) => f.pastFreeMinutes > (f.previousPastFreeMinutes ?? 0))
    .slice(0, MOVERS);
  const better = byChange
    .filter((f) => f.pastFreeMinutes < (f.previousPastFreeMinutes ?? 0))
    .reverse()
    .slice(0, MOVERS);
  const change = shareOf(
    totals.pastFreeMinutes - previous.pastFreeMinutes,
    previous.pastFreeMinutes,
  );
  const previousWeeks = Math.max(
    Math.round((data.previousTo - data.previousFrom) / SECONDS_PER_WEEK),
    1,
  );
  const baseline =
    previous.pastFreeMinutes > 0 ? previous.pastFreeMinutes / 60 / previousWeeks : null;

  return (
    <IntelligenceSection
      id="trend"
      title={t("Is it getting better")}
      answer={
        previous.pastFreeMinutes === 0
          ? t("Nothing to compare yet: no truck waited past free time in the window before.")
          : change > 0
            ? rt(
                "Hours past free time are <b>{0} up</b> on the window before.",
                {
                  b: (chunk) => <span className="text-danger-foreground font-medium">{chunk}</span>,
                },
                percent(change),
              )
            : rt(
                "Hours past free time are <b>{0} down</b> on the window before.",
                {
                  b: (chunk) => (
                    <span className="text-success-foreground font-medium">{chunk}</span>
                  ),
                },
                percent(-change),
              )
      }
    >
      <div className="grid gap-3 xl:grid-cols-[minmax(0,1.6fr)_minmax(18rem,1fr)]">
        <SectionPanel
          title={t("Hours past free time, weekly")}
          description={
            baseline === null
              ? t("Each week of this window. Policy changes are marked where they took effect.")
              : t(
                  "The solid line is this window; the dashed line, the window before as a weekly average. Policy changes are marked where they took effect.",
                )
          }
          action={
            <span className="text-muted-foreground hidden items-center gap-3 text-xs md:flex">
              <span className="inline-flex items-center gap-1.5">
                <span aria-hidden className="bg-foreground h-0.5 w-3 rounded-full" />
                {t("This window")}
              </span>
              {baseline !== null ? (
                <span className="inline-flex items-center gap-1.5">
                  <span
                    aria-hidden
                    className="border-muted-foreground w-3 border-t border-dashed"
                  />
                  {t("The window before")}
                </span>
              ) : null}
              {data.policyChanges.length > 0 ? (
                <span className="inline-flex items-center gap-1.5">
                  <span aria-hidden className="bg-warning size-1.5 rounded-full" />
                  {t("Policy change")}
                </span>
              ) : null}
            </span>
          }
        >
          <div className="p-3">
            <WeeklyHoursChart
              weeks={data.weeks}
              baseline={baseline}
              changes={data.policyChanges}
              window={data}
            />
          </div>
        </SectionPanel>
        <div className="relative">
          <SectionPanel
            title={t("Who moved")}
            description={t("Change in hours past free time against the window before.")}
            className={BESIDE_PANEL}
          >
            <ScrollArea
              maskVariant="card"
              maskClassName="top-7"
              className="min-h-0 flex-1"
              viewportRef={setMoversViewport}
            >
              <div className="px-3 pb-3">
                <MoverGroup title={t("Getting worse")} root={moversViewport}>
                  {worse.length ? (
                    worse.map((facility) => (
                      <MoverRow
                        key={facility.locationId}
                        facility={facility}
                        onOpen={() => onOpenFacility(facility.locationId)}
                      />
                    ))
                  ) : (
                    <p className="text-muted-foreground border-t py-2 text-xs">
                      {t("No facility got worse.")}
                    </p>
                  )}
                </MoverGroup>
                <MoverGroup title={t("Getting better")} root={moversViewport}>
                  {better.length ? (
                    better.map((facility) => (
                      <MoverRow
                        key={facility.locationId}
                        facility={facility}
                        onOpen={() => onOpenFacility(facility.locationId)}
                      />
                    ))
                  ) : (
                    <p className="text-muted-foreground border-t py-2 text-xs">
                      {t("No facility got better.")}
                    </p>
                  )}
                </MoverGroup>
              </div>
            </ScrollArea>
          </SectionPanel>
        </div>
      </div>
    </IntelligenceSection>
  );
}
