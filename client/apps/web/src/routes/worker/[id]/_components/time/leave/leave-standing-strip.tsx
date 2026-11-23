import { usePermission } from "@/hooks/use-permission";
import { formatRange } from "@trenova/shared/lib/date";
import { fetchWorkerLeaveFile, WORKER_LEAVE_KEY } from "@/lib/graphql/worker-leave";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  certificationOutstanding,
  formatLeaveHours,
  measurementMethodLabel,
} from "@trenova/shared/lib/leave";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { EntitlementBar } from "./entitlement-bar";

/**
 * The worker's leave standing, shown beside the employment timeline. A leave of absence is an
 * employment event, and the balance is the question anybody reading that event asks next.
 */
export function LeaveStandingStrip({ workerId }: { workerId: string }) {
  const t = useT();
  const { allowed: canRead } = usePermission(Resource.WorkerLeave, Operation.Read);
  const leaveQuery = useQuery({
    queryKey: [WORKER_LEAVE_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerLeaveFile(workerId, { signal }),
    enabled: canRead,
  });

  if (!canRead) return null;
  if (leaveQuery.isPending) {
    return (
      <div
        aria-busy
        aria-label={t("Loading the leave standing")}
        className="h-16 w-full ui-shimmer rounded-lg border"
      />
    );
  }

  if (leaveQuery.isError) {
    return (
      <ErrorState
        layout="compact"
        error={leaveQuery.error}
        onRetry={() => void leaveQuery.refetch()}
        title={t("The leave standing could not be loaded")}
      />
    );
  }

  const file = leaveQuery.data;
  if (file.cases.length === 0) return null;

  const { entitlement } = file;
  const owing = file.cases.filter((row) =>
    certificationOutstanding(row.certificationStatus),
  ).length;

  return (
    <section
      aria-label={t("Leave standing")}
      className="flex flex-col gap-2 rounded-lg border bg-card px-3 py-2.5"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="flex flex-wrap items-center gap-2">
          <h4 className="text-xs font-medium">{t("Leave standing")}</h4>
          {entitlement.exhausted ? (
            <Badge variant="danger" dot>
              {t("Exhausted")}
            </Badge>
          ) : null}
          {entitlement.eligibleOnTenure ? null : (
            <Badge variant="warning" dot>
              {t("Under 12 months' service")}
            </Badge>
          )}
          {owing > 0 ? (
            <Badge variant="warning" dot>
              {t("{0, plural, one {# certification} other {# certifications}} owed", owing)}
            </Badge>
          ) : null}
        </span>
        <span className="text-xs tabular-nums">
          <span className="font-semibold">
            {t("{0} h", formatLeaveHours(entitlement.remainingHours))}
          </span>{" "}
          <span className="text-muted-foreground">
            {t("of {0} left", formatLeaveHours(entitlement.totalHours))}
          </span>
        </span>
      </div>
      <EntitlementBar entitlement={entitlement} scale={false} />
      <p className="text-xs text-muted-foreground">
        {t(
          "{0} · {1} · {2, plural, one {# case} other {# cases}}",
          measurementMethodLabel(entitlement.method),
          formatRange(entitlement.window.from, entitlement.window.through, {
            year: "always",
            join: "words",
          }),
          file.cases.length,
        )}
      </p>
    </section>
  );
}
