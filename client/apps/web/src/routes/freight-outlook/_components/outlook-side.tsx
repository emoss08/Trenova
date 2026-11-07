import {
  HoverCardHeading,
  HoverCardRow,
  PointerHoverCard,
  usePointerHoverTarget,
} from "@/components/pointer-hover-card";
import type { ForecastDay, FreightOutlook } from "@/lib/graphql/freight-outlook";
import { formatNumber } from "@trenova/shared/i18n/format";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { daysInRange, paceReadings, type PaceStanding } from "./outlook-model";
import { PanelSection, slotLabel } from "./outlook-parts";

const STANDING_LABELS = defineLabels({
  Behind: "Behind",
  Ahead: "Ahead",
  OnPace: "On pace",
});

const STANDING_TONE: Record<PaceStanding, string> = {
  Behind: "text-warning-foreground",
  Ahead: "text-accent-sky-on-subtle",
  OnPace: "text-foreground-subtle",
};

/**
 * Each customer's freight in the reading against how far ahead it usually books: the bar is
 * what is tendered out of what it usually sends, the tick what is usually in by this hour.
 */
export function OutlookPace({
  outlook,
  indexes,
  onOpenArea,
}: {
  outlook: FreightOutlook;
  indexes: readonly number[];
  onOpenArea: (cell: number, slotStart: number) => void;
}) {
  const t = useT();
  const readings = paceReadings(outlook.pace, indexes);
  const behind = readings.filter((reading) => reading.standing === "Behind").length;

  return (
    <PanelSection
      id="outlook-pace"
      title={t("Customer pace")}
      figure={
        readings.length === 0 ? null : behind > 0 ? (
          <span className="text-warning-foreground">{t("{0} behind", behind)}</span>
        ) : (
          t("On pace")
        )
      }
    >
      {readings.length === 0 ? (
        <p className="text-xs text-foreground-subtle">
          {t("No customer has freight due in this range.")}
        </p>
      ) : (
        <>
          <ScrollArea
            maskVariant="card"
            maskHeight={18}
            className="max-h-49"
            viewportClassName="overscroll-contain"
          >
            <ul className="divide-y divide-border-subtle">
              {readings.map((reading) => {
                const fill = reading.usual > 0 ? Math.min(reading.tendered / reading.usual, 1) : 0;
                const tick = reading.usual > 0 ? Math.min(reading.byNow / reading.usual, 1) : 0;
                return (
                  <li
                    key={`${reading.pace.customerId}:${reading.pace.cell}`}
                    className="flex flex-col gap-1.25 py-2 first:pt-0.5"
                  >
                    <div className="flex items-baseline gap-2">
                      <span className="min-w-0 truncate text-sm font-medium">
                        {reading.pace.customerName}
                      </span>
                      <span
                        className={cn(
                          "ml-auto shrink-0 text-xs",
                          reading.standing === "Behind" && "font-medium",
                          STANDING_TONE[reading.standing],
                        )}
                      >
                        {STANDING_LABELS[reading.standing]}
                      </span>
                    </div>
                    <div
                      role="meter"
                      aria-valuemin={0}
                      aria-valuemax={Math.round(reading.usual)}
                      aria-valuenow={reading.tendered}
                      aria-label={t(
                        "{0} of {1} tendered",
                        reading.tendered,
                        Math.round(reading.usual),
                      )}
                      className="relative h-2 rounded-full bg-sunken"
                    >
                      <span
                        className="absolute inset-y-0 left-0 rounded-full bg-foreground/85"
                        style={{ width: `${fill * 100}%` }}
                      />
                      <span
                        aria-hidden
                        className="absolute -inset-y-0.75 w-0.5 rounded-full bg-warning-foreground"
                        style={{ left: `calc(${tick * 100}% - 1px)` }}
                      />
                    </div>
                    <span className="text-xs text-foreground-subtle">
                      {t(
                        "{0} of {1} in, {2} by now ·",
                        reading.tendered,
                        Math.round(reading.usual),
                        Math.round(reading.byNow),
                      )}{" "}
                      <button
                        type="button"
                        onClick={() => onOpenArea(reading.pace.cell, reading.busiestStart)}
                        className="cursor-pointer rounded-sm ui-focus-ring text-foreground underline decoration-border-strong underline-offset-4 hover:decoration-brand"
                      >
                        {reading.pace.label}
                      </button>{" "}
                      {slotLabel(reading.busiestStart)}
                    </span>
                  </li>
                );
              })}
            </ul>
          </ScrollArea>
          <p className="text-xs leading-snug text-foreground-subtle">
            {t(
              "Against each customer's own booking curve; the tick is where they usually are by now.",
            )}
          </p>
        </>
      )}
    </PanelSection>
  );
}

const RECORD_WIDTH = 372;
const RECORD_HEIGHT = 120;
const RECORD_LEFT = 26;
const RECORD_TOP = 6;
const RECORD_BOTTOM = 18;

function RecordMark({ day, x, y }: { day: ForecastDay; x: number; y: (value: number) => number }) {
  const hover = usePointerHoverTarget(day, String(day.dayStart));
  const outside = day.actual < day.low || day.actual > day.high;
  return (
    <g>
      <rect
        x={x - 5}
        y={y(day.high)}
        width={10}
        height={Math.max(2, y(day.low) - y(day.high))}
        rx={3}
        className="fill-brand-subtle stroke-brand-border"
      />
      <circle
        cx={x}
        cy={y(day.actual)}
        r={3}
        strokeWidth={2}
        className={cn("stroke-card", outside ? "fill-warning" : "fill-foreground")}
      />
      <rect
        x={x - 10}
        y={RECORD_TOP}
        width={20}
        height={RECORD_HEIGHT - RECORD_TOP - RECORD_BOTTOM}
        className="fill-transparent"
        {...hover}
      />
    </g>
  );
}

function RecordDetail({ day }: { day: ForecastDay }) {
  const t = useT();
  return (
    <>
      <HoverCardHeading>
        {formatUnixInUserTimezone(day.dayStart, {
          weekday: "long",
          month: "short",
          day: "numeric",
          timezone: "UTC",
        })}
      </HoverCardHeading>
      <HoverCardRow
        label={t("Forecast range")}
        value={t("{0}–{1}", formatNumber(day.low), formatNumber(day.high))}
      />
      <HoverCardRow label={t("Loads that came")} value={formatNumber(day.actual)} />
      <p className="mt-1 text-2xs text-muted-foreground">
        {day.actual > day.high
          ? t("Busier than the range.")
          : day.actual < day.low
            ? t("Quieter than the range.")
            : t("Inside the range.")}
      </p>
    </>
  );
}

/**
 * How far to trust the forecast: each of the last fourteen days forecast only from the weeks
 * before it, out of sample, against the loads that came.
 */
export function OutlookRecord({ outlook }: { outlook: FreightOutlook }) {
  const t = useT();
  const days = outlook.record;
  const inside = daysInRange(days);
  const above = days.filter((day) => day.actual > day.high).length;
  const typical =
    days.length > 0
      ? days.reduce((sum, day) => sum + Math.abs(day.actual - (day.low + day.high) / 2), 0) /
        days.length
      : 0;
  const top = Math.max(1, ...days.map((day) => Math.max(day.high, day.actual)));
  const y = (value: number) =>
    RECORD_TOP + (1 - value / top) * (RECORD_HEIGHT - RECORD_TOP - RECORD_BOTTOM);
  const step = (RECORD_WIDTH - RECORD_LEFT - 4) / Math.max(days.length, 1);
  const ticks = [0, Math.round(top / 2), top];

  return (
    <PanelSection
      id="outlook-record"
      title={t("Track record")}
      figure={
        days.length > 0
          ? t("{0} of {1, plural, one {# day} other {# days}}", inside, days.length)
          : null
      }
    >
      {days.length === 0 ? (
        <p className="text-xs text-foreground-subtle">
          {t("The track record appears after the next nightly fit.")}
        </p>
      ) : (
        <>
          <PointerHoverCard<ForecastDay>
            itemId={(day) => String(day.dayStart)}
            render={(day) => <RecordDetail day={day} />}
          >
            <svg
              viewBox={`0 0 ${RECORD_WIDTH} ${RECORD_HEIGHT}`}
              className="block h-auto w-full"
              role="img"
              aria-label={t("Daily forecast ranges and the loads that came, last fourteen days")}
            >
              {ticks.map((tick) => (
                <g key={tick}>
                  <line
                    x1={RECORD_LEFT}
                    x2={RECORD_WIDTH - 4}
                    y1={y(tick)}
                    y2={y(tick)}
                    className="stroke-border-subtle"
                  />
                  <text
                    x={RECORD_LEFT - 5}
                    y={y(tick) + 3}
                    textAnchor="end"
                    className="fill-foreground-subtle text-2xs"
                  >
                    {formatNumber(tick)}
                  </text>
                </g>
              ))}
              {days.map((day, index) => {
                const x = RECORD_LEFT + step * (index + 0.5);
                return (
                  <g key={day.dayStart}>
                    <RecordMark day={day} x={x} y={y} />
                    <text
                      x={x}
                      y={RECORD_HEIGHT - 4}
                      textAnchor="middle"
                      className="fill-foreground-subtle text-2xs"
                    >
                      {formatUnixInUserTimezone(day.dayStart, {
                        weekday: "narrow",
                        timezone: "UTC",
                      })}
                    </text>
                  </g>
                );
              })}
            </svg>
          </PointerHoverCard>
          <p className="text-xs leading-snug text-foreground-subtle">
            {t(
              "Each day forecast only from the weeks before it. {0, plural, one {# day was} other {# days were}} busier than the range; a typical day lands ±{1} loads from the middle.",
              above,
              Math.round(typical),
            )}
          </p>
        </>
      )}
    </PanelSection>
  );
}
