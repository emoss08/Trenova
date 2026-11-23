import { HoverGlideGroup } from "@/components/hover-glide";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { useWorkPane } from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { fetchWorkerInjuries, WORKER_INJURIES_KEY } from "@/lib/graphql/worker-injury";
import { previewPendingRefetchMs } from "@/lib/resource-documents";
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
import type { WorkerTabProps, WorkerTarget } from "@/routes/worker/use-worker-state";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { DisciplinaryLevel } from "@trenova/shared/types/worker-safety";
import { useState } from "react";
import { CloseEventDialog } from "./close-event-dialog";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { DisciplinaryActionPane } from "./disciplinary-action-pane";
import { DisciplineStair } from "./discipline-stair";
import { InjuryList } from "./injury-list";
import { InjuryPane } from "./injury-pane";
import { IssueActionPane } from "./issue-action-pane";
import { RecognitionList } from "./recognition-list";
import { RecognitionPane } from "./recognition-pane";
import { RecordSafetyEventPane } from "./record-safety-event-pane";
import { RescindActionDialog } from "./rescind-action-dialog";
import { SafetyEventPane } from "./safety-event-pane";
import {
  SafetyEventRow,
  safetyEventHeadline,
  type SafetyEventPermissions,
} from "./safety-event-row";
import { filterEvents, sortEvents, type EventFilter } from "./safety-model";
import { SafetyScorecardCard } from "./safety-scorecard-card";
import { SafetySkeleton } from "./safety-skeleton";
import { useSafetyInvalidation } from "./use-safety-invalidation";

type Dialog =
  | { kind: "close"; event: WorkerSafetyEventRow }
  | { kind: "delete"; event: WorkerSafetyEventRow }
  | { kind: "rescind"; action: WorkerDisciplinaryActionRow }
  | { kind: "remove-recognition"; recognition: WorkerRecognitionRow };

/**
 * The worker's safety record: the scorecard band with the points gauge, the events, the
 * discipline stair, recognition and injury cases. Forms and a record's details open in the work
 * pane from the page's address (`form`, `safetyEvent`, `action`, `injury`); decisions and every
 * deletion stay dialogs.
 */
export default function WorkerSafetyTab({ workerId, worker, open, onOpen }: WorkerTabProps) {
  const t = useT();
  const { closeThen } = useWorkPane();
  const invalidate = useSafetyInvalidation(workerId);
  const workerName = worker.firstName;

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
  const { allowed: canReadInjury } = usePermission(Resource.WorkerInjury, Operation.Read);
  const { allowed: canRecordInjury } = usePermission(Resource.WorkerInjury, Operation.Create);
  const { allowed: canUpdateInjury } = usePermission(Resource.WorkerInjury, Operation.Update);
  const { allowed: canDeleteInjury } = usePermission(Resource.WorkerInjury, Operation.Delete);

  const [filter, setFilter] = useState<EventFilter>("all");
  const [dialog, setDialog] = useState<Dialog | null>(null);
  const [preview, setPreview] = useState<DisciplinaryLevel | null>(null);

  const scorecardQuery = useQuery({
    queryKey: [WORKER_SAFETY_SCORECARD_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerSafetyScorecard(workerId, { signal }),
  });
  const eventsQuery = useQuery({
    queryKey: [WORKER_SAFETY_EVENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerSafetyEvents(workerId, { signal }),
    refetchInterval: (query) =>
      previewPendingRefetchMs((query.state.data ?? []).map((event) => event.document)),
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
  const injuriesQuery = useQuery({
    queryKey: [WORKER_INJURIES_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerInjuries(workerId, { signal }),
    enabled: canReadInjury,
  });

  const review = useApiMutation({
    scope: { id: `worker-safety-status:${workerId}` },
    resourceName: "Safety event",
    mutationFn: (event: WorkerSafetyEventRow) =>
      reviewWorkerSafetyEvent({ id: event.id, version: event.version }),
    onSuccess: () => void invalidate(),
  });
  const reopen = useApiMutation({
    scope: { id: `worker-safety-status:${workerId}` },
    resourceName: "Safety event",
    mutationFn: (event: WorkerSafetyEventRow) =>
      reopenWorkerSafetyEvent({ id: event.id, version: event.version }),
    onSuccess: () => void invalidate(),
  });
  const removeEvent = useApiMutation({
    scope: { id: `worker-safety-status:${workerId}` },
    resourceName: "Safety event",
    mutationFn: (event: WorkerSafetyEventRow) => deleteWorkerSafetyEvent(event.id),
    onSuccess: (_, event) => {
      if (open.safetyEvent === event.id) onOpen({ tab: "safety" });
      void invalidate();
    },
  });
  const removeRecognition = useApiMutation({
    scope: { id: `worker-recognition-remove:${workerId}` },
    resourceName: "Recognition",
    mutationFn: (recognition: WorkerRecognitionRow) => deleteWorkerRecognition(recognition.id),
    onSuccess: () => void invalidate(),
  });

  if (scorecardQuery.isPending || eventsQuery.isPending || ladderQuery.isPending) {
    return <SafetySkeleton />;
  }

  const failed = [scorecardQuery, eventsQuery, ladderQuery].find((query) => query.isError);
  if (failed || !scorecardQuery.data || !eventsQuery.data || !ladderQuery.data) {
    return (
      <ErrorState
        error={failed?.error}
        layout="sheet"
        title={t("The safety record could not be loaded")}
        onRetry={() => {
          void scorecardQuery.refetch();
          void eventsQuery.refetch();
          void ladderQuery.refetch();
        }}
      />
    );
  }

  const scorecard = scorecardQuery.data;
  const ladder = ladderQuery.data;
  const events = sortEvents(eventsQuery.data);
  const actions = actionsQuery.data ?? [];
  const injuries = injuriesQuery.data ?? [];
  const openCount = events.filter((event) => event.status !== "Closed").length;
  const shown = filterEvents(events, filter);
  const permissions: SafetyEventPermissions = { canUpdate, canClose, canDelete, canDiscipline };
  const busy = review.isPending || reopen.isPending || removeEvent.isPending;

  const eventById = (id: string | null) =>
    id === null ? null : (events.find((event) => event.id === id) ?? null);
  const formEvent = eventById(open.safetyEvent);
  const readAction = open.action
    ? (actions.find((action) => action.id === open.action) ?? null)
    : null;
  const editingInjury = open.injury
    ? (injuries.find((injury) => injury.id === open.injury) ?? null)
    : null;

  const go = (target: Omit<Extract<WorkerTarget, { tab: "safety" }>, "tab">) =>
    closeThen(() => onOpen({ tab: "safety", ...target }));
  const close = () => onOpen({ tab: "safety" });

  const handlers = {
    onOpen: (event: WorkerSafetyEventRow) => go({ safetyEvent: event.id }),
    onEdit: (event: WorkerSafetyEventRow) =>
      go({ form: "record-safety-event", safetyEvent: event.id }),
    onClose: (event: WorkerSafetyEventRow) => setDialog({ kind: "close", event }),
    onReview: (event: WorkerSafetyEventRow) =>
      void review.mutateAsync(event).catch(() => undefined),
    onReopen: (event: WorkerSafetyEventRow) =>
      void reopen.mutateAsync(event).catch(() => undefined),
    onDelete: (event: WorkerSafetyEventRow) => setDialog({ kind: "delete", event }),
    onDiscipline: (event: WorkerSafetyEventRow) =>
      go({ form: "issue-action", safetyEvent: event.id }),
  };

  return (
    <div className="flex min-w-0 flex-col gap-4 text-sm">
      <SafetyScorecardCard
        scorecard={scorecard}
        events={events}
        canRecord={canRecord}
        canRecognise={canRecognise}
        onRecordEvent={() => go({ form: "record-safety-event" })}
        onRecognise={() => go({ form: "add-recognition" })}
      />

      <div className="@container">
        <div className="grid items-start gap-4 @5xl:grid-cols-[minmax(0,1fr)_24rem]">
          <SectionPanel
            title={t("Events")}
            count={events.length}
            action={
              events.length > 0 ? (
                <SegmentedControl<EventFilter>
                  aria-label={t("Show events")}
                  value={filter}
                  onValueChange={setFilter}
                  items={[
                    {
                      value: "open",
                      label: (
                        <>
                          {t("Open")}{" "}
                          <span className="font-normal text-foreground-subtle tabular-nums">
                            {formatNumber(openCount)}
                          </span>
                        </>
                      ),
                    },
                    {
                      value: "all",
                      label: (
                        <>
                          {t("All")}{" "}
                          <span className="font-normal text-foreground-subtle tabular-nums">
                            {formatNumber(events.length)}
                          </span>
                        </>
                      ),
                    },
                  ]}
                />
              ) : null
            }
          >
            {events.length === 0 ? (
              <SectionPanelQuiet>
                {t("No accidents, incidents, citations or inspections on record.")}
              </SectionPanelQuiet>
            ) : shown.length === 0 ? (
              <SectionPanelQuiet>
                {t("No open events. Everything on the record is closed.")}
              </SectionPanelQuiet>
            ) : (
              <ScrollArea viewportClassName="max-h-[34rem]" maskVariant="card">
                <HoverGlideGroup className="flex flex-col gap-px px-1.5 py-1">
                  {shown.map((event) => (
                    <div key={event.id} className="motion-safe:animate-expand-in">
                      <SafetyEventRow
                        event={event}
                        permissions={permissions}
                        busy={busy}
                        handlers={handlers}
                      />
                    </div>
                  ))}
                </HoverGlideGroup>
              </ScrollArea>
            )}
          </SectionPanel>

          <div className="flex min-w-0 flex-col gap-4">
            <DisciplineStair
              ladder={ladder}
              actions={actions}
              preview={open.form === "issue-action" ? preview : null}
              canIssue={canDiscipline}
              canRescind={canRescind}
              onIssue={() => go({ form: "issue-action" })}
              onOpenAction={(action) => go({ action: action.id })}
              onOpenEvent={(safetyEventId) => go({ safetyEvent: safetyEventId })}
              onRescind={(action) => setDialog({ kind: "rescind", action })}
            />
            <RecognitionList
              recognitions={recognitionsQuery.data ?? []}
              workerName={workerName}
              canAdd={canRecognise}
              canDelete={canDeleteRecognition}
              removingId={
                removeRecognition.isPending ? (removeRecognition.variables?.id ?? null) : null
              }
              onAdd={() => go({ form: "add-recognition" })}
              onDelete={(recognition) => setDialog({ kind: "remove-recognition", recognition })}
            />
            {canReadInjury ? (
              <InjuryList
                workerId={workerId}
                injuries={injuries}
                loading={injuriesQuery.isPending}
                error={injuriesQuery.error}
                onRetry={() => void injuriesQuery.refetch()}
                canRecord={canRecordInjury}
                canUpdate={canUpdateInjury}
                canDelete={canDeleteInjury}
                onRecord={() => go({ form: "record-injury" })}
                onEdit={(injury) => go({ injury: injury.id })}
              />
            ) : null}
          </div>
        </div>
      </div>

      <SafetyEventPane
        open={open.safetyEvent !== null && open.form === null}
        onOpenChange={(next) => !next && close()}
        event={formEvent}
        permissions={permissions}
        busy={busy}
        onEdit={handlers.onEdit}
        onClose={handlers.onClose}
        onReopen={(event) => reopen.mutateAsync(event)}
        onDelete={handlers.onDelete}
        onDiscipline={handlers.onDiscipline}
        onOpenDocuments={() => closeThen(() => onOpen({ tab: "files", view: "documents" }))}
        onViolationsChanged={invalidate}
      />
      <RecordSafetyEventPane
        open={
          open.form === "record-safety-event" &&
          (open.safetyEvent === null ? canRecord : canUpdate && formEvent !== null)
        }
        onOpenChange={(next) => !next && close()}
        workerId={workerId}
        event={formEvent}
        onSaved={close}
      />
      <IssueActionPane
        open={open.form === "issue-action" && canDiscipline}
        onOpenChange={(next) => !next && close()}
        workerId={workerId}
        workerName={workerName}
        suggestedLevel={ladder.suggestedLevel as DisciplinaryLevel}
        highestLevel={(ladder.highestLevel ?? null) as DisciplinaryLevel | null}
        event={formEvent}
        onLevelChange={setPreview}
        onSaved={close}
      />
      <RecognitionPane
        open={open.form === "add-recognition" && canRecognise}
        onOpenChange={(next) => !next && close()}
        workerId={workerId}
        workerName={workerName}
        onSaved={close}
      />
      <InjuryPane
        open={
          canReadInjury &&
          ((open.form === "record-injury" && canRecordInjury) ||
            (open.form === null && editingInjury !== null && canUpdateInjury))
        }
        onOpenChange={(next) => !next && close()}
        workerId={workerId}
        injury={open.form === "record-injury" ? null : editingInjury}
        onSaved={close}
      />
      <DisciplinaryActionPane
        open={open.action !== null && open.form === null}
        onOpenChange={(next) => !next && close()}
        action={readAction}
        canRescind={canRescind}
        onOpenEvent={(safetyEventId) => go({ safetyEvent: safetyEventId })}
        onRescind={(action) => setDialog({ kind: "rescind", action })}
      />

      <CloseEventDialog
        open={dialog?.kind === "close"}
        onOpenChange={(next) => !next && setDialog(null)}
        workerId={workerId}
        event={dialog?.kind === "close" ? dialog.event : null}
      />
      <RescindActionDialog
        open={dialog?.kind === "rescind"}
        onOpenChange={(next) => !next && setDialog(null)}
        workerId={workerId}
        action={dialog?.kind === "rescind" ? dialog.action : null}
      />
      <ConfirmDialog
        open={dialog?.kind === "delete"}
        onOpenChange={(next) => !next && setDialog(null)}
        title={t("Delete this safety event?")}
        description={
          dialog?.kind === "delete"
            ? dialog.event.activePoints > 0
              ? t(
                  "Its {0, plural, one {# point comes} other {# points come}} off the score and the {1} on {2} comes off the record. This cannot be undone.",
                  dialog.event.activePoints,
                  safetyEventHeadline(dialog.event).toLowerCase(),
                  formatUnixDateMedium(dialog.event.occurredAt),
                )
              : t(
                  "The {0} on {1} comes off the record. This cannot be undone.",
                  safetyEventHeadline(dialog.event).toLowerCase(),
                  formatUnixDateMedium(dialog.event.occurredAt),
                )
            : ""
        }
        confirmLabel={t("Delete event")}
        pending={removeEvent.isPending}
        onConfirm={() =>
          dialog?.kind === "delete" ? removeEvent.mutateAsync(dialog.event) : Promise.resolve()
        }
      />
      <ConfirmDialog
        open={dialog?.kind === "remove-recognition"}
        onOpenChange={(next) => !next && setDialog(null)}
        title={t("Remove this recognition?")}
        description={
          dialog?.kind === "remove-recognition"
            ? t(
                "{0} comes off the safety record and out of Dash. This cannot be undone.",
                dialog.recognition.title,
              )
            : ""
        }
        confirmLabel={t("Remove")}
        pending={removeRecognition.isPending}
        onConfirm={() =>
          dialog?.kind === "remove-recognition"
            ? removeRecognition.mutateAsync(dialog.recognition)
            : Promise.resolve()
        }
      />
    </div>
  );
}
