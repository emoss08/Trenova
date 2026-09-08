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
  const selected = overview.drugSelected + overview.alcoholSelected;
  const target = overview.drugTarget + overview.alcoholTarget;

  return (
    <div className="grid grid-cols-4 gap-3 lg:grid-cols-8">
      <KpiCard span={2}>
        <KpiHeader
          icon={<CalendarClockIcon className="size-[11px]" />}
          label="Owed now"
          info={
            <InfoPopover title="Owed now">
              {
                "Active pools whose current round has not been drawn. Missed means a past round of this year was never drawn, which a DOT audit will find."
              }
            </InfoPopover>
          }
          right={overview.missed > 0 ? <Badge variant="inactive">Missed</Badge> : null}
        />
        <NumberFlow value={overview.owedNow} className={VALUE_CLASS} aria-label="Owed now" />
        <KpiSub>
          {overview.missed > 0
            ? `${overview.missed} round${overview.missed === 1 ? "" : "s"} missed this year`
            : overview.owedNow > 0
              ? `${overview.owedNow === 1 ? "A pool is" : "Pools are"} waiting on this period's draw`
              : overview.activePools === 0
                ? "No active pool to draw from"
                : "Every current round is drawn"}
        </KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<CalendarCheckIcon className="size-[11px]" />}
          label="Rounds this year"
          info={
            <InfoPopover title="Rounds this year">
              {
                "Draws made this year that were not voided. A draft round can still change; a final one is the record."
              }
            </InfoPopover>
          }
        />
        <NumberFlow
          value={overview.roundsThisYear}
          className={VALUE_CLASS}
          aria-label="Rounds this year"
        />
        <KpiSub>
          {overview.roundsThisYear === 0
            ? "Nothing drawn yet this year"
            : `${overview.finalRounds} final · ${overview.draftRounds} draft`}
        </KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<FlaskConicalIcon className="size-[11px]" />}
          label="Selected this year"
          info={
            <InfoPopover title="Selected this year">
              {
                "Drivers picked across this year's rounds against what those rounds asked for. Short means a pool was smaller than its target."
              }
            </InfoPopover>
          }
          right={
            target > 0 && selected < target ? (
              <Badge variant="warning">Short of target</Badge>
            ) : null
          }
        />
        <div className="flex items-baseline gap-1">
          <NumberFlow value={selected} className={VALUE_CLASS} aria-label="Selected this year" />
          {target > 0 ? (
            <span className="text-muted-foreground font-mono text-[11px] tabular-nums">
              / {target}
            </span>
          ) : null}
        </div>
        <div className="mt-auto flex flex-col gap-1">
          <RateLine label="Drug" selected={overview.drugSelected} target={overview.drugTarget} />
          <RateLine
            label="Alcohol"
            selected={overview.alcoholSelected}
            target={overview.alcoholTarget}
          />
        </div>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<UsersIcon className="size-[11px]" />}
          label="In the hat"
          info={
            <InfoPopover title="In the hat">
              {
                "Drivers eligible at the most recent draw. It changes as people join and leave the pool."
              }
            </InfoPopover>
          }
        />
        {overview.lastPoolSize === null ? (
          <span className={cn(VALUE_CLASS, "text-muted-foreground")} aria-label="In the hat">
            —
          </span>
        ) : (
          <NumberFlow
            value={overview.lastPoolSize}
            className={VALUE_CLASS}
            aria-label="In the hat"
          />
        )}
        <KpiSub>
          {overview.lastDrawnAt
            ? `At the last draw, ${formatUnixDate(overview.lastDrawnAt)}`
            : "No round has been drawn yet"}
        </KpiSub>
        <KpiSub>
          {overview.activePools} active pool{overview.activePools === 1 ? "" : "s"}
          {overview.belowMinimum > 0 ? ` · ${overview.belowMinimum} below the DOT minimum` : ""}
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
