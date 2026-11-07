import { KPI_STRIP_CELL_CLASS, KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type { FreightOutlook } from "@/lib/graphql/freight-outlook";
import { TrendUp01Icon } from "@trenova/shared/components/icons";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatList, formatNumber, formatRelativeTime } from "@trenova/shared/i18n/format";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn, formatCentsWhole } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import {
  SLOT_SECONDS,
  daysInRange,
  movesIn,
  paceReadings,
  peakOf,
  totalsOf,
  type Totals,
} from "./outlook-model";
import { slotLabel, slotRange } from "./outlook-parts";

const BRIEF_AREAS = 2;
const BRIEF_SPARE_AREAS = 3;
/** The figures run edge to edge across the workspace, as the console's summary strip does. */
const STRIP_CLASS = "rounded-none border-x-0 border-t-0";

type Reading = {
  outlook: FreightOutlook;
  indexes: readonly number[];
  pinned: number | null;
};

/** "through Fri 12 PM" for the whole range, "on Fri 4 AM – 8 AM" for a pinned window. */
export function readingSpan(
  outlook: FreightOutlook,
  indexes: readonly number[],
  pinned: number | null,
  t: TranslateFn,
): string {
  if (pinned !== null) {
    return t("on {0}", slotRange(outlook.slotStarts[pinned] ?? 0));
  }
  const last = indexes[indexes.length - 1] ?? 0;
  return t("through {0}", slotLabel((outlook.slotStarts[last] ?? 0) + SLOT_SECONDS));
}

function Ref({
  tone,
  onClick,
  children,
}: {
  tone?: "short";
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "cursor-pointer rounded-sm ui-focus-ring underline decoration-border-strong underline-offset-4 hover:decoration-brand",
        tone === "short" ? "text-warning-foreground" : "text-foreground",
      )}
    >
      {children}
    </button>
  );
}

/**
 * The reading in one sentence or three, written from the same figures as the page so it never
 * says something the numbers under it do not. Areas open in the inspector; a customer jumps to
 * its pace in the brief.
 */
export function OutlookBrief({
  outlook,
  indexes,
  pinned,
  onOpenArea,
  onJump,
}: Reading & {
  onOpenArea: (cell: number, slotStart: number) => void;
  onJump: (section: string) => void;
}) {
  const t = useT();
  const rt = useRichT();
  const now = useNow("minute");
  const totals = totalsOf(outlook, indexes);
  const peaks = outlook.areas.map((area) => ({ area, peak: peakOf(area, indexes) }));
  const short = peaks
    .filter(({ peak }) => peak.kind === "short")
    .sort((a, b) => b.peak.amount - a.peak.amount)
    .slice(0, BRIEF_AREAS);
  const spare = peaks
    .filter(({ peak }) => peak.kind === "spare")
    .sort((a, b) => b.peak.amount - a.peak.amount)
    .slice(0, BRIEF_SPARE_AREAS);
  const moves = movesIn(outlook.moves, outlook.slotStarts, indexes).filter(
    (move) => move.kind !== "Call",
  );
  const behind = paceReadings(outlook.pace, indexes).find(
    (reading) => reading.standing === "Behind",
  );
  const span = readingSpan(outlook, indexes, pinned, t);

  const areaRef = (entry: (typeof short)[number]) => (chunks: ReactNode) => (
    <Ref
      tone="short"
      onClick={() => onOpenArea(entry.area.cell, outlook.slotStarts[entry.peak.index] ?? 0)}
    >
      {chunks}
    </Ref>
  );
  const strong = (chunks: ReactNode) => (
    <span className="font-medium text-foreground">{chunks}</span>
  );
  const spareDrivers = (chunks: ReactNode) => (
    <span className="text-accent-sky-on-subtle">{chunks}</span>
  );

  return (
    <section aria-label={t("Summary")} className="flex items-start gap-3">
      <span
        aria-hidden
        className="mt-0.5 grid size-5.5 shrink-0 place-items-center rounded-md border border-brand-border bg-brand-subtle text-brand"
      >
        <TrendUp01Icon className="size-3" />
      </span>
      <p className="min-w-0 flex-1 text-base leading-relaxed text-pretty text-muted-foreground">
        {short.length === 2
          ? rt(
              "Freight outruns your drivers in <a>{0}</a> and <c>{1}</c> {2}, <b>{3, plural, one {# load} other {# loads}}</b> with no truck nearby.",
              { a: areaRef(short[0]), c: areaRef(short[1]), b: strong },
              short[0].area.label,
              short[1].area.label,
              span,
              totals.uncovered,
            )
          : short.length === 1
            ? rt(
                "Freight outruns your drivers in <a>{0}</a> {1}, <b>{2, plural, one {# load} other {# loads}}</b> with no truck nearby.",
                { a: areaRef(short[0]), b: strong },
                short[0].area.label,
                span,
                totals.uncovered,
              )
            : t("Your drivers cover the freight {0}.", span)}{" "}
        {spare.length > 0
          ? moves.length > 0
            ? rt(
                "<d>{0, plural, one {# driver} other {# drivers}}</d> will sit with little to haul, most in {1}, and moving {2, plural, one {one of them} other {# of them}} first is worth <b>{3}</b>.",
                { d: spareDrivers, b: strong },
                totals.idle,
                formatList(spare.map(({ area }) => area.label)),
                moves.length,
                formatCentsWhole(totals.worthCents),
              )
            : rt(
                "<d>{0, plural, one {# driver} other {# drivers}}</d> will sit with little to haul, most in {1}.",
                { d: spareDrivers },
                totals.idle,
                formatList(spare.map(({ area }) => area.label)),
              )
          : null}{" "}
        {behind
          ? rt(
              "<c>{0}</c> is behind its usual pace in {1}.",
              { c: (chunks) => <Ref onClick={() => onJump("outlook-pace")}>{chunks}</Ref> },
              behind.pace.customerName,
              behind.pace.label,
            )
          : null}
      </p>
      <span className="shrink-0 pt-1 text-xs text-foreground-subtle max-lg:hidden">
        {t(
          "Updated {0} · learned from {1, plural, one {# week} other {# weeks}}",
          formatRelativeTime(Math.min(0, outlook.generatedAt - now)),
          outlook.quality.historyWeeks,
        )}
      </span>
    </section>
  );
}

function networkFigure(totals: Totals, outlook: FreightOutlook, t: TranslateFn) {
  if (totals.networkShort >= 3) {
    return {
      value: t("{0} short", Math.round(totals.networkShort)),
      sub: t(
        "at {0}; the rest is placement",
        slotLabel(outlook.slotStarts[totals.networkShortIndex] ?? 0),
      ),
    };
  }
  return { value: t("Enough drivers"), sub: t("the gap is placement, not headcount") };
}

function RangeHeld({ outlook, onJump }: { outlook: FreightOutlook; onJump: () => void }) {
  const t = useT();
  const days = outlook.record;
  const inside = daysInRange(days);
  return (
    <button
      type="button"
      onClick={onJump}
      className={cn(
        KPI_STRIP_CELL_CLASS,
        "cursor-pointer text-left ui-inset-focus-ring transition-colors hover:bg-surface-hover",
      )}
    >
      <div className="text-xs font-medium text-foreground-subtle">{t("Range held")}</div>
      {days.length > 0 ? (
        <>
          <div className="mt-1.5 flex items-baseline gap-1.5">
            <span className="text-xl leading-none font-semibold text-foreground tabular-nums">
              {formatNumber(inside)}
            </span>
            <span className="text-xs text-muted-foreground">
              {t("of {0, plural, one {# day} other {# days}}", days.length)}
            </span>
          </div>
          <div aria-hidden className="mt-2 flex gap-0.75">
            {days.map((day) => (
              <span
                key={day.dayStart}
                className={cn(
                  "h-2.5 flex-1 rounded-xs",
                  day.actual < day.low || day.actual > day.high ? "bg-warning" : "bg-success/75",
                )}
              />
            ))}
          </div>
        </>
      ) : (
        <div className="mt-1.5 text-xs text-muted-foreground">
          {t("After the next nightly fit")}
        </div>
      )}
    </button>
  );
}

/** The reading's figures as a bar across the workspace, as the console's summary strip. */
export function OutlookFigures({
  outlook,
  indexes,
  pinned,
  onJump,
}: Reading & { onJump: (section: string) => void }) {
  const t = useT();
  const totals = totalsOf(outlook, indexes);
  const network = networkFigure(totals, outlook, t);
  const span = readingSpan(outlook, indexes, pinned, t);

  return (
    <KpiStrip aria-label={t("Outlook figures")} className={STRIP_CLASS}>
      <KpiStripItem
        tone="warning"
        label={t("Loads with no truck nearby")}
        value={formatNumber(totals.uncovered)}
        sub={t("in {0, plural, one {# area} other {# areas}} {1}", totals.shortAreas, span)}
      />
      <KpiStripItem
        tone="info"
        label={t("Drivers with nothing nearby")}
        value={formatNumber(totals.idle)}
        sub={t(
          "in {0, plural, one {# area} other {# areas}}, at the worst hour",
          totals.spareAreas,
        )}
      />
      <KpiStripItem label={t("Across the network")} value={network.value} sub={network.sub} />
      <KpiStripItem
        label={t("Worth acting on")}
        value={formatCentsWhole(totals.worthCents)}
        sub={t("{0, plural, one {# move} other {# moves}} in the brief", totals.moves)}
      />
      <RangeHeld outlook={outlook} onJump={() => onJump("outlook-record")} />
    </KpiStrip>
  );
}
