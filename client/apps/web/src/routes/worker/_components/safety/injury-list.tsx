import { usePermission } from "@/hooks/use-permission";
import {
  deleteWorkerInjury,
  fetchWorkerInjuries,
  WORKER_INJURIES_KEY,
  type WorkerInjuryRow,
} from "@/lib/graphql/worker-injury";
import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  caseClassificationLabel,
  claimStatusLabel,
  claimStatusTone,
  classificationTone,
  illnessTypeLabel,
} from "@trenova/shared/lib/injury";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PencilIcon, Trash2Icon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { InjuryDialog } from "./injury-dialog";

/**
 * A worker's injury and illness cases, shown beside their safety events because
 * an injury usually has one behind it. The log itself is read for the whole
 * establishment on the OSHA page; this is the per-worker view.
 */
export function InjuryList({ workerId }: { workerId: string }) {
  const queryClient = useQueryClient();
  const { allowed: canRead } = usePermission(Resource.WorkerInjury, Operation.Read);
  const { allowed: canRecord } = usePermission(Resource.WorkerInjury, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerInjury, Operation.Update);
  const { allowed: canDelete } = usePermission(Resource.WorkerInjury, Operation.Delete);
  const [dialog, setDialog] = useState<{ injury: WorkerInjuryRow | null } | null>(null);

  const injuriesQuery = useQuery({
    queryKey: [WORKER_INJURIES_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerInjuries(workerId, { signal }),
    enabled: canRead,
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteWorkerInjury(id),
    onSuccess: () => {
      toast.success("Case deleted", {
        description: "The case number is not reused — two case 4s cannot be told apart.",
      });
      void queryClient.invalidateQueries({ queryKey: [WORKER_INJURIES_KEY, workerId] });
      void queryClient.invalidateQueries({ queryKey: ["osha-log"] });
    },
    onError: (error: Error) =>
      toast.error("Could not delete the case", { description: error.message }),
  });

  // The log carries body parts, treatment and claims. Somebody without the
  // grant sees nothing at all rather than an empty section that implies there
  // is nothing to see.
  if (!canRead) return null;

  const injuries = injuriesQuery.data ?? [];

  return (
    <section className="flex flex-col gap-2">
      <div className="flex items-center justify-between gap-3">
        <h4 className="text-muted-foreground text-[11px] font-semibold uppercase">
          Injuries &amp; illnesses
        </h4>
        {canRecord ? (
          <Button size="sm" variant="outline" onClick={() => setDialog({ injury: null })}>
            Record a case
          </Button>
        ) : null}
      </div>

      {injuriesQuery.isLoading ? (
        <Skeleton className="h-16 w-full rounded-lg" />
      ) : injuries.length === 0 ? (
        <p className="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-center text-xs">
          No injury or illness has been recorded for this worker.
        </p>
      ) : (
        <ul className="divide-border divide-y rounded-lg border">
          {injuries.map((injury) => {
            const actions: RowAction[] = [];
            if (canUpdate) {
              actions.push({
                id: "edit",
                label: `Edit case ${injury.caseYear}-${injury.caseNumber}`,
                icon: PencilIcon,
                onSelect: () => setDialog({ injury }),
              });
            }
            if (canDelete) {
              actions.push({
                id: "delete",
                label: `Delete case ${injury.caseYear}-${injury.caseNumber}`,
                icon: Trash2Icon,
                destructive: true,
                disabled: deleteMutation.isPending,
                onSelect: () => deleteMutation.mutate(injury.id),
              });
            }
            return (
              <li key={injury.id} className="px-3 py-2.5 text-xs">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="flex flex-wrap items-center gap-2">
                    <span className="text-muted-foreground tabular-nums">
                      {injury.caseYear}-{injury.caseNumber}
                    </span>
                    <Badge variant={classificationTone(injury.classification)}>
                      {caseClassificationLabel(injury.classification)}
                    </Badge>
                    {injury.recordable ? <Badge variant="info">On the 300 log</Badge> : null}
                    {injury.status === "Open" ? <Badge variant="warning">Open</Badge> : null}
                    {injury.privacyCase ? <Badge variant="secondary">Privacy case</Badge> : null}
                  </span>
                  <span className="flex items-center gap-2">
                    <span className="text-muted-foreground tabular-nums">
                      {formatUnixDate(injury.occurredAt)}
                    </span>
                    <RowActionsMenu
                      label={`Actions for case ${injury.caseYear}-${injury.caseNumber}`}
                      actions={actions}
                    />
                  </span>
                </div>
                <p className="text-muted-foreground mt-1">{injury.description}</p>
                <p className="text-muted-foreground mt-1">
                  {illnessTypeLabel(injury.illnessType)}
                  {injury.bodyPart ? ` · ${injury.bodyPart}` : ""}
                  {injury.daysAway > 0 ? ` · ${injury.daysAway} days away` : ""}
                  {injury.daysRestricted > 0 ? ` · ${injury.daysRestricted} restricted` : ""}
                </p>
                {injury.claimStatus !== "NotFiled" ? (
                  <p className="mt-1 flex items-center gap-2">
                    <Badge variant={claimStatusTone(injury.claimStatus)}>
                      Claim {claimStatusLabel(injury.claimStatus).toLowerCase()}
                    </Badge>
                    {injury.claimNumber ? (
                      <span className="text-muted-foreground">{injury.claimNumber}</span>
                    ) : null}
                  </p>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}

      <InjuryDialog
        open={dialog !== null}
        onOpenChange={(open) => !open && setDialog(null)}
        workerId={workerId}
        injury={dialog?.injury ?? null}
      />
    </section>
  );
}
