import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiCard, KpiHeader } from "@/components/kpi/kpi-card";
import { KPI_VALUE_CLASS, KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type { RotaBoard, ShiftSwapRow } from "@/lib/graphql/scheduling";
import {
  coverageByDay,
  coverageOn,
  rotaComposition,
  summarizeSwaps,
  unrosteredWorkers,
} from "@/lib/scheduling-board";
import NumberFlow from "@number-flow/react";
import { CompositionBar } from "@trenova/shared/components/ui/composition-bar";
import { RingGauge } from "@trenova/shared/components/ui/ring-gauge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useMemo } from "react";

type SchedulingOverviewProps = {
  rota: RotaBoard | undefined;
  swaps: readonly ShiftSwapRow[] | undefined;
  today: number;
  showSwaps: boolean;
};

/**
 * The board in four numbers: who is on it, whether today is covered, what is
 * rostered wrong, and what is waiting on the office. Read from the same rota
 * the board draws, so the strip and the grid never disagree.
 */
export function SchedulingOverview({ rota, swaps, today, showSwaps }: SchedulingOverviewProps) {
  const t = useT();

  const rows = rota?.rows;
  const composition = useMemo(() => rotaComposition(rows ?? []), [rows]);
  const coverage = useMemo(() => coverageByDay(rows ?? []), [rows]);
  const todayCover = useMemo(() => coverageOn(coverage, today), [coverage, today]);
  const unrostered = useMemo(() => unrosteredWorkers(rows ?? []).length, [rows]);
  const swapSummary = useMemo(() => summarizeSwaps(swaps ?? []), [swaps]);
  const conflicts = rota?.conflicts ?? 0;

  return (
    <KpiStrip>
      <KpiCard span={2}>
        <KpiHeader
          label={t("On the board")}
          info={
            <InfoPopover title={t("On the board")}>
              {t(
                "Rows on the rota for these weeks: everyone on a pattern or with a shift, plus people with nothing rostered, so the gaps show.",
              )}
            </InfoPopover>
          }
        />
        {rota ? (
          <NumberFlow
            value={rota.rows.length}
            className={KPI_VALUE_CLASS}
            aria-label={t("On the board")}
          />
        ) : (
          <Skeleton className="h-6 w-10" />
        )}
        <CompositionBar
          size="sm"
          className="mt-auto"
          aria-label={t("Person-days on the board")}
          segments={[
            { key: "working", label: t("Working"), value: composition.working },
            { key: "off", label: t("Off"), value: composition.off },
            { key: "away", label: t("Away"), value: composition.away },
          ]}
        />
      </KpiCard>

      <KpiStripItem
        label={t("Cover today")}
        info={
          <InfoPopover title={t("Cover today")}>
            {t(
              "How many people have a shift today, read against the busiest day on the board. The ring is relative because a small yard and a large terminal share no number.",
            )}
          </InfoPopover>
        }
        value={
          rota ? (
            todayCover ? (
              <span className="flex items-center gap-2">
                <RingGauge
                  value={todayCover.expected > 0 ? todayCover.covered / todayCover.expected : 0}
                  size={24}
                  strokeWidth={3}
                  tone="brand"
                  aria-label={t("Share of today's rostered people who can work")}
                />
                <span className="flex items-baseline gap-1">
                  <NumberFlow value={todayCover.covered} aria-label={t("Cover today")} />
                  <span className="text-muted-foreground text-xs font-normal">
                    {t("of {0}", todayCover.expected)}
                  </span>
                </span>
              </span>
            ) : (
              <span className="text-muted-foreground">—</span>
            )
          ) : (
            <Skeleton className="h-6 w-24" />
          )
        }
        sub={
          todayCover
            ? describeToday(todayCover.timeOff, todayCover.leave, todayCover.conflicts)
            : t("Today is outside the weeks shown")
        }
      />

      <KpiStripItem
        label={t("Conflicts")}
        tone={conflicts > 0 ? "danger" : undefined}
        info={
          <InfoPopover title={t("Conflicts")}>
            {t(
              "Rostered days the person cannot work: time off, leave or a stated unavailability won over the pattern.",
            )}
          </InfoPopover>
        }
        value={
          rota ? (
            <NumberFlow value={conflicts} aria-label={t("Conflicts")} />
          ) : (
            <Skeleton className="h-6 w-10" />
          )
        }
        sub={
          conflicts > 0
            ? t("Rostered on a day they cannot work")
            : unrostered > 0
              ? t("Nothing rostered wrong. {0} on no shift.", unrostered)
              : t("Nothing rostered wrong")
        }
      />

      {showSwaps ? (
        <KpiStripItem
          label={t("Swaps waiting on you")}
          info={
            <InfoPopover title={t("Swaps waiting on you")}>
              {t(
                "Shift swaps the colleague has accepted that still need an office decision. Ones the colleague has not answered are not counted.",
              )}
            </InfoPopover>
          }
          value={
            swaps ? (
              <NumberFlow
                value={swapSummary.awaitingOffice}
                aria-label={t("Swaps waiting on you")}
              />
            ) : (
              <Skeleton className="h-6 w-10" />
            )
          }
          sub={
            swapSummary.awaitingColleague > 0
              ? t("{0} more still waiting on a colleague", swapSummary.awaitingColleague)
              : t("Accepted by the colleague, needing the office's say")
          }
        />
      ) : null}
    </KpiStrip>
  );
}

function describeToday(timeOff: number, leave: number, conflicts: number): string {
  const parts: string[] = [];
  if (timeOff > 0) parts.push(`${timeOff} on time off`);
  if (leave > 0) parts.push(`${leave} on leave`);
  if (conflicts > 0) parts.push(`${conflicts} in conflict`);
  return parts.length > 0 ? parts.join(" · ") : "Everyone rostered today can work it";
}
