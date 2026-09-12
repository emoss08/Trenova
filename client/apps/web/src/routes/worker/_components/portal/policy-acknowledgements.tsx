import { useT } from "@trenova/shared/i18n/use-t";
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
  const t = useT();

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
    <div className="rounded-lg border p-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <FileSignatureIcon className="text-muted-foreground size-4" />
          <h3 className="text-sm font-semibold">{t("Policies signed")}</h3>
        </div>
        <span className="text-muted-foreground text-xs tabular-nums">
          {t("{0, plural, one {# signature} other {# signatures}}", rows.length)}
        </span>
      </div>
      {rows.length === 0 ? (
        <p className="text-muted-foreground mt-2 text-xs">
          {t("Nothing signed yet. Policies are signed from Dash under Profile.")}
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
                      superseded ? "bg-muted-foreground/40" : "bg-primary",
                    )}
                  />
                  {index < rows.length - 1 ? <span className="bg-border my-1 w-px flex-1" /> : null}
                </span>
                <div className="flex min-w-0 flex-1 flex-wrap items-center justify-between gap-2 pb-3">
                  <span className="flex min-w-0 flex-wrap items-center gap-1.5">
                    <span className="truncate font-medium">{ack.policy?.title ?? t("Policy")}</span>
                    <Badge variant="outline">{t("v{0}", ack.versionLabel)}</Badge>
                    {superseded ? (
                      <Badge variant="secondary">{t("Superseded by v{0}", ack.policy?.versionLabel)}</Badge>
                    ) : null}
                  </span>
                  <span className="text-muted-foreground flex items-center gap-1 tabular-nums">
                    {formatShiftDate(ack.acknowledgedAt)}
                    {ack.signatureName ? (
                      <>
                        <PenLineIcon className="size-3" />
                        <span>{ack.signatureName}</span>
                      </>
                    ) : (
                      ` ${t("· read")}`
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
