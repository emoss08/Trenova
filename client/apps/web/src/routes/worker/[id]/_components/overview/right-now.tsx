import { SectionPanel } from "@/components/section-panel";
import type { WorkerOverview } from "@/lib/graphql/worker-overview";
import { dutyStatusLabel, dutyStatusVariant } from "@/lib/hos";
import { formatHoursMinutes } from "@/lib/hours-format";
import { PlanFeature } from "@/lib/plan-feature";
import type { WorkerTarget } from "@/routes/worker/use-worker-state";
import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  daysUntil,
  formatRange,
  formatUnixDateTimeMedium,
  formatUnixMonthDay,
  formatUnixTime,
} from "@trenova/shared/lib/date";
import { formatNumber } from "@trenova/shared/i18n/format";
import { cn } from "@trenova/shared/lib/utils";
import { CoffeeIcon } from "@trenova/shared/components/icons";
import { HourRuler } from "@/components/hos/hour-ruler";
import { withoutRecordIds } from "@/lib/record-ids";
import { HOUR_MS } from "@/lib/time-units";
import { useNow } from "@trenova/shared/hooks/use-now";
import { ComplianceBadge } from "../compliance-badge";

type Hos = NonNullable<WorkerOverview["hos"]>;

/** Whether two instants fall on the same day in the reader's zone, where the time is shown. */
function sameDay(a: number, b: number): boolean {
  return daysUntil(a, { now: b }) === 0;
}

const CYCLE_TICK_MS = 10 * HOUR_MS;

type Clock = {
  key: string;
  name: string;
  remainingMs: number | null;
  limitMs: number | null;
  tickMs?: number;
};

/**
 * One clock: a ruler when both what is left and the limit are stored, a line of text when only
 * what is left is, and nothing when neither is.
 */
function ClockRow({ clock }: { clock: Clock }) {
  const t = useT();
  if (clock.remainingMs === null) return null;
  if (clock.limitMs === null || clock.limitMs <= 0) {
    return (
      <p data-clock={clock.key} className="text-sm tabular-nums">
        {t("{0}: {1} left", clock.name, formatHoursMinutes(t, clock.remainingMs))}
      </p>
    );
  }
  return (
    <div data-clock={clock.key}>
      <HourRuler
        name={clock.name}
        remainingMs={clock.remainingMs}
        limitMs={clock.limitMs}
        tickMs={clock.tickMs}
        caption={t(
          "left of {0}h",
          formatNumber(clock.limitMs / HOUR_MS, { maximumFractionDigits: 1 }),
        )}
      />
    </div>
  );
}

function dutyLine(hos: Hos, asOf: number, t: TranslateFn): string | null {
  const since = hos.dutyStatusSince;
  const when = since
    ? sameDay(since, asOf)
      ? formatUnixTime(since)
      : formatUnixDateTimeMedium(since)
    : null;
  const tractor = hos.tractorCode ? withoutRecordIds(hos.tractorCode).trim() : "";
  if (when && tractor) return t("since {0} · Tractor {1}", when, tractor);
  if (when) return t("since {0}", when);
  if (tractor) return t("Tractor {0}", tractor);
  return null;
}

function HosLine({ hos, asOf }: { hos: Hos; asOf: number }) {
  const t = useT();
  switch (hos.link) {
    case "NotConnected":
      return (
        <p className="text-xs text-muted-foreground">
          {t("No telematics provider is connected, so hours of service are not shown.")}
        </p>
      );
    case "NotLinked":
      return (
        <p className="text-xs text-muted-foreground">
          {t("This driver is not linked to a driver in telematics yet.")}
        </p>
      );
    case "NoData":
      return (
        <p className="text-xs text-muted-foreground">
          {t("Telematics has not reported this driver's hours yet.")}
        </p>
      );
    case "Reporting": {
      const line = dutyLine(hos, asOf, t);
      const clocks: Clock[] = [
        {
          key: "drive",
          name: t("Drive"),
          remainingMs: hos.driveRemainingMs,
          limitMs: hos.driveLimitMs,
        },
        {
          key: "shift",
          name: t("Shift"),
          remainingMs: hos.shiftRemainingMs,
          limitMs: hos.shiftLimitMs,
        },
        {
          key: "cycle",
          name: t("Cycle"),
          remainingMs: hos.cycleRemainingMs,
          limitMs: hos.cycleLimitMs,
          tickMs: CYCLE_TICK_MS,
        },
      ];
      const shown = clocks.filter((clock) => clock.remainingMs !== null);
      const breakLeft = hos.breakRemainingMs;
      return (
        <div className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <Badge dot variant={dutyStatusVariant(hos.dutyStatus)}>
              {hos.dutyStatus ? dutyStatusLabel(hos.dutyStatus) : t("Duty status unknown")}
            </Badge>
            {line ? (
              <span className="text-sm text-muted-foreground tabular-nums">{line}</span>
            ) : null}
          </div>
          {shown.length > 0 ? (
            <div className="flex flex-col gap-2.5">
              {shown.map((clock) => (
                <ClockRow key={clock.key} clock={clock} />
              ))}
            </div>
          ) : null}
          {breakLeft !== null ? (
            <p
              data-break
              className={cn(
                "flex items-center gap-1.5 text-xs tabular-nums",
                breakLeft <= 0 ? "text-warning" : "text-muted-foreground",
              )}
            >
              <CoffeeIcon aria-hidden className="size-3" />
              {breakLeft <= 0
                ? t("Break due now")
                : t("Break due in {0}", formatHoursMinutes(t, breakLeft))}
            </p>
          ) : null}
        </div>
      );
    }
  }
}

type NextTimeOff = NonNullable<WorkerOverview["nextTimeOff"]>;

function NextTimeOffValue({
  timeOff,
  canOpen,
  onOpen,
}: {
  timeOff: NextTimeOff | null;
  canOpen: boolean;
  onOpen: (target: WorkerTarget) => void;
}) {
  const t = useT();
  if (!timeOff) return <span className="text-muted-foreground">{t("None requested")}</span>;
  const text = t(
    "{0} · {1}",
    formatRange(timeOff.startDate, timeOff.endDate, { year: "never", join: "words" }),
    timeOff.status === "Approved" ? t("approved") : t("requested"),
  );
  if (!canOpen) return <span className="tabular-nums">{text}</span>;
  return (
    <button
      type="button"
      onClick={() => onOpen({ tab: "time", view: "time-off", timeOff: timeOff.id })}
      className="rounded-xs ui-focus-ring tabular-nums underline-offset-2 hover:underline"
    >
      {text}
    </button>
  );
}

/** The day they were last home and how many days out since, moving at midnight. */
function LastHomeLine({ lastHome }: { lastHome: number }) {
  const t = useT();
  const now = useNow("minute");
  return (
    <span className="tabular-nums">
      {t(
        "{0} · {1, plural, one {# day out} other {# days out}}",
        formatUnixMonthDay(lastHome),
        Math.max(0, -daysUntil(lastHome, { now })),
      )}
    </span>
  );
}

/**
 * Where the driver is in their day, from the hours of service stored for them (this read never
 * calls the provider): duty status and since when, the tractor, the drive, shift and cycle
 * clocks and the next break; then when they were last home and their next time off. A plan
 * without telematics shows only the last two.
 */
export function RightNowPanel({
  overview,
  canOpenHours,
  canOpenTimeOff,
  onOpen,
}: {
  overview: WorkerOverview;
  canOpenHours: boolean;
  /** Time › Time off is drawable: the next time off is shown and opens there. */
  canOpenTimeOff: boolean;
  onOpen: (target: WorkerTarget) => void;
}) {
  const t = useT();
  const { hos, worker, asOf } = overview;
  const lastHome = worker.lastHomeAt;
  return (
    <SectionPanel
      title={t("Right now")}
      hint={
        hos?.link === "Reporting" && hos.recordedAt
          ? t("as of {0}", formatUnixTime(hos.recordedAt))
          : undefined
      }
      action={
        canOpenHours && hos ? (
          <button
            type="button"
            onClick={() => onOpen({ tab: "time", view: "hours-of-service" })}
            className="rounded-sm ui-focus-ring text-xs text-muted-foreground transition-colors hover:text-foreground hover:underline"
          >
            {t("Hours")}
          </button>
        ) : undefined
      }
    >
      {hos ? (
        <div className="border-b border-border-subtle p-3">
          <HosLine hos={hos} asOf={asOf} />
        </div>
      ) : null}
      <DescriptionList layout="split" className="px-3 py-1">
        <DescriptionItem label={t("Last home")}>
          {lastHome ? <LastHomeLine lastHome={lastHome} /> : <DescriptionEmpty />}
        </DescriptionItem>
        {canOpenTimeOff ? (
          <DescriptionItem label={t("Next time off")}>
            <NextTimeOffValue
              timeOff={overview.nextTimeOff}
              canOpen={canOpenTimeOff}
              onOpen={onOpen}
            />
          </DescriptionItem>
        ) : null}
      </DescriptionList>
    </SectionPanel>
  );
}

/** Whether dispatch can put the worker on a load, and the facts it decides on. */
export function AssignmentPanel({ overview }: { overview: WorkerOverview }) {
  const t = useT();
  const complianceOnPlan = usePlanFeature(PlanFeature.WorkforceCompliance).available;
  const { worker } = overview;
  const profile = worker.profile;
  const compliance = overview.credentials?.complianceStatus ?? profile?.complianceStatus ?? null;
  return (
    <SectionPanel title={t("Assignment")}>
      <DescriptionList layout="split" className="px-3 py-1">
        <DescriptionItem label={t("Dispatch")}>
          {worker.canBeAssigned ? t("Assignable") : t("Held")}
        </DescriptionItem>
        <DescriptionItem label={t("Available for dispatch")}>
          {worker.availableForDispatch ? t("Yes") : t("No")}
        </DescriptionItem>
        <DescriptionItem label={t("Qualified")}>
          {profile ? profile.isQualified ? t("Yes") : t("No") : <DescriptionEmpty />}
        </DescriptionItem>
        {complianceOnPlan ? (
          <DescriptionItem label={t("Compliance")}>
            {!profile ? (
              <DescriptionEmpty />
            ) : compliance === null ? (
              <WithheldValue plan={PlanFeature.WorkforceCompliance} />
            ) : (
              <ComplianceBadge status={compliance} />
            )}
          </DescriptionItem>
        ) : null}
        <DescriptionItem label={t("ELD")}>
          {profile ? profile.eldExempt ? t("Exempt") : t("Required") : <DescriptionEmpty />}
        </DescriptionItem>
      </DescriptionList>
    </SectionPanel>
  );
}
