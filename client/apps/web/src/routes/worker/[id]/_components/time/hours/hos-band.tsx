import { HourRuler } from "@/components/hos/hour-ruler";
import type { WorkerHosClocks } from "@/lib/graphql/telematics";
import { dutyStatusLabel } from "@/lib/hos";
import { AlertOctagonIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, formatUnixTime } from "@trenova/shared/lib/date";
import { HOUR_MS } from "@/lib/time-units";
import { dutyStatusVariant } from "@/lib/hos";
import { formatHoursMinutes, limitHours } from "@/lib/hours-format";

const DEFAULT_LIMITS = {
  break: 8 * HOUR_MS,
  drive: 11 * HOUR_MS,
  shift: 14 * HOUR_MS,
  cycle: 70 * HOUR_MS,
};

export type HosBandProps = {
  clocks: WorkerHosClocks;
  /** The vehicle on today's open log segment, when the provider named it. */
  vehicleName: string | null;
  /** Driving logged today, from today's segments; null until they load. */
  drivenTodayMs: number | null;
};

/**
 * Hours of service right now: the duty status and since when, then the four clocks as ruled
 * bars with hour ticks, the cycle's every ten hours.
 */
export function HosBand({ clocks, vehicleName, drivenTodayMs }: HosBandProps) {
  const t = useT();
  const breakLimit = clocks.breakLimitMs || DEFAULT_LIMITS.break;
  const driveLimit = clocks.driveLimitMs || DEFAULT_LIMITS.drive;
  const shiftLimit = clocks.shiftLimitMs || DEFAULT_LIMITS.shift;
  const cycleLimit = clocks.cycleLimitMs || DEFAULT_LIMITS.cycle;
  const context = [
    clocks.dutyStatusSince ? t("since {0}", formatUnixTime(clocks.dutyStatusSince)) : null,
    vehicleName,
    t("as of {0}", formatUnixTime(clocks.recordedAt)),
  ].filter((part): part is string => part !== null);
  const cycleNote = clocks.cycleStartedAt
    ? t(
        "{0} available tomorrow · cycle began {1}",
        formatHoursMinutes(t, clocks.cycleTomorrowMs),
        formatUnixDateMedium(clocks.cycleStartedAt),
      )
    : t("{0} available tomorrow", formatHoursMinutes(t, clocks.cycleTomorrowMs));
  const violations = [
    clocks.shiftDrivingViolationMs > 0
      ? t(
          "Shift driving limit exceeded by {0}.",
          formatHoursMinutes(t, clocks.shiftDrivingViolationMs),
        )
      : null,
    clocks.cycleViolationMs > 0
      ? t("Cycle limit exceeded by {0}.", formatHoursMinutes(t, clocks.cycleViolationMs))
      : null,
  ].filter((part): part is string => part !== null);

  return (
    <section
      aria-label={t("Hours of service right now")}
      className="flex flex-col rounded-lg border bg-card"
    >
      <div className="flex min-h-11 flex-wrap items-center gap-2 border-b border-border-subtle px-4 py-2">
        <Badge variant={dutyStatusVariant(clocks.dutyStatus)} dot>
          {clocks.dutyStatus ? dutyStatusLabel(clocks.dutyStatus) : t("Status not reported")}
        </Badge>
        <span className="text-muted-foreground tabular-nums">{context.join(" · ")}</span>
      </div>
      {violations.length > 0 ? (
        <div className="border-b border-border-subtle px-4 py-3">
          <Alert variant="destructive" size="sm">
            <AlertOctagonIcon />
            <AlertTitle>{t("In violation now")}</AlertTitle>
            <AlertDescription>{violations.join(" ")}</AlertDescription>
          </Alert>
        </div>
      ) : null}
      <div className="@container/hos">
        <div className="grid grid-cols-1 @min-[35rem]/hos:grid-cols-2 @min-[62rem]/hos:grid-cols-4">
          <HourRuler
            size="large"
            name={t("Break")}
            remainingMs={clocks.breakRemainingMs}
            limitMs={breakLimit}
            caption={t("of {0}h until break", limitHours(breakLimit))}
            className="border-border-subtle px-4 pt-3 pb-3.5"
          />
          <HourRuler
            size="large"
            name={t("Drive")}
            remainingMs={clocks.driveRemainingMs}
            limitMs={driveLimit}
            caption={t("of {0}h driving", limitHours(driveLimit))}
            note={
              drivenTodayMs !== null
                ? t("{0} driven today", formatHoursMinutes(t, drivenTodayMs))
                : undefined
            }
            className="border-t border-border-subtle px-4 pt-3 pb-3.5 @min-[35rem]/hos:border-t-0 @min-[35rem]/hos:border-l"
          />
          <HourRuler
            size="large"
            name={t("Shift")}
            remainingMs={clocks.shiftRemainingMs}
            limitMs={shiftLimit}
            caption={t("of {0}h shift", limitHours(shiftLimit))}
            className="border-t border-border-subtle px-4 pt-3 pb-3.5 @min-[62rem]/hos:border-t-0 @min-[62rem]/hos:border-l"
          />
          <HourRuler
            size="large"
            name={t("Cycle")}
            remainingMs={clocks.cycleRemainingMs}
            limitMs={cycleLimit}
            tickMs={10 * HOUR_MS}
            caption={t("of {0}h cycle", limitHours(cycleLimit))}
            note={cycleNote}
            className="border-t border-border-subtle px-4 pt-3 pb-3.5 @min-[35rem]/hos:border-l @min-[62rem]/hos:border-t-0"
          />
        </div>
      </div>
    </section>
  );
}
