import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiCard, KpiHeader, KpiSub } from "@/components/kpi/kpi-card";
import type { OpenTimeEntryRow, TimesheetRow } from "@/lib/graphql/timesheet";
import {
  isOverlong,
  longestRunning,
  rankRunning,
  summarizeSheets,
  waitingDays,
} from "@/lib/time-attendance";
import NumberFlow from "@number-flow/react";
import { CompositionBar } from "@trenova/shared/components/ui/composition-bar";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatHours } from "@trenova/shared/lib/timesheet";
import { cn } from "@trenova/shared/lib/utils";
import { BanknoteIcon, CalendarRangeIcon, ClipboardCheckIcon, TimerIcon } from "lucide-react";
import { useMemo } from "react";

const VALUE_CLASS = "font-mono text-[26px] leading-none font-semibold tracking-tight tabular-nums";

type OverviewProps = {
  awaiting: readonly TimesheetRow[] | undefined;
  thisWeek: readonly TimesheetRow[] | undefined;
  unpaid: readonly TimesheetRow[] | undefined;
  openEntries: readonly OpenTimeEntryRow[] | undefined;
  now: number;
  showPayroll: boolean;
};

/**
 * The four numbers a manager opens the page for: what is waiting on them, who
 * is punched in this minute, what this week is adding up to, and what payroll
 * has not been sent yet. Each one is read from the list the tab below shows,
 * so the headline never disagrees with the detail. The corner of each tile
 * carries the one thing that makes the number urgent: the oldest wait, the
 * punch that has run too long, the people behind the hours.
 */
export function TimeAttendanceOverview({
  awaiting,
  thisWeek,
  unpaid,
  openEntries,
  now,
  showPayroll,
}: OverviewProps) {
  const t = useT();

  const awaitingSummary = useMemo(() => summarizeSheets(awaiting ?? []), [awaiting]);
  const weekSummary = useMemo(() => summarizeSheets(thisWeek ?? []), [thisWeek]);
  const unpaidSummary = useMemo(() => summarizeSheets(unpaid ?? []), [unpaid]);
  const longest = useMemo(() => longestRunning(openEntries ?? [], now), [openEntries, now]);
  const overlong = useMemo(
    () =>
      rankRunning(openEntries ?? [], now).filter((row) => isOverlong(row.runningMinutes)).length,
    [openEntries, now],
  );
  const oldestWait = useMemo(
    () =>
      (awaiting ?? []).reduce(
        (oldest, sheet) =>
          sheet.submittedAt ? Math.max(oldest, waitingDays(sheet.submittedAt, now)) : oldest,
        0,
      ),
    [awaiting, now],
  );

  return (
    <div
      className={cn("grid grid-cols-4 gap-3", showPayroll ? "lg:grid-cols-8" : "lg:grid-cols-6")}
    >
      <KpiCard span={2}>
        <KpiHeader
          icon={<ClipboardCheckIcon className="size-[11px]" />}
          label={t("Awaiting approval")}
          info={
            <InfoPopover title={t("Awaiting approval")}>
              {t(
                "Timesheets submitted and waiting on a manager. The hours are the totals frozen when each week was submitted.",
              )}
            </InfoPopover>
          }
          right={
            oldestWait > 0 ? (
              <Corner tone={oldestWait >= 3 ? "warning" : "muted"}>
                {t("oldest {0}d", oldestWait)}
              </Corner>
            ) : null
          }
        />
        {awaiting ? (
          <NumberFlow
            value={awaitingSummary.count}
            className={VALUE_CLASS}
            aria-label={t("Awaiting approval")}
          />
        ) : (
          <Skeleton className="h-6.5 w-10" />
        )}
        <KpiSub>
          {awaitingSummary.count === 0
            ? t("Nothing waiting on you")
            : t(
                "{0} across {1} {2}",
                formatHours(awaitingSummary.totalMinutes),
                awaitingSummary.workers,
                awaitingSummary.workers === 1 ? "person" : "people",
              )}
        </KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<TimerIcon className="size-[11px]" />}
          label={t("On the clock now")}
          info={
            <InfoPopover title={t("On the clock now")}>
              {t(
                "Punches with no clock-out yet. A punch that has run past a working day is more likely forgotten than worked.",
              )}
            </InfoPopover>
          }
          right={
            overlong > 0 ? <Corner tone="warning">{t("{0} past 12h", overlong)}</Corner> : null
          }
        />
        {openEntries ? (
          <NumberFlow
            value={openEntries.length}
            className={VALUE_CLASS}
            aria-label={t("On the clock now")}
          />
        ) : (
          <Skeleton className="h-6.5 w-10" />
        )}
        <KpiSub>
          {longest
            ? t(
                "Longest running: {0}, {1}",
                workerName(longest.entry),
                formatHours(longest.runningMinutes),
              )
            : t("Nobody is punched in")}
        </KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<CalendarRangeIcon className="size-[11px]" />}
          label={t("This week so far")}
          info={
            <InfoPopover title={t("This week so far")}>
              {t(
                "Regular, overtime and paid leave across every timesheet for the current week, whatever state each is in.",
              )}
            </InfoPopover>
          }
          right={
            weekSummary.workers > 0 ? (
              <Corner tone="muted">
                {weekSummary.workers} {weekSummary.workers === 1 ? "person" : "people"}
              </Corner>
            ) : null
          }
        />
        {thisWeek ? (
          <span className={VALUE_CLASS}>{formatHours(weekSummary.totalMinutes)}</span>
        ) : (
          <Skeleton className="h-6.5 w-16" />
        )}
        <CompositionBar
          size="sm"
          className="mt-auto"
          aria-label={t("This week's hours")}
          formatValue={formatHours}
          segments={[
            { key: "regular", label: t("Regular"), value: weekSummary.regularMinutes },
            { key: "overtime", label: t("Overtime"), value: weekSummary.overtimeMinutes },
            { key: "leave", label: t("Leave"), value: weekSummary.paidLeaveMinutes },
          ]}
        />
      </KpiCard>

      {showPayroll ? (
        <KpiCard span={2}>
          <KpiHeader
            icon={<BanknoteIcon className="size-[11px]" />}
            label={t("Approved, not paid")}
            info={
              <InfoPopover title={t("Approved, not paid")}>
                {t("Weeks a manager has approved that payroll has not yet locked into an export.")}
              </InfoPopover>
            }
            right={
              unpaidSummary.overtimeWeeks > 0 ? (
                <Corner tone="muted">{t("{0} with OT", unpaidSummary.overtimeWeeks)}</Corner>
              ) : null
            }
          />
          {unpaid ? (
            <NumberFlow
              value={unpaidSummary.count}
              className={VALUE_CLASS}
              aria-label={t("Approved, not paid")}
            />
          ) : (
            <Skeleton className="h-6.5 w-10" />
          )}
          <KpiSub>
            {unpaidSummary.count === 0
              ? t("Every approved week has gone to payroll")
              : t("{0} waiting for a run", formatHours(unpaidSummary.totalMinutes))}
          </KpiSub>
        </KpiCard>
      ) : null}
    </div>
  );
}

function Corner({ tone, children }: { tone: "muted" | "warning"; children: React.ReactNode }) {
  return (
    <span
      className={cn(
        "text-2xs leading-none tabular-nums",
        tone === "warning" ? "text-warning-foreground" : "text-muted-foreground",
      )}
    >
      {children}
    </span>
  );
}

function workerName(entry: OpenTimeEntryRow): string {
  return entry.worker ? `${entry.worker.firstName} ${entry.worker.lastName}` : "somebody";
}
