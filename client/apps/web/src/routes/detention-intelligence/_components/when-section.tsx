import {
  HoverCardHeading,
  HoverCardRow,
  PointerHoverCard,
  usePointerHoverTarget,
} from "@/components/pointer-hover-card";
import { SectionPanel } from "@/components/section-panel";
import type {
  DetentionIntelligence,
  DetentionSplitFigure,
} from "@/lib/graphql/detention-intelligence";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import {
  ARRIVAL_BANDS,
  ARRIVAL_BAND_LABELS,
  MIN_ARRIVALS_PER_CELL,
  WEEKDAYS,
  WEEKDAY_LABELS,
  afternoonShare,
  arrivalPeak,
  foldArrivals,
  shareOf,
  type ArrivalBand,
} from "./intelligence-model";
import { Finding, HATCH_BACKGROUND, IntelligenceSection, percent } from "./intelligence-parts";
import { useFacilityIntelligence } from "./use-detention-intelligence";

const strong = (chunk: React.ReactNode) => (
  <span className="text-foreground font-medium">{chunk}</span>
);

/** The facilities the grid can narrow to: the ones with the most hours. */
const HEAT_FACILITIES = 4;
/**
 * The ramp, as the design draws it: the heat hue mixed into the card from 8% for the fewest
 * detained to 90% for the most, with white figures once a cell is past the ramp's middle.
 */
const HEAT_FLOOR = 8;
const HEAT_SPAN = 82;
const HEAT_INK_FROM = 0.55;
const HEAT_LEGEND_STEPS = [0, 0.33, 0.66, 1] as const;
function heatShade(position: number): string {
  return `color-mix(in oklch, var(--accent-orange) ${Math.round(HEAT_FLOOR + position * HEAT_SPAN)}%, var(--color-card))`;
}

function heatPosition(share: number, min: number, max: number): number {
  return max > min ? Math.min(Math.max((share - min) / (max - min), 0), 1) : 0;
}

/**
 * The shares the ramp spans: the cells with enough arrivals to say, or every cell with an arrival
 * when none has enough yet, so a small window still shades its grid.
 */
function heatRange(grid: readonly ArrivalBand[][]): { min: number; max: number } {
  const cells = grid.flat().filter((cell) => cell.stopCount > 0);
  const solid = cells.filter((cell) => cell.stopCount >= MIN_ARRIVALS_PER_CELL);
  const shares = (solid.length > 0 ? solid : cells).map((cell) =>
    shareOf(cell.detainedCount, cell.stopCount),
  );
  return {
    min: shares.length ? Math.min(...shares) : 0,
    max: shares.length ? Math.max(...shares) : 0,
  };
}

/** The ramp's key, as the design draws it: fewer to more detained. */
function HeatLegend() {
  const t = useT();
  return (
    <span className="text-muted-foreground text-2xs hidden items-center gap-1 sm:flex">
      {t("Fewer")}
      {HEAT_LEGEND_STEPS.map((step) => (
        <span
          key={step}
          className="h-2.5 w-3.5 rounded-xs"
          style={{ background: heatShade(step) }}
        />
      ))}
      {t("More")}
    </span>
  );
}

/** One weekday and arrival band, as the hover card reads it. */
type HeatSlot = { id: string; day: number; band: number; cell: ArrivalBand };

function slotName(slot: HeatSlot): string {
  return `${WEEKDAY_LABELS[WEEKDAYS[slot.day]]} ${ARRIVAL_BAND_LABELS[ARRIVAL_BANDS[slot.band].key]}`;
}

function HeatSlotDetail({ slot }: { slot: HeatSlot }) {
  const t = useT();
  const { cell } = slot;
  return (
    <>
      <HoverCardHeading>{slotName(slot)}</HoverCardHeading>
      {cell.stopCount === 0 ? (
        <HoverCardRow label={t("No arrivals")} />
      ) : (
        <HoverCardRow
          label={t("{0, plural, one {# arrival} other {# arrivals}}", cell.stopCount)}
          value={t("{0} detained", percent(shareOf(cell.detainedCount, cell.stopCount)))}
        />
      )}
    </>
  );
}

function HeatCell({
  slot,
  min,
  max,
  compact,
}: {
  slot: HeatSlot;
  min: number;
  max: number;
  compact: boolean;
}) {
  const t = useT();
  const hover = usePointerHoverTarget(slot, slot.id);
  const { cell } = slot;
  const size = compact ? "h-5" : "text-2xs h-8";
  // Only an hour nobody arrived in is hatched. Every other cell is shaded, however few arrived;
  // its hover card says when there are too few to call a pattern.
  if (cell.stopCount === 0) {
    return (
      <span
        role="cell"
        aria-label={t("{0}: no arrivals", slotName(slot))}
        className={cn(
          "text-foreground-subtle grid place-items-center rounded-sm font-mono font-medium",
          size,
        )}
        style={{ background: HATCH_BACKGROUND }}
        {...hover}
      >
        {compact ? null : "–"}
      </span>
    );
  }
  const share = shareOf(cell.detainedCount, cell.stopCount);
  const position = heatPosition(share, min, max);
  return (
    <span
      role="cell"
      aria-label={t(
        "{0}: {1, plural, one {# arrival} other {# arrivals}}, {2} detained",
        slotName(slot),
        cell.stopCount,
        percent(share),
      )}
      className={cn(
        "grid place-items-center rounded-sm font-mono font-medium tabular-nums",
        size,
        // Light figures on a deep cell in both themes; the on-solid ink is dark in the dark theme.
        position > HEAT_INK_FROM
          ? "text-foreground-on-solid dark:text-foreground"
          : "text-foreground",
      )}
      style={{ background: heatShade(position) }}
      {...hover}
    >
      {compact ? null : percent(share)}
    </span>
  );
}

export function ArrivalGrid({
  grid,
  compact = false,
}: {
  grid: ArrivalBand[][];
  compact?: boolean;
}) {
  const t = useT();
  const { min, max } = heatRange(grid);

  return (
    <PointerHoverCard<HeatSlot>
      itemId={(slot) => slot.id}
      render={(slot) => <HeatSlotDetail slot={slot} />}
    >
      <div
        role="table"
        aria-label={t("Share of stops detained by weekday and arrival hour")}
        className="grid grid-cols-[2rem_repeat(8,minmax(0,1fr))] gap-0.5"
      >
        <span role="row" className="contents">
          <span role="columnheader" />
          {ARRIVAL_BANDS.map((band) => (
            <span
              role="columnheader"
              key={band.key}
              className="text-muted-foreground text-2xs truncate pb-0.5 text-center font-mono"
            >
              {ARRIVAL_BAND_LABELS[band.key]}
            </span>
          ))}
        </span>
        {grid.map((row, day) => (
          <span role="row" key={WEEKDAYS[day]} className="contents">
            <span role="rowheader" className="text-muted-foreground flex items-center text-xs">
              {WEEKDAY_LABELS[WEEKDAYS[day]]}
            </span>
            {row.map((cell, band) => (
              <HeatCell
                key={band}
                slot={{ id: `${day}-${band}`, day, band, cell }}
                min={min}
                max={max}
                compact={compact}
              />
            ))}
          </span>
        ))}
      </div>
      {compact ? null : (
        <p className="text-muted-foreground mt-2 text-xs">{t("Hatched cells had no arrivals.")}</p>
      )}
    </PointerHoverCard>
  );
}

function SplitTile({ label, figure }: { label: string; figure: DetentionSplitFigure }) {
  const t = useT();
  return (
    <div className="bg-muted/40 rounded-md border px-3 py-2">
      <p className="text-muted-foreground text-xs">{label}</p>
      <p className="mt-0.5 font-mono text-base font-semibold tabular-nums">
        {figure.stopCount > 0 ? percent(shareOf(figure.detainedCount, figure.stopCount)) : "—"}
      </p>
      <p className="text-muted-foreground text-2xs">
        {t("detained · {0, plural, one {# stop} other {# stops}}", figure.stopCount)}
      </p>
    </div>
  );
}

function SplitsCard({ splits }: { splits: DetentionIntelligence["splits"] }) {
  const t = useT();
  const rt = useRichT();
  const pickup = shareOf(splits.pickup.detainedCount, splits.pickup.stopCount);
  const delivery = shareOf(splits.delivery.detainedCount, splits.delivery.stopCount);
  const open = shareOf(splits.open.detainedCount, splits.open.stopCount);
  const drop = shareOf(splits.drop.detainedCount, splits.drop.stopCount);
  const live = shareOf(splits.live.detainedCount, splits.live.stopCount);

  return (
    <SectionPanel
      title={t("Which stops wait")}
      description={t(
        "Pickups against deliveries, appointments against open docks, live stops against drops.",
      )}
    >
      <div className="flex flex-col gap-2 p-3">
        <Finding>
          {splits.drop.stopCount > 0
            ? rt(
                "Live stops are detained <b>{0}</b> of the time against <b>{1}</b> for drops. Pickups wait {2}, deliveries {3}.",
                { b: strong },
                percent(live),
                percent(drop),
                percent(pickup),
                percent(delivery),
              )
            : rt(
                "Pickups are detained <b>{0}</b> of the time against <b>{1}</b> of deliveries, and open docks <b>{2}</b>. Mark drop stops on the shipment to see what dropping trailers saves.",
                { b: strong },
                percent(pickup),
                percent(delivery),
                percent(open),
              )}
        </Finding>
        <div className="grid grid-cols-2 gap-2">
          <SplitTile label={t("Pickups")} figure={splits.pickup} />
          <SplitTile label={t("Deliveries")} figure={splits.delivery} />
          <SplitTile label={t("Appointment")} figure={splits.appointment} />
          <SplitTile label={t("Open dock")} figure={splits.open} />
          <SplitTile label={t("Live stops")} figure={splits.live} />
          <SplitTile label={t("Drop stops")} figure={splits.drop} />
        </div>
      </div>
    </SectionPanel>
  );
}

/** The grid for every facility, or for one read on its own when the reader narrows to it. */
function useArrivalGrid(data: DetentionIntelligence, locationId: string | null) {
  const scoped = useFacilityIntelligence(data, locationId);
  const source = locationId ? scoped.data : data;
  return {
    grid: source ? foldArrivals(source.arrivals) : null,
    loading: locationId !== null && scoped.isLoading,
  };
}

export function WhenSection({ data }: { data: DetentionIntelligence }) {
  const t = useT();
  const rt = useRichT();
  const [scope, setScope] = useState<string>("all");
  const options = [
    { value: "all", label: t("Every facility") },
    ...data.facilities.slice(0, HEAT_FACILITIES).map((facility) => ({
      value: facility.locationId,
      // A facility's name is as long as its owner typed it; capped, so switching facilities
      // never moves the header.
      label: (
        <span className="block max-w-32 truncate" title={facility.locationName}>
          {facility.locationName}
        </span>
      ),
    })),
  ];
  // A facility chosen in a longer window may not rank in a shorter one; fall back to all of them.
  const active = options.some((option) => option.value === scope) ? scope : "all";
  const { grid, loading } = useArrivalGrid(data, active === "all" ? null : active);
  const peak = grid ? arrivalPeak(grid) : null;

  return (
    <IntelligenceSection
      id="when"
      title={t("When they wait")}
      answer={
        peak && grid
          ? rt(
              "Trucks arriving <b>{0} {1}</b> are detained <b>{2}</b> of the time; after noon, {3}. Book appointments away from the dark cells.",
              { b: strong },
              WEEKDAY_LABELS[WEEKDAYS[peak.weekday]],
              ARRIVAL_BAND_LABELS[ARRIVAL_BANDS[peak.band].key],
              percent(peak.share),
              percent(afternoonShare(grid)),
            )
          : t(
              "No hour has enough arrivals yet to name the worst; the shaded cells show what there is so far.",
            )
      }
      action={
        options.length > 2 ? (
          <SegmentedControl
            items={options}
            value={active}
            onValueChange={setScope}
            aria-label={t("Facility")}
          />
        ) : null
      }
    >
      <div className="grid gap-3 xl:grid-cols-[minmax(0,1.6fr)_minmax(18rem,1fr)]">
        <SectionPanel
          title={t("Share of stops detained, by arrival")}
          description={t(
            "Weekday against the hour the truck arrived, in the organization's time zone.",
          )}
          action={<HeatLegend />}
        >
          <div className={cn("p-3", loading && "opacity-60")} aria-busy={loading}>
            {grid ? <ArrivalGrid grid={grid} /> : <div className="ui-shimmer h-64 rounded-md" />}
          </div>
        </SectionPanel>
        <SplitsCard splits={data.splits} />
      </div>
    </IntelligenceSection>
  );
}
