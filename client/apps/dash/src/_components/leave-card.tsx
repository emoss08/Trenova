import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Progress } from "@trenova/shared/components/ui/progress";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { fetchMyLeave } from "@trenova/shared/lib/graphql/driver-portal";
import {
  certificationOutstanding,
  certificationStatusLabel,
  entitlementUsedPercent,
  formatLeaveHours,
  leaveCaseStatusLabel,
  leaveCaseStatusTone,
  leaveFrequencyLabel,
  leaveTypeLabel,
  measurementMethodLabel,
} from "@trenova/shared/lib/leave";
import { useQuery } from "@tanstack/react-query";

/**
 * The driver's own FMLA standing. Shown only once they have a leave case: an
 * empty entitlement card reads as time they are owed rather than time they have
 * not started drawing on.
 */
export function LeaveCard() {
  const t = useT();

  const leave = useQuery({
    queryKey: ["dash-leave"],
    queryFn: ({ signal }) => fetchMyLeave({ signal }),
    staleTime: 60 * 1000,
  });

  if (leave.isPending) {
    return <Skeleton className="h-40 w-full rounded-2xl" />;
  }
  if (!leave.data || leave.data.cases.length === 0) {
    return null;
  }

  const { entitlement, cases } = leave.data;
  const usedPercent = entitlementUsedPercent(entitlement.usedHours, entitlement.totalHours);
  const owing = cases.filter((row) => certificationOutstanding(row.certificationStatus));

  return (
    <div className="border-border bg-card rounded-2xl border p-4" data-testid="leave-card">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm font-semibold">{t("Family & medical leave")}</p>
          <p className="text-muted-foreground text-xs">
            {t("{0} · {1} to {2}", measurementMethodLabel(entitlement.method), formatUnixDateMedium(entitlement.window.from), formatUnixDateMedium(entitlement.window.through))}
          </p>
        </div>
        {entitlement.exhausted ? <Badge variant="inactive">{t("Used up")}</Badge> : null}
      </div>

      <p className="mt-3 text-2xl font-semibold tabular-nums">
        {formatLeaveHours(entitlement.remainingHours)}
        <span className="text-muted-foreground ml-1 text-xs font-normal">{t("hours left")}</span>
      </p>
      <p className="text-muted-foreground text-xs">
        {t("{0} of {1} weeks", formatLeaveHours(entitlement.remainingWeeks), formatLeaveHours(entitlement.totalWeeks))}
      </p>
      <Progress value={usedPercent} className="mt-2" />

      {owing.length > 0 ? (
        <p className="border-warning/40 bg-warning/10 text-warning-foreground mt-3 rounded-lg border px-3 py-2 text-xs">
          {owing.length === 1
            ? t("Your carrier is waiting on a medical certification.")
            : t("Your carrier is waiting on {0} medical certifications.", owing.length)}{" "}
          {owing.some((row) => row.certificationLate)
            ? t("One is past its deadline — leave can be denied once it is.")
            : t("Send it in before the deadline on the request.")}
        </p>
      ) : null}

      <ul className="border-border mt-3 flex flex-col gap-2 border-t pt-3">
        {cases.map((row) => (
          <li key={row.id} className="text-xs">
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="font-medium">{leaveTypeLabel(row.leaveType)}</span>
              <Badge variant={leaveCaseStatusTone(row.status)}>
                {leaveCaseStatusLabel(row.status)}
              </Badge>
              {row.fmlaDesignated ? <Badge variant="info">{t("Counts as FMLA")}</Badge> : null}
              <Badge variant="secondary">{leaveFrequencyLabel(row.frequency)}</Badge>
            </div>
            <p className="text-muted-foreground mt-0.5">
              {t("{0} {1} · {2} h taken {3}", formatUnixDateMedium(row.startsAt), row.endsAt ? ` – ${formatUnixDateMedium(row.endsAt)}` : t("– ongoing"), formatLeaveHours(row.hoursUsed), row.hoursCharged !== row.hoursUsed
                ? t("({0} h counted)", formatLeaveHours(row.hoursCharged))
                : "")}
            </p>
            {row.certificationStatus === "NotRequired" ? null : (
              <p className="text-muted-foreground mt-0.5">
                {t("Certification {0} {1}", certificationStatusLabel(row.certificationStatus).toLowerCase(), row.certificationDueAt
                  ? t("· due {0}", formatUnixDateMedium(row.certificationDueAt))
                  : "")}
              </p>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
