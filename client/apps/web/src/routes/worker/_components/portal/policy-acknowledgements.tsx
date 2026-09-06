import { usePermission } from "@/hooks/use-permission";
import {
  fetchWorkerPolicyAcknowledgements,
  POLICY_ACKNOWLEDGEMENTS_KEY,
} from "@/lib/graphql/self-service";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatShiftDate } from "@trenova/shared/lib/scheduling";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { FileSignatureIcon } from "lucide-react";

/**
 * Every policy version this worker has signed. Older versions stay listed:
 * a signature on a superseded handbook is still a fact about what they had
 * agreed to at the time.
 */
export function PolicyAcknowledgements({ workerId }: { workerId: string }) {
  const { allowed: canRead } = usePermission(Resource.WorkerPolicy, Operation.Read);

  const acks = useQuery({
    queryKey: [POLICY_ACKNOWLEDGEMENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerPolicyAcknowledgements(workerId, { signal }),
    enabled: canRead && workerId.length > 0,
  });

  if (!canRead) return null;
  if (acks.isLoading) return <Skeleton className="h-24 w-full rounded-lg" />;

  const rows = acks.data ?? [];

  return (
    <div className="border-border rounded-lg border p-4">
      <div className="flex items-center gap-2">
        <FileSignatureIcon className="text-muted-foreground size-4" />
        <p className="text-sm font-semibold">Policies signed</p>
      </div>
      {rows.length === 0 ? (
        <p className="text-muted-foreground mt-2 text-xs">
          Nothing signed yet. Policies are signed from Dash under Profile.
        </p>
      ) : (
        <ul className="mt-2 flex flex-col gap-1 text-xs">
          {rows.map((ack) => (
            <li
              key={ack.id}
              className="flex flex-wrap items-center justify-between gap-2 border-t py-1 first:border-t-0"
            >
              <span className="flex flex-wrap items-center gap-2">
                <span className="font-medium">{ack.policy?.title ?? "Policy"}</span>
                <Badge variant="outline">v{ack.versionLabel}</Badge>
                {ack.policy && ack.policy.versionLabel !== ack.versionLabel ? (
                  <Badge variant="secondary">Superseded by v{ack.policy.versionLabel}</Badge>
                ) : null}
              </span>
              <span className="text-muted-foreground tabular-nums">
                {formatShiftDate(ack.acknowledgedAt)}
                {ack.signatureName ? ` · signed “${ack.signatureName}”` : " · read"}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
