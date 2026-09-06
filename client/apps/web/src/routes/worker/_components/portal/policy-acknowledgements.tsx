import { usePermission } from "@/hooks/use-permission";
import {
  fetchWorkerPolicyAcknowledgements,
  POLICY_ACKNOWLEDGEMENTS_KEY,
} from "@/lib/graphql/self-service";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatShiftDate } from "@trenova/shared/lib/scheduling";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { FileSignatureIcon, PenLineIcon } from "lucide-react";

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
  if (acks.isLoading) return <Skeleton className="h-24 w-full rounded-xl" />;

  const rows = acks.data ?? [];

  return (
    <div className="border-border/80 bg-card rounded-xl border p-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="grid size-7 place-items-center rounded-lg bg-emerald-500/12 text-emerald-600 dark:text-emerald-300">
            <FileSignatureIcon className="size-3.5" />
          </span>
          <p className="text-sm font-semibold">Policies signed</p>
        </div>
        <span className="text-muted-foreground text-xs tabular-nums">
          {rows.length} signature{rows.length === 1 ? "" : "s"}
        </span>
      </div>
      {rows.length === 0 ? (
        <p className="text-muted-foreground mt-2 text-xs">
          Nothing signed yet. Policies are signed from Dash under Profile.
        </p>
      ) : (
        <ol className="mt-3 flex flex-col">
          {rows.map((ack, index) => {
            const superseded = Boolean(ack.policy && ack.policy.versionLabel !== ack.versionLabel);
            return (
              <li key={ack.id} className="flex gap-3 text-xs">
                <span className="flex flex-col items-center" aria-hidden>
                  <span
                    className={cn(
                      "mt-1.5 size-2 shrink-0 rounded-full",
                      superseded ? "bg-muted-foreground/40" : "bg-emerald-500",
                    )}
                  />
                  {index < rows.length - 1 ? <span className="bg-border my-1 w-px flex-1" /> : null}
                </span>
                <div className="flex min-w-0 flex-1 flex-wrap items-center justify-between gap-2 pb-3">
                  <span className="flex min-w-0 flex-wrap items-center gap-1.5">
                    <span className="truncate font-medium">{ack.policy?.title ?? "Policy"}</span>
                    <Badge variant="outline">v{ack.versionLabel}</Badge>
                    {superseded ? (
                      <Badge variant="secondary">Superseded by v{ack.policy?.versionLabel}</Badge>
                    ) : null}
                  </span>
                  <span className="text-muted-foreground flex items-center gap-1 tabular-nums">
                    {formatShiftDate(ack.acknowledgedAt)}
                    {ack.signatureName ? (
                      <>
                        <PenLineIcon className="size-3" />
                        <span className="font-serif italic">{ack.signatureName}</span>
                      </>
                    ) : (
                      " · read"
                    )}
                  </span>
                </div>
              </li>
            );
          })}
        </ol>
      )}
    </div>
  );
}
