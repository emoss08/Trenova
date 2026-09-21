import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiCard, KpiHeader } from "@/components/kpi/kpi-card";
import { KPI_VALUE_CLASS, KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type { ProgrammeOverview } from "@/lib/random-testing";
import NumberFlow from "@number-flow/react";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

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
    <KpiStrip>
      <KpiStripItem
        label={t("Owed now")}
        tone={overview.missed > 0 ? "danger" : undefined}
        info={
          <>
            <InfoPopover title={t("Owed now")}>
              {t(
                "Active pools whose current round has not been drawn. Missed means a past round of this year was never drawn, which a DOT audit will find.",
              )}
            </InfoPopover>
            {overview.missed > 0 ? (
              <Badge variant="danger" className="ml-auto">
                {t("Missed")}
              </Badge>
            ) : null}
          </>
        }
        value={<NumberFlow value={overview.owedNow} aria-label={t("Owed now")} />}
        sub={
          overview.missed > 0
            ? t("{0, plural, one {# round} other {# rounds}} missed this year", overview.missed)
            : overview.owedNow > 0
              ? t(
                  "{0} waiting on this period's draw",
                  overview.owedNow === 1 ? t("A pool is") : t("Pools are"),
                )
              : overview.activePools === 0
                ? t("No active pool to draw from")
                : t("Every current round is drawn")
        }
      />

      <KpiStripItem
        label={t("Rounds this year")}
        info={
          <InfoPopover title={t("Rounds this year")}>
            {t(
              "Draws made this year that were not voided. A draft round can still change; a final one is the record.",
            )}
          </InfoPopover>
        }
        value={<NumberFlow value={overview.roundsThisYear} aria-label={t("Rounds this year")} />}
        sub={
          overview.roundsThisYear === 0
            ? t("Nothing drawn yet this year")
            : t("{0} final · {1} draft", overview.finalRounds, overview.draftRounds)
        }
      />

      <KpiCard span={2}>
        <KpiHeader
          label={t("Selected this year")}
          info={
            <InfoPopover title={t("Selected this year")}>
              {t(
                "Drivers picked across this year's rounds against what those rounds asked for. Short means a pool was smaller than its target.",
              )}
            </InfoPopover>
          }
          right={
            target > 0 && selected < target ? (
              <Badge variant="warning">{t("Short of target")}</Badge>
            ) : null
          }
        />
        <div className="flex items-baseline gap-1">
          <NumberFlow
            value={selected}
            className={KPI_VALUE_CLASS}
            aria-label={t("Selected this year")}
          />
          {target > 0 ? (
            <span className="text-muted-foreground text-xs tabular-nums">/ {target}</span>
          ) : null}
        </div>
        <div className="mt-auto flex flex-col gap-1">
          <RateLine
            label={t("Drug")}
            selected={overview.drugSelected}
            target={overview.drugTarget}
          />
          <RateLine
            label={t("Alcohol")}
            selected={overview.alcoholSelected}
            target={overview.alcoholTarget}
          />
        </div>
      </KpiCard>

      <KpiStripItem
        label={t("In the hat")}
        info={
          <InfoPopover title={t("In the hat")}>
            {t(
              "Drivers eligible at the most recent draw. It changes as people join and leave the pool.",
            )}
          </InfoPopover>
        }
        value={
          overview.lastPoolSize === null ? (
            <span className="text-muted-foreground" aria-label={t("In the hat")}>
              —
            </span>
          ) : (
            <NumberFlow value={overview.lastPoolSize} aria-label={t("In the hat")} />
          )
        }
        sub={
          <>
            <span className="block truncate">
              {overview.lastDrawnAt
                ? t("At the last draw, {0}", formatUnixDate(overview.lastDrawnAt))
                : t("No round has been drawn yet")}
            </span>
            <span className="block truncate">
              {t(
                "{0, plural, one {# active pool} other {# active pools}}{1}",
                overview.activePools,
                overview.belowMinimum > 0
                  ? ` ${t("· {0} below the DOT minimum", overview.belowMinimum)}`
                  : "",
              )}
            </span>
          </>
        }
      />
    </KpiStrip>
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
      className="grid grid-cols-[3.25rem_minmax(0,1fr)_auto] items-center gap-2 text-xs"
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
