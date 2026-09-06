import { usePermission } from "@/hooks/use-permission";
import { fetchWorkerLeaveFile, WORKER_LEAVE_KEY } from "@/lib/graphql/worker-leave";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Progress } from "@trenova/shared/components/ui/progress";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  certificationOutstanding,
  entitlementUsedPercent,
  formatLeaveHours,
  measurementMethodLabel,
} from "@trenova/shared/lib/leave";
import { Operation, Resource } from "@trenova/shared/types/permission";

/**
 * The worker's leave standing, shown beside the employment timeline. A leave of
 * absence is an employment event, and the balance is the question anybody
 * reading that event asks next.
 */
export function LeaveStandingStrip({ workerId }: { workerId: string }) {
  const { allowed: canRead } = usePermission(Resource.WorkerLeave, Operation.Read);

  const leaveQuery = useQuery({
    queryKey: [WORKER_LEAVE_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerLeaveFile(workerId, { signal }),
    enabled: canRead,
  });

  if (!canRead) return null;
  if (leaveQuery.isLoading) return <Skeleton className="h-16 w-full" />;

  const file = leaveQuery.data;
  // Nothing to say about a worker who has never taken leave, and an empty
  // entitlement card beside the timeline reads as leave they are owed.
  if (!file || file.cases.length === 0) return null;

  const { entitlement } = file;
  const owing = file.cases.filter((row) =>
    certificationOutstanding(row.certificationStatus),
  ).length;

  return (
    <section className="rounded-lg border px-3 py-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="flex flex-wrap items-center gap-2">
          <h4 className="text-xs font-medium">Leave standing</h4>
          {entitlement.exhausted ? <Badge variant="inactive">Exhausted</Badge> : null}
          {entitlement.eligibleOnTenure ? null : (
            <Badge variant="warning">Under 12 months&apos; service</Badge>
          )}
          {owing > 0 ? (
            <Badge variant="warning">
              {owing} certification{owing === 1 ? "" : "s"} owed
            </Badge>
          ) : null}
        </span>
        <span className="text-xs tabular-nums">
          <span className="font-semibold">{formatLeaveHours(entitlement.remainingHours)} h</span>
          <span className="text-muted-foreground">
            {" "}
            of {formatLeaveHours(entitlement.totalHours)} left
          </span>
        </span>
      </div>
      <Progress
        value={entitlementUsedPercent(entitlement.usedHours, entitlement.totalHours)}
        className="mt-2"
      />
      <p className="text-muted-foreground mt-1.5 text-[11px]">
        {measurementMethodLabel(entitlement.method)} · {formatUnixDate(entitlement.window.from)} to{" "}
        {formatUnixDate(entitlement.window.through)} · {file.cases.length} case
        {file.cases.length === 1 ? "" : "s"}
      </p>
    </section>
  );
}
