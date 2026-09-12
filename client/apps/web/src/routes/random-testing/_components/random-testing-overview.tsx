import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiCard, KpiHeader, KpiSub } from "@/components/kpi/kpi-card";
import type { ProgrammeOverview } from "@/lib/random-testing";
import NumberFlow from "@number-flow/react";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { CalendarCheckIcon, CalendarClockIcon, FlaskConicalIcon, UsersIcon } from "lucide-react";

const VALUE_CLASS = "font-mono text-[26px] leading-none font-semibold tracking-tight tabular-nums";

type RandomTestingOverviewProps = {
  overview: ProgrammeOverview;
};

/**
 * The programme in four numbers, in the order a safety manager asks them:
 * is a draw owed, what has been drawn this year, how do the collections
 * stand against the targets, and how many drivers were in the hat.
 */
export function RandomTestingOverview({ overview }: RandomTestingOverviewProps) {
  const t = useT();

  const selected = overview.drugSelected + overview.alcoholSelected;
  const target = overview.drugTarget + overview.alcoholTarget;

  return (
    <div className="grid grid-cols-4 gap-3 lg:grid-cols-8">
      <KpiCard span={2}>
        <KpiHeader
          icon={<CalendarClockIcon className="size-[11px]" />}
          label={t("Owed now")}
          info={
            <InfoPopover title={t("Owed now")}>
              {
                t("Active pools whose current round has not been drawn. Missed means a past round of this year was never drawn, which a DOT audit will find.")
              }
            </InfoPopover>
          }
          right={overview.missed > 0 ? <Badge variant="inactive">{t("Missed")}</Badge> : null}
        />
        <NumberFlow value={overview.owedNow} className={VALUE_CLASS} aria-label={t("Owed now")} />
        <KpiSub>
          {overview.missed > 0
            ? t("{0, plural, one {# round} other {# rounds}} missed this year", overview.missed)
            : overview.owedNow > 0
              ? t("{0} waiting on this period's draw", overview.owedNow === 1 ? t("A pool is") : t("Pools are"))
              : overview.activePools === 0
                ? t("No active pool to draw from")
                : t("Every current round is drawn")}
        </KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<CalendarCheckIcon className="size-[11px]" />}
          label={t("Rounds this year")}
          info={
            <InfoPopover title={t("Rounds this year")}>
              {
                t("Draws made this year that were not voided. A draft round can still change; a final one is the record.")
              }
            </InfoPopover>
          }
        />
        <NumberFlow
          value={overview.roundsThisYear}
          className={VALUE_CLASS}
          aria-label={t("Rounds this year")}
        />
        <KpiSub>
          {overview.roundsThisYear === 0
            ? t("Nothing drawn yet this year")
            : t("{0} final · {1} draft", overview.finalRounds, overview.draftRounds)}
        </KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<FlaskConicalIcon className="size-[11px]" />}
          label={t("Selected this year")}
          info={
            <InfoPopover title={t("Selected this year")}>
              {
                t("Drivers picked across this year's rounds against what those rounds asked for. Short means a pool was smaller than its target.")
              }
            </InfoPopover>
          }
          right={
            target > 0 && selected < target ? (
              <Badge variant="warning">{t("Short of target")}</Badge>
            ) : null
          }
        />
        <div className="flex items-baseline gap-1">
          <NumberFlow value={selected} className={VALUE_CLASS} aria-label={t("Selected this year")} />
          {target > 0 ? (
            <span className="text-muted-foreground font-mono text-[11px] tabular-nums">
              / {target}
            </span>
          ) : null}
        </div>
        <div className="mt-auto flex flex-col gap-1">
          <RateLine label={t("Drug")} selected={overview.drugSelected} target={overview.drugTarget} />
          <RateLine
            label={t("Alcohol")}
            selected={overview.alcoholSelected}
            target={overview.alcoholTarget}
          />
        </div>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<UsersIcon className="size-[11px]" />}
          label={t("In the hat")}
          info={
            <InfoPopover title={t("In the hat")}>
              {
                t("Drivers eligible at the most recent draw. It changes as people join and leave the pool.")
              }
            </InfoPopover>
          }
        />
        {overview.lastPoolSize === null ? (
          <span className={cn(VALUE_CLASS, "text-muted-foreground")} aria-label={t("In the hat")}>
            —
          </span>
        ) : (
          <NumberFlow
            value={overview.lastPoolSize}
            className={VALUE_CLASS}
            aria-label={t("In the hat")}
          />
        )}
        <KpiSub>
          {overview.lastDrawnAt
            ? t("At the last draw, {0}", formatUnixDate(overview.lastDrawnAt))
            : t("No round has been drawn yet")}
        </KpiSub>
        <KpiSub>
          {t("{0, plural, one {# active pool} other {# active pools}} {1}", overview.activePools, overview.belowMinimum > 0 ? t("· {0} below the DOT minimum", overview.belowMinimum) : "")}
        </KpiSub>
      </KpiCard>
    </div>
  );
}

type RateLineProps = {
  label: string;
  selected: number;
  target: number;
};

function RateLine({ label, selected, target }: RateLineProps) {
  const share = target > 0 ? Math.min(100, Math.round((selected / target) * 100)) : 0;
  return (
    <div
      role="img"
      aria-label={`${label}: ${selected} of ${target}`}
      className="grid grid-cols-[3.25rem_minmax(0,1fr)_auto] items-center gap-2 text-[11px]"
    >
      <span className="text-muted-foreground">{label}</span>
      <span className="bg-muted flex h-1 overflow-hidden rounded-full">
        <span
          aria-hidden
          className={cn(
            "h-full rounded-full transition-[width] duration-500 ease-out motion-reduce:transition-none",
            target > 0 && selected < target ? "bg-brand/50" : "bg-brand",
          )}
          style={{ width: `${share}%` }}
        />
      </span>
      <span className="text-muted-foreground font-mono tabular-nums">
        {selected}/{target}
      </span>
    </div>
  );
}
