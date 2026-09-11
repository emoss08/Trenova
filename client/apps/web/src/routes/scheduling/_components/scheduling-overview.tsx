import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiCard, KpiHeader, KpiSub } from "@/components/kpi/kpi-card";
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
import { cn } from "@trenova/shared/lib/utils";
import { AlertTriangleIcon, CalendarCheckIcon, RepeatIcon, UsersIcon } from "lucide-react";
import { useMemo } from "react";

const VALUE_CLASS = "font-mono text-[26px] leading-none font-semibold tracking-tight tabular-nums";

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
    <div className={cn("grid grid-cols-4 gap-3", showSwaps ? "lg:grid-cols-8" : "lg:grid-cols-6")}>
      <KpiCard span={2}>
        <KpiHeader
          icon={<UsersIcon className="size-[11px]" />}
          label={t("On the board")}
          info={
            <InfoPopover title={t("On the board")}>
              {
                "Rows on the rota for these weeks: everyone on a pattern or with a shift, plus people with nothing rostered, so the gaps show."
              }
            </InfoPopover>
          }
        />
        {rota ? (
          <NumberFlow value={rota.rows.length} className={VALUE_CLASS} aria-label={t("On the board")} />
        ) : (
          <Skeleton className="h-6.5 w-10" />
        )}
        <CompositionBar
          size="sm"
          className="mt-auto"
          aria-label={t("Person-days on the board")}
          segments={[
            { key: "working", label: "Working", value: composition.working },
            { key: "off", label: "Off", value: composition.off },
            { key: "away", label: "Away", value: composition.away },
          ]}
        />
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<CalendarCheckIcon className="size-[11px]" />}
          label={t("Cover today")}
          info={
            <InfoPopover title={t("Cover today")}>
              {
                "How many people have a shift today, read against the busiest day on the board. The ring is relative because a small yard and a large terminal share no number."
              }
            </InfoPopover>
          }
        />
        {rota ? (
          todayCover ? (
            <div className="flex items-center gap-2.5">
              <RingGauge
                value={todayCover.expected > 0 ? todayCover.covered / todayCover.expected : 0}
                size={40}
                strokeWidth={4}
                tone="brand"
                aria-label={t("Share of today's rostered people who can work")}
              />
              <div className="flex items-baseline gap-1">
                <NumberFlow
                  value={todayCover.covered}
                  className={VALUE_CLASS}
                  aria-label={t("Cover today")}
                />
                <span className="text-muted-foreground font-mono text-[11px]">
                  {t("of {0}", todayCover.expected)}
                </span>
              </div>
            </div>
          ) : (
            <span className={cn(VALUE_CLASS, "text-muted-foreground")}>—</span>
          )
        ) : (
          <Skeleton className="h-10 w-24" />
        )}
        <KpiSub>
          {todayCover
            ? describeToday(todayCover.timeOff, todayCover.leave, todayCover.conflicts)
            : "Today is outside the weeks shown"}
        </KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<AlertTriangleIcon className="size-[11px]" />}
          label={t("Conflicts")}
          info={
            <InfoPopover title={t("Conflicts")}>
              {
                "Rostered days the person cannot work: time off, leave or a stated unavailability won over the pattern."
              }
            </InfoPopover>
          }
        />
        {rota ? (
          <NumberFlow
            value={conflicts}
            className={cn(VALUE_CLASS, conflicts > 0 && "text-destructive")}
            aria-label={t("Conflicts")}
          />
        ) : (
          <Skeleton className="h-6.5 w-10" />
        )}
        <KpiSub>
          {conflicts > 0
            ? "Rostered on a day they cannot work"
            : unrostered > 0
              ? `Nothing rostered wrong. ${unrostered} on no shift.`
              : "Nothing rostered wrong"}
        </KpiSub>
      </KpiCard>

      {showSwaps ? (
        <KpiCard span={2}>
          <KpiHeader
            icon={<RepeatIcon className="size-[11px]" />}
            label={t("Swaps waiting on you")}
            info={
              <InfoPopover title={t("Swaps waiting on you")}>
                {
                  "Shift swaps the colleague has accepted that still need an office decision. Ones the colleague has not answered are not counted."
                }
              </InfoPopover>
            }
          />
          {swaps ? (
            <NumberFlow
              value={swapSummary.awaitingOffice}
              className={VALUE_CLASS}
              aria-label={t("Swaps waiting on you")}
            />
          ) : (
            <Skeleton className="h-6.5 w-10" />
          )}
          <KpiSub>
            {swapSummary.awaitingColleague > 0
              ? `${swapSummary.awaitingColleague} more still waiting on a colleague`
              : "Accepted by the colleague, needing the office's say"}
          </KpiSub>
        </KpiCard>
      ) : null}
    </div>
  );
}

function describeToday(timeOff: number, leave: number, conflicts: number): string {
  const parts: string[] = [];
  if (timeOff > 0) parts.push(`${timeOff} on time off`);
  if (leave > 0) parts.push(`${leave} on leave`);
  if (conflicts > 0) parts.push(`${conflicts} in conflict`);
  return parts.length > 0 ? parts.join(" · ") : "Everyone rostered today can work it";
}
