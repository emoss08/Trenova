import { SectionPanel } from "@/components/section-panel";
import type { DetentionIntelligence } from "@/lib/graphql/detention-intelligence";
import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import {
  CAUSES,
  CAUSE_LABELS,
  FACILITY_RANKS,
  FACILITY_RANK_LABELS,
  amount,
  causeMinutes,
  concentration,
  hours,
  rankFacilities,
  shareOf,
  type FacilityRank,
} from "./intelligence-model";
import { FacilityHoverCard, useFacilityMark } from "./facility-hover";
import { IntelligenceSection, money, percent } from "./intelligence-parts";

/** The ranking draws the facilities that matter; the strip below accounts for the rest. */
const MAX_COLUMNS = 14;
const CHART_HEIGHT = 240;
const AXIS_LEFT = 40;
const AXIS_BOTTOM = 30;
const AXIS_TOP = 18;
const NARROW_COLUMN = 64;

type Facility = DetentionIntelligence["facilities"][number];

function niceMax(value: number): number {
  const step = value > 400 ? 200 : value > 100 ? 50 : value > 20 ? 10 : 5;
  return Math.max(step, Math.ceil(value / step) * step);
}

function Column({
  facility,
  x,
  width,
  y,
  base,
  labelY,
  tilt,
  top,
  figure,
  onOpen,
}: {
  facility: Facility;
  x: number;
  width: number;
  y: (hours: number) => number;
  base: number;
  labelY: number;
  tilt: boolean;
  top: boolean;
  /** The figure over the column: the hours, or what the facilities are ranked by. */
  figure: string;
  onOpen: () => void;
}) {
  const t = useT();
  const { attributes, handlers } = useFacilityMark(facility);
  const barWidth = Math.min(32, width * 0.6);
  const center = x + width / 2;
  const total = hours(facility.pastFreeMinutes);
  const parts = CAUSES.filter((cause) => causeMinutes(facility, cause.key) > 0);

  const barTotal = parts.reduce((sum, part) => sum + causeMinutes(facility, part.key) / 60, 0);

  return (
    <g
      role="button"
      tabIndex={0}
      aria-label={t("{0}: {1} h past free time", facility.locationName, total)}
      className={cn("ui-focus-ring cursor-pointer outline-none transition-opacity")}
      onClick={onOpen}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onOpen();
        }
      }}
      {...attributes}
      {...handlers}
    >
      <rect
        x={x}
        y={AXIS_TOP}
        width={width}
        height={base - AXIS_TOP}
        className="fill-transparent"
      />
      {parts.map((cause, index) => {
        const stacked = parts
          .slice(0, index)
          .reduce((sum, part) => sum + causeMinutes(facility, part.key) / 60, 0);
        const value = causeMinutes(facility, cause.key) / 60;
        const top = y(stacked + value);
        const bottom = y(stacked);
        return (
          <rect
            key={cause.key}
            x={center - barWidth / 2}
            y={top}
            width={barWidth}
            height={Math.max(bottom - top - (index > 0 ? 1.5 : 0), 0.5)}
            rx={index === parts.length - 1 ? 2.5 : 0}
            className={cause.fillClass}
          />
        );
      })}
      <text
        x={center}
        y={y(barTotal) - 6}
        textAnchor="middle"
        className="fill-foreground font-mono text-xs tabular-nums"
      >
        {figure}
      </text>
      <text
        x={center}
        y={labelY}
        textAnchor={tilt ? "end" : "middle"}
        transform={tilt ? `rotate(-40 ${center} ${labelY - 4})` : undefined}
        className={cn("text-xs", top ? "fill-foreground font-medium" : "fill-muted-foreground")}
      >
        {facility.locationName.length > 16
          ? `${facility.locationName.slice(0, 15)}…`
          : facility.locationName}
      </text>
    </g>
  );
}

function FocusChart({
  facilities,
  topIds,
  figureOf,
  onOpen,
}: {
  facilities: readonly Facility[];
  topIds: Set<string>;
  figureOf: (facility: Facility) => string;
  onOpen: (locationId: string) => void;
}) {
  const t = useT();
  const [ref, width] = useElementWidth<HTMLDivElement>();
  const shown = facilities.slice(0, MAX_COLUMNS);
  const columnWidth = shown.length > 0 ? (width - AXIS_LEFT) / shown.length : 0;
  const tilt = columnWidth < NARROW_COLUMN;
  const height = CHART_HEIGHT + (tilt ? 30 : 0);
  const base = height - AXIS_BOTTOM - (tilt ? 30 : 0);
  const max = niceMax(Math.max(...shown.map((f) => f.pastFreeMinutes / 60), 1));
  const y = (value: number) => AXIS_TOP + (1 - value / max) * (base - AXIS_TOP);
  const ticks = [0, max / 4, max / 2, (max * 3) / 4, max];

  return (
    <div ref={ref} className="min-h-60">
      {width > 0 ? (
        <svg
          width={width}
          height={height}
          viewBox={`0 0 ${width} ${height}`}
          role="img"
          aria-label={t("Hours past free time by facility, split by who caused the wait")}
          className="block overflow-visible"
        >
          {ticks.map((tick) => (
            <g key={tick}>
              <line
                x1={AXIS_LEFT}
                x2={width}
                y1={y(tick)}
                y2={y(tick)}
                className="stroke-border-subtle"
              />
              <text
                x={AXIS_LEFT - 8}
                y={y(tick) + 3}
                textAnchor="end"
                className="fill-muted-foreground font-mono text-2xs tabular-nums"
              >
                {t("{0}h", Math.round(tick))}
              </text>
            </g>
          ))}
          {shown.map((facility, index) => (
            <Column
              key={facility.locationId}
              facility={facility}
              x={AXIS_LEFT + columnWidth * index}
              width={columnWidth}
              y={y}
              base={base}
              labelY={base + 16}
              tilt={tilt}
              top={topIds.has(facility.locationId)}
              figure={figureOf(facility)}
              onOpen={() => onOpen(facility.locationId)}
            />
          ))}
        </svg>
      ) : null}
    </div>
  );
}

function ConcentrationStrip({
  facilities,
  total,
  cut,
  truncated,
  onOpen,
}: {
  facilities: readonly Facility[];
  total: number;
  truncated: boolean;
  cut: ReturnType<typeof concentration>;
  onOpen: (locationId: string) => void;
}) {
  const t = useT();
  // The facilities past the list's limit still count toward the total; they are one grey cell.
  const unlisted = Math.max(total - facilities.reduce((sum, f) => sum + f.pastFreeMinutes, 0), 0);

  const rest = Math.max(total - cut.minutes, 0);
  // Past the list's limit there are facilities nobody counted, so the rest is not given a number.
  const restCount = truncated ? 0 : Math.max(facilities.length - cut.count, 0);

  return (
    <div className="mt-4">
      <div className="flex h-5 gap-0.5 overflow-hidden rounded-xs">
        {facilities.map((facility) => (
          <StripCell
            key={facility.locationId}
            facility={facility}
            top={cut.ids.has(facility.locationId)}
            onOpen={() => onOpen(facility.locationId)}
          />
        ))}
        {unlisted > 0 ? (
          <span className="bg-muted block h-full" style={{ flexGrow: unlisted }} />
        ) : null}
      </div>
      {/* A bracket under each group, as wide as the hours it holds, so the split reads at a
          glance: the few facilities worth fixing first, then everything else. */}
      <div aria-hidden className="mt-1 flex gap-1">
        <span
          className="border-border-strong h-1.5 rounded-b-xs border-x border-b"
          style={{ flexGrow: cut.minutes }}
        />
        {rest > 0 ? (
          <span
            className="border-border h-1.5 rounded-b-xs border-x border-b"
            style={{ flexGrow: rest }}
          />
        ) : null}
      </div>
      <div className="mt-1 flex justify-between gap-4 text-xs">
        <span className="font-medium">
          {t(
            "{0, plural, one {# facility} other {# facilities}} · {1} of the hours",
            cut.count,
            percent(cut.share),
          )}
        </span>
        {rest > 0 ? (
          <span className="text-muted-foreground">
            {restCount > 0
              ? t("{0} more · {1}", restCount.toLocaleString(), percent(1 - cut.share))
              : t("Everything else · {0}", percent(1 - cut.share))}
          </span>
        ) : null}
      </div>
    </div>
  );
}

function StripCell({
  facility,
  top,
  onOpen,
}: {
  facility: Facility;
  top: boolean;
  onOpen: () => void;
}) {
  const { attributes, handlers } = useFacilityMark(facility);

  return (
    <button
      type="button"
      aria-label={facility.locationName}
      onClick={onOpen}
      className={cn(
        "ui-focus-ring block h-full min-w-1 transition-opacity",
        top ? "bg-foreground/60" : "bg-foreground/15",
      )}
      style={{ flexGrow: facility.pastFreeMinutes }}
      {...attributes}
      {...handlers}
    />
  );
}

export function FocusSection({
  data,
  onOpenFacility,
}: {
  data: DetentionIntelligence;
  onOpenFacility: (locationId: string) => void;
}) {
  const t = useT();
  const rt = useRichT();
  const [rank, setRank] = useState<FacilityRank>("hours");
  const total = data.totals.pastFreeMinutes;
  const cut = concentration(data.facilities, total);
  const marginsVisible = data.totals.netMargin !== null;
  const ranks = FACILITY_RANKS.filter((option) => option !== "net" || marginsVisible);
  // A rank the reader can no longer use (net margin after margins were withheld) falls back.
  const active = ranks.includes(rank) ? rank : "hours";
  // A facility whose stops all finished inside free time has nothing to draw: no column, no cell.
  const waited = data.facilities.filter((facility) => facility.pastFreeMinutes > 0);
  const ranked = rankFacilities(waited, active);
  const items = ranks.map((option) => ({ value: option, label: FACILITY_RANK_LABELS[option] }));

  const figureOf = (facility: Facility): string => {
    switch (active) {
      case "dock":
        return String(hours(facility.dockMinutes));
      case "detained":
        return percent(shareOf(facility.detainedCount, facility.stopCount));
      case "p90":
        return formatDetentionMinutes(Math.round(facility.p90DwellMinutes));
      case "billed":
        return money(amount(facility.billedAmount));
      case "net":
        return money(amount(facility.netMargin));
      default:
        // Under half an hour rounds to nothing; the column is short, not empty.
        return hours(facility.pastFreeMinutes) === 0
          ? t("<1")
          : String(hours(facility.pastFreeMinutes));
    }
  };

  return (
    <IntelligenceSection
      id="focus"
      title={t("Where to focus")}
      answer={
        total > 0
          ? rt(
              "<b>{0, plural, one {# facility causes} other {# facilities cause}}</b> {1} of the hours past free time. Fixing those first is worth up to <b>{2} h</b> a window.",
              { b: (chunk) => <span className="text-foreground font-medium">{chunk}</span> },
              cut.count,
              percent(cut.share),
              hours(cut.minutes).toLocaleString(),
            )
          : t("Every stop in this window finished inside its free time.")
      }
      action={
        <Select
          value={active}
          items={items}
          onValueChange={(value) => setRank(value as FacilityRank)}
        >
          <SelectTrigger size="sm" aria-label={t("Rank facilities by")}>
            <span className="text-muted-foreground">{t("Rank by")}</span>
            <SelectValue className="font-medium" />
          </SelectTrigger>
          <SelectContent>
            {items.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      }
    >
      <SectionPanel
        title={t("Hours past free time, by facility")}
        hint={
          active === "hours"
            ? undefined
            : t("Ordered by {0}; the columns show hours", FACILITY_RANK_LABELS[active])
        }
        description={t(
          "Each column is split by who caused the wait. Click a facility for its scorecard.",
        )}
        action={
          <span className="text-muted-foreground hidden flex-wrap gap-3 text-xs md:flex">
            {CAUSES.map((cause) => (
              <span key={cause.key} className="inline-flex items-center gap-1.5">
                <span className={cn("size-2 rounded-sm", cause.bgClass)} />
                {CAUSE_LABELS[cause.key]}
              </span>
            ))}
          </span>
        }
      >
        <FacilityHoverCard>
          <div className="p-3">
            <FocusChart
              facilities={ranked}
              topIds={cut.ids}
              figureOf={figureOf}
              onOpen={onOpenFacility}
            />
            <ConcentrationStrip
              facilities={waited}
              total={total}
              cut={cut}
              truncated={data.truncated}
              onOpen={onOpenFacility}
            />
            {data.truncated ? (
              <p className="text-muted-foreground mt-2 text-xs">
                {t(
                  "Only the facilities with the most hours are listed; the totals cover every stop.",
                )}
              </p>
            ) : null}
          </div>
        </FacilityHoverCard>
      </SectionPanel>
    </IntelligenceSection>
  );
}
