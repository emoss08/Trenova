import { usePermission } from "@/hooks/use-permission";
import {
  deleteWorkerRecognition,
  deleteWorkerSafetyEvent,
  fetchWorkerDisciplinaryActions,
  fetchWorkerDisciplinaryLadder,
  fetchWorkerRecognitions,
  fetchWorkerSafetyEvents,
  fetchWorkerSafetyScorecard,
  reopenWorkerSafetyEvent,
  reviewWorkerSafetyEvent,
  WORKER_DISCIPLINARY_ACTIONS_KEY,
  WORKER_DISCIPLINARY_LADDER_KEY,
  WORKER_RECOGNITIONS_KEY,
  WORKER_SAFETY_EVENTS_KEY,
  WORKER_SAFETY_SCORECARD_KEY,
  type WorkerDisciplinaryActionRow,
  type WorkerRecognitionRow,
  type WorkerSafetyEventRow,
} from "@/lib/graphql/worker-safety";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { DisciplinaryLevel } from "@trenova/shared/types/worker-safety";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { CloseEventDialog } from "./safety/close-event-dialog";
import { DisciplineLadder } from "./safety/discipline-ladder";
import { IssueActionDialog } from "./safety/issue-action-dialog";
import { RecognitionDialog } from "./safety/recognition-dialog";
import { InjuryList } from "./safety/injury-list";
import { RecognitionList } from "./safety/recognition-list";
import { RescindActionDialog } from "./safety/rescind-action-dialog";
import { SafetyEventDialog } from "./safety/safety-event-dialog";
import { SafetyEventRow, type SafetyEventPermissions } from "./safety/safety-event-row";
import { SafetyScorecardCard } from "./safety/safety-scorecard-card";
import { useSafetyInvalidation } from "./safety/use-safety-invalidation";

type DialogState =
  | { kind: "event"; event?: WorkerSafetyEventRow | null }
  | { kind: "close"; event: WorkerSafetyEventRow }
  | { kind: "action"; safetyEventId?: string | null }
  | { kind: "rescind"; action: WorkerDisciplinaryActionRow }
  | { kind: "recognition" };

export default function WorkerSafetyTab({ workerId }: { workerId: string }) {
  const { allowed: canRecord } = usePermission(Resource.WorkerSafetyEvent, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerSafetyEvent, Operation.Update);
  const { allowed: canClose } = usePermission(Resource.WorkerSafetyEvent, Operation.Close);
  const { allowed: canDelete } = usePermission(Resource.WorkerSafetyEvent, Operation.Delete);
  const { allowed: canDiscipline } = usePermission(
    Resource.WorkerDisciplinaryAction,
    Operation.Create,
  );
  const { allowed: canRescind } = usePermission(
    Resource.WorkerDisciplinaryAction,
    Operation.Cancel,
  );
  const { allowed: canRecognise } = usePermission(Resource.WorkerRecognition, Operation.Create);
  const { allowed: canDeleteRecognition } = usePermission(
    Resource.WorkerRecognition,
    Operation.Delete,
  );
  const invalidate = useSafetyInvalidation(workerId);
  const [dialog, setDialog] = useState<DialogState | null>(null);

  const scorecardQuery = useQuery({
    queryKey: [WORKER_SAFETY_SCORECARD_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerSafetyScorecard(workerId, { signal }),
  });
  const eventsQuery = useQuery({
    queryKey: [WORKER_SAFETY_EVENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerSafetyEvents(workerId, { signal }),
  });
  const actionsQuery = useQuery({
    queryKey: [WORKER_DISCIPLINARY_ACTIONS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerDisciplinaryActions(workerId, { signal }),
  });
  const ladderQuery = useQuery({
    queryKey: [WORKER_DISCIPLINARY_LADDER_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerDisciplinaryLadder(workerId, { signal }),
  });
  const recognitionsQuery = useQuery({
    queryKey: [WORKER_RECOGNITIONS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerRecognitions(workerId, { signal }),
  });

  const events = useMemo(() => {
    const rows = eventsQuery.data ?? [];
    return [...rows].sort((a, b) => {
      const aOpen = a.status !== "Closed";
      const bOpen = b.status !== "Closed";
      if (aOpen !== bOpen) return aOpen ? -1 : 1;
      return b.occurredAt - a.occurredAt;
    });
  }, [eventsQuery.data]);

  const review = useMutation({
    mutationFn: (event: WorkerSafetyEventRow) =>
      reviewWorkerSafetyEvent({ id: event.id, version: event.version }),
    onSuccess: () => {
      toast.success("Marked under review");
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not update event", { description: error.message }),
  });
  const reopen = useMutation({
    mutationFn: (event: WorkerSafetyEventRow) =>
      reopenWorkerSafetyEvent({ id: event.id, version: event.version }),
    onSuccess: () => {
      toast.success("Event reopened");
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not reopen event", { description: error.message }),
  });
  const remove = useMutation({
    mutationFn: (event: WorkerSafetyEventRow) => deleteWorkerSafetyEvent(event.id),
    onSuccess: () => {
      toast.success("Event deleted");
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not delete event", { description: error.message }),
  });
  const removeRecognition = useMutation({
    mutationFn: (recognition: WorkerRecognitionRow) => deleteWorkerRecognition(recognition.id),
    onSuccess: () => {
      toast.success("Recognition removed");
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not remove recognition", { description: error.message }),
  });

  const permissions = useMemo<SafetyEventPermissions>(
    () => ({ canUpdate, canClose, canDelete, canDiscipline }),
    [canClose, canDelete, canDiscipline, canUpdate],
  );
  const busy = review.isPending || reopen.isPending || remove.isPending;

  if (scorecardQuery.isLoading || eventsQuery.isLoading || ladderQuery.isLoading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-24 w-full rounded-xl" />
        <Skeleton className="h-32 w-full rounded-xl" />
        <Skeleton className="h-32 w-full rounded-xl" />
      </div>
    );
  }

  const scorecard = scorecardQuery.data;
  const ladder = ladderQuery.data;
  if (!scorecard || !ladder) {
    return (
      <p className="text-muted-foreground text-sm">
        The safety record could not be loaded. Try again in a moment.
      </p>
    );
  }

  const suggestedLevel = (
    dialog?.kind === "action" ? ladder.suggestedLevel : ladder.suggestedLevel
  ) as DisciplinaryLevel;

  return (
    <div className="flex flex-col gap-5">
      <SafetyScorecardCard
        scorecard={scorecard}
        canRecord={canRecord}
        canRecognise={canRecognise}
        onRecordEvent={() => setDialog({ kind: "event", event: null })}
        onRecognise={() => setDialog({ kind: "recognition" })}
      />

      <section className="flex flex-col gap-2">
        <div>
          <h3 className="text-sm font-semibold">Events</h3>
          <p className="text-muted-foreground text-xs">
            Accidents, incidents, near misses, citations and roadside inspections. Open ones come
            first.
          </p>
        </div>
        {events.length === 0 ? (
          <p className="text-muted-foreground border-border rounded-lg border border-dashed px-3 py-6 text-center text-xs">
            Nothing on record
          </p>
        ) : (
          <div className="flex flex-col gap-3">
            {events.map((event) => (
              <SafetyEventRow
                key={event.id}
                event={event}
                permissions={permissions}
                busy={busy}
                onEdit={(row) => setDialog({ kind: "event", event: row })}
                onClose={(row) => setDialog({ kind: "close", event: row })}
                onReview={(row) => review.mutate(row)}
                onReopen={(row) => reopen.mutate(row)}
                onDelete={(row) => remove.mutate(row)}
                onDiscipline={(row) => setDialog({ kind: "action", safetyEventId: row.id })}
              />
            ))}
          </div>
        )}
      </section>

      <DisciplineLadder
        ladder={ladder}
        actions={actionsQuery.data ?? []}
        canIssue={canDiscipline}
        canRescind={canRescind}
        busy={busy}
        onIssue={() => setDialog({ kind: "action", safetyEventId: null })}
        onRescind={(action) => setDialog({ kind: "rescind", action })}
      />

      <RecognitionList
        recognitions={recognitionsQuery.data ?? []}
        canDelete={canDeleteRecognition}
        busy={removeRecognition.isPending}
        onDelete={(recognition) => removeRecognition.mutate(recognition)}
      />

      <InjuryList workerId={workerId} />

      <SafetyEventDialog
        open={dialog?.kind === "event"}
        onOpenChange={(open) => {
          if (!open) setDialog(null);
        }}
        workerId={workerId}
        event={dialog?.kind === "event" ? dialog.event : null}
      />
      <CloseEventDialog
        open={dialog?.kind === "close"}
        onOpenChange={(open) => {
          if (!open) setDialog(null);
        }}
        workerId={workerId}
        event={dialog?.kind === "close" ? dialog.event : null}
      />
      <IssueActionDialog
        open={dialog?.kind === "action"}
        onOpenChange={(open) => {
          if (!open) setDialog(null);
        }}
        workerId={workerId}
        suggestedLevel={suggestedLevel}
        safetyEventId={dialog?.kind === "action" ? dialog.safetyEventId : null}
      />
      <RescindActionDialog
        open={dialog?.kind === "rescind"}
        onOpenChange={(open) => {
          if (!open) setDialog(null);
        }}
        workerId={workerId}
        action={dialog?.kind === "rescind" ? dialog.action : null}
      />
      <RecognitionDialog
        open={dialog?.kind === "recognition"}
        onOpenChange={(open) => {
          if (!open) setDialog(null);
        }}
        workerId={workerId}
      />
    </div>
  );
}
