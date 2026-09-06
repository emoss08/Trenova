import {
  fetchWorkerPolicyCompliance,
  POLICY_COMPLIANCE_KEY,
  type WorkerPolicyRow,
} from "@/lib/graphql/self-service";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Progress } from "@trenova/shared/components/ui/progress";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatShiftDate } from "@trenova/shared/lib/scheduling";
import { useState } from "react";

export type PolicyComplianceDialogProps = {
  policy: WorkerPolicyRow | null;
  onOpenChange: (open: boolean) => void;
};

/**
 * Who has signed the version in force and who has not. Derived on every read:
 * the roster and the signatures both move, and a stored answer would be wrong
 * by the next hire.
 */
export function PolicyComplianceDialog({ policy, onOpenChange }: PolicyComplianceDialogProps) {
  const [showSigned, setShowSigned] = useState(false);

  const compliance = useQuery({
    queryKey: [POLICY_COMPLIANCE_KEY, policy?.id],
    queryFn: ({ signal }) => fetchWorkerPolicyCompliance(policy?.id ?? "", { signal }),
    enabled: Boolean(policy),
  });

  const view = compliance.data;
  const total = view ? view.signed + view.outstanding : 0;
  const percent = total === 0 ? 0 : Math.round((view!.signed / total) * 100);
  const rows = (view?.rows ?? []).filter((row) => showSigned || !row.acknowledgedAt);

  return (
    <Dialog open={policy !== null} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{policy?.title ?? "Policy"}</DialogTitle>
          <DialogDescription>
            Version {view?.policy.versionLabel ?? policy?.versionLabel}. A signature on an earlier
            version does not count — those people are outstanding again.
          </DialogDescription>
        </DialogHeader>

        {compliance.isLoading || !view ? (
          <Skeleton className="h-40 w-full" />
        ) : (
          <div className="flex flex-col gap-3">
            <div className="rounded-lg border p-3">
              <div className="flex items-center justify-between text-xs">
                <span className="font-medium tabular-nums">
                  {view.signed} of {total} signed
                </span>
                <span className="text-muted-foreground tabular-nums">{percent}%</span>
              </div>
              <Progress value={percent} className="mt-2" />
            </div>

            <div className="flex items-center justify-between text-xs">
              <p className="text-muted-foreground">
                {view.outstanding === 0
                  ? "Everybody it applies to has signed."
                  : `${view.outstanding} still to sign.`}
              </p>
              <button
                type="button"
                className="text-muted-foreground underline-offset-2 hover:underline"
                onClick={() => setShowSigned((value) => !value)}
              >
                {showSigned ? "Hide signed" : "Show signed"}
              </button>
            </div>

            <ul className="flex max-h-72 flex-col gap-1 overflow-y-auto">
              {rows.length === 0 ? (
                <li className="text-muted-foreground rounded-md border border-dashed p-3 text-xs">
                  Nobody outstanding.
                </li>
              ) : (
                rows.map((row) => (
                  <li
                    key={row.workerId}
                    className="flex flex-wrap items-center justify-between gap-2 border-t py-1 text-xs first:border-t-0"
                  >
                    <span className="flex items-center gap-2">
                      <span className="font-medium">{row.workerName}</span>
                      <span className="text-muted-foreground">{row.workerType}</span>
                    </span>
                    {row.acknowledgedAt ? (
                      <span className="flex items-center gap-2">
                        <span className="text-muted-foreground tabular-nums">
                          {formatShiftDate(row.acknowledgedAt)}
                          {row.signatureName ? ` · signed “${row.signatureName}”` : " · read"}
                        </span>
                        <Badge variant="active">Signed</Badge>
                      </span>
                    ) : (
                      <Badge variant="warning">Outstanding</Badge>
                    )}
                  </li>
                ))
              )}
            </ul>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
