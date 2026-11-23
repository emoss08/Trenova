import { ConfirmDialog } from "@/components/confirm-dialog";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { VirtualRows, type VirtualRow } from "@/components/virtual-rows";
import { lazyWorkPane, preloadOn, useWorkPane } from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { fetchWorkerInjury, WORKER_INJURIES_KEY } from "@/lib/graphql/worker-injury";
import {
  deleteWorkerRecognition,
  deleteWorkerSafetyEvent,
  fetchWorkerDisciplinaryAction,
  fetchWorkerSafetyEvent,
  reopenWorkerSafetyEvent,
  reviewWorkerSafetyEvent,
  WORKER_DISCIPLINARY_ACTIONS_KEY,
  WORKER_SAFETY_EVENTS_KEY,
  type WorkerDisciplinaryActionRow,
  type WorkerRecognitionRow,
  type WorkerSafetyEventRow,
} from "@/lib/graphql/worker-safety";
import type { WorkerTabProps, WorkerTarget } from "@/routes/worker/use-worker-state";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  DISCIPLINARY_LEVEL_LABELS,
  type DisciplinaryLevel,
} from "@trenova/shared/types/worker-safety";
import { useState } from "react";
import { ListCapLine, SERVER_LIST_CAP } from "../list-more";
import { useRecordById } from "../use-record-by-id";
import { useWorkerInvalidation } from "../use-worker-invalidation";
import { WORKER_FORM_PANES } from "../worker-form-panes";
import { CloseEventDialog } from "./close-event-dialog";
import { DisciplineStair } from "./discipline-stair";
import { InjuryList } from "./injury-list";
import { RecognitionList } from "./recognition-list";
import { RescindActionDialog } from "./rescind-action-dialog";
import { SafetyEventRow, type SafetyEventPermissions } from "./safety-event-row";
import { filterEvents, safetyEventHeadline, sortEvents, type EventFilter } from "./safety-model";
import {
  disciplinaryLadderQuery,
  openSafetyEventsQuery,
  recognitionsQuery,
  safetyEventsQuery,
  safetyScorecardQuery,
  useDisciplinaryActionList,
  useInjuryList,
} from "./safety-reads";
import { SafetyScorecardCard } from "./safety-scorecard-card";
import { SafetySkeleton } from "./safety-skeleton";

/*
 * Every pane loads the first time it opens (or earlier, on intent): the tab's chunk carries the
 * record as it shows on arrival and nothing a person has not opened.
 */
const SafetyEventPane = lazyWorkPane(
  () => import("./safety-event-pane"),
  (module) => module.SafetyEventPane,
  {
    title: (props, t) => (props.event ? safetyEventHeadline(props.event) : t("Safety event")),
    openKey: (props) => props.event?.id,
  },
);
const RecordSafetyEventPane = lazyWorkPane(
  WORKER_FORM_PANES["record-safety-event"],
  (module) => module.RecordSafetyEventPane,
  {
    title: (props, t) => (props.event ? t("Edit safety event") : t("Record a safety event")),
    openKey: (props) => props.event?.id ?? "new",
    fields: 8,
  },
);
const IssueActionPane = lazyWorkPane(
  WORKER_FORM_PANES["issue-action"],
  (module) => module.IssueActionPane,
  { title: (_, t) => t("Issue a disciplinary action") },
);
const RecognitionPane = lazyWorkPane(
  WORKER_FORM_PANES["add-recognition"],
  (module) => module.RecognitionPane,
  { title: (_, t) => t("Add recognition") },
);
const InjuryPane = lazyWorkPane(WORKER_FORM_PANES["record-injury"], (module) => module.InjuryPane, {
  title: (props, t) => (props.injury ? t("Edit case") : t("Record an injury or illness")),
  openKey: (props) => props.injury?.id ?? "new",
  fields: 8,
});
const DisciplinaryActionPane = lazyWorkPane(
  () => import("./disciplinary-action-pane"),
  (module) => module.DisciplinaryActionPane,
  {
    title: (props, t) =>
      props.action
        ? t(
            DISCIPLINARY_LEVEL_LABELS[props.action.level as DisciplinaryLevel] ??
              props.action.level,
          )
        : t("Disciplinary action"),
    openKey: (props) => props.action?.id,
  },
);

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
  const invalidate = useWorkerInvalidation(workerId, "safety");
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
  // The level the issue pane has picked, previewed on the stair; the suggested one until a pick.
  const [picked, setPicked] = useState<DisciplinaryLevel | null>(null);

  const scorecardQuery = useQuery(safetyScorecardQuery(workerId));
  const eventsQuery = useQuery(safetyEventsQuery(workerId));
  const ladderQuery = useQuery(disciplinaryLadderQuery(workerId));
  const actions = useDisciplinaryActionList(workerId);
  const recognitionsRead = useQuery(recognitionsQuery(workerId));
  const injuries = useInjuryList(workerId, canReadInjury);
  // The server returns the newest `SERVER_LIST_CAP`: a full list may be cut short, so the Open
  // filter then asks the server for the open events rather than filtering what came back.
  const capped = (eventsQuery.data?.length ?? 0) >= SERVER_LIST_CAP;
  const openQuery = useQuery({
    ...openSafetyEventsQuery(workerId),
    enabled: filter === "open" && capped,
  });
  // The records the address names, from the rows read or, past them, by id.
  const eventRecord = useRecordById({
    root: WORKER_SAFETY_EVENTS_KEY,
    workerId,
    id: open.safetyEvent,
    rows: [...(eventsQuery.data ?? []), ...(openQuery.data ?? [])],
    rowsSettled: !eventsQuery.isPending,
    fetchRecord: fetchWorkerSafetyEvent,
  });
  const actionRecord = useRecordById({
    root: WORKER_DISCIPLINARY_ACTIONS_KEY,
    workerId,
    id: open.action,
    rows: actions.rows,
    rowsSettled: !actions.isPending,
    fetchRecord: fetchWorkerDisciplinaryAction,
  });
  const injuryRecord = useRecordById({
    root: WORKER_INJURIES_KEY,
    workerId,
    id: open.injury,
    rows: injuries.rows,
    rowsSettled: !injuries.isPending,
    fetchRecord: fetchWorkerInjury,
    enabled: canReadInjury,
  });

  const review = useApiMutation({
    scope: { id: `worker-safety-status:${workerId}` },
    resourceName: "Safety event",
    mutationFn: (event: WorkerSafetyEventRow) =>
      reviewWorkerSafetyEvent({ id: event.id, version: event.version }),
    onSuccess: () => invalidate(),
  });
  const reopen = useApiMutation({
    scope: { id: `worker-safety-status:${workerId}` },
    resourceName: "Safety event",
    mutationFn: (event: WorkerSafetyEventRow) =>
      reopenWorkerSafetyEvent({ id: event.id, version: event.version }),
    onSuccess: () => invalidate(),
  });
  const removeEvent = useApiMutation({
    scope: { id: `worker-safety-status:${workerId}` },
    resourceName: "Safety event",
    mutationFn: (event: WorkerSafetyEventRow) => deleteWorkerSafetyEvent(event.id),
    onSuccess: (_, event) => {
      if (open.safetyEvent === event.id) onOpen({ tab: "safety" });
      return invalidate();
    },
  });
  const removeRecognition = useApiMutation({
    scope: { id: `worker-recognition-remove:${workerId}` },
    resourceName: "Recognition",
    mutationFn: (recognition: WorkerRecognitionRow) => deleteWorkerRecognition(recognition.id),
    onSuccess: () => invalidate(),
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
  const fromServer = filter === "open" && capped;
  // Until the server's open events arrive, the open ones already read stand in for them.
  const shown =
    fromServer && openQuery.data ? sortEvents(openQuery.data) : filterEvents(events, filter);
  const permissions: SafetyEventPermissions = { canUpdate, canClose, canDelete, canDiscipline };
  const busy = review.isPending || reopen.isPending || removeEvent.isPending;
  const suggestedLevel = ladder.suggestedLevel as DisciplinaryLevel;

  const formEvent = eventRecord.record;
  const readAction = actionRecord.record;
  const editingInjury = injuryRecord.record;

  const go = (target: Omit<Extract<WorkerTarget, { tab: "safety" }>, "tab">) =>
    closeThen(() => onOpen({ tab: "safety", ...target }));
  const close = () => {
    setPicked(null);
    onOpen({ tab: "safety" });
  };
  const issue = (target: Omit<Extract<WorkerTarget, { tab: "safety" }>, "tab">) => {
    setPicked(null);
    go({ form: "issue-action", ...target });
  };

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
    onDiscipline: (event: WorkerSafetyEventRow) => issue({ safetyEvent: event.id }),
    preloadEdit: RecordSafetyEventPane.preload,
    preloadDiscipline: IssueActionPane.preload,
  };
  const eventIntent = preloadOn(SafetyEventPane);
  const eventRows: VirtualRow[] = shown.map((event) => ({
    key: event.id,
    render: () => (
      <SafetyEventRow
        event={event}
        permissions={permissions}
        busy={busy}
        handlers={handlers}
        openIntent={eventIntent}
      />
    ),
  }));

  return (
    <div className="flex min-w-0 flex-col gap-4 text-sm">
      <SafetyScorecardCard
        scorecard={scorecard}
        events={events}
        canRecord={canRecord}
        canRecognise={canRecognise}
        onRecordEvent={() => go({ form: "record-safety-event" })}
        onRecognise={() => go({ form: "add-recognition" })}
        recordIntent={preloadOn(RecordSafetyEventPane)}
        recogniseIntent={preloadOn(RecognitionPane)}
      />

      <div className="@container">
        <div className="grid items-start gap-4 @5xl:grid-cols-[minmax(0,1fr)_24rem]">
          <SectionPanel
            title={t("Events")}
            count={capped ? undefined : events.length}
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
                            {formatNumber(scorecard.openEvents)}
                          </span>
                        </>
                      ),
                    },
                    {
                      value: "all",
                      label: capped ? (
                        t("All")
                      ) : (
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
            ) : fromServer && openQuery.isError ? (
              <div className="p-3">
                <ErrorState
                  layout="compact"
                  error={openQuery.error}
                  title={t("The open events could not be loaded")}
                  onRetry={() => void openQuery.refetch()}
                />
              </div>
            ) : shown.length === 0 ? (
              <SectionPanelQuiet>
                {t("No open events. Everything on the record is closed.")}
              </SectionPanelQuiet>
            ) : (
              <VirtualRows
                rows={eventRows}
                mask="card"
                className="max-h-136"
                contentClassName="flex flex-col gap-px px-1.5 py-1"
                hoverGlide
                estimateSize={96}
                footer={
                  <ListCapLine
                    count={fromServer && openQuery.data ? openQuery.data.length : events.length}
                  />
                }
                aria-label={t("Events")}
              />
            )}
          </SectionPanel>

          <div className="flex min-w-0 flex-col gap-4">
            <DisciplineStair
              ladder={ladder}
              actions={actions}
              preview={open.form === "issue-action" ? (picked ?? suggestedLevel) : null}
              canIssue={canDiscipline}
              canRescind={canRescind}
              onIssue={() => issue({})}
              issueIntent={preloadOn(IssueActionPane)}
              openActionIntent={preloadOn(DisciplinaryActionPane)}
              preloadEvent={SafetyEventPane.preload}
              onOpenAction={(action) => go({ action: action.id })}
              onOpenEvent={(safetyEventId) => go({ safetyEvent: safetyEventId })}
              onRescind={(action) => setDialog({ kind: "rescind", action })}
            />
            <RecognitionList
              recognitions={recognitionsRead}
              workerName={workerName}
              canAdd={canRecognise}
              canDelete={canDeleteRecognition}
              removingId={
                removeRecognition.isPending ? (removeRecognition.variables?.id ?? null) : null
              }
              onAdd={() => go({ form: "add-recognition" })}
              addIntent={preloadOn(RecognitionPane)}
              onDelete={(recognition) => setDialog({ kind: "remove-recognition", recognition })}
            />
            {canReadInjury ? (
              <InjuryList
                workerId={workerId}
                injuries={injuries}
                canRecord={canRecordInjury}
                canUpdate={canUpdateInjury}
                canDelete={canDeleteInjury}
                onRecord={() => go({ form: "record-injury" })}
                onEdit={(injury) => go({ injury: injury.id })}
                paneIntent={preloadOn(InjuryPane)}
                onDeleted={(injury) => {
                  if (open.injury === injury.id) onOpen({ tab: "safety" });
                }}
              />
            ) : null}
          </div>
        </div>
      </div>

      <SafetyEventPane.Component
        open={open.safetyEvent !== null && open.form === null && !eventRecord.isPending}
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
      <RecordSafetyEventPane.Component
        open={
          open.form === "record-safety-event" &&
          (open.safetyEvent === null ? canRecord : canUpdate && formEvent !== null)
        }
        onOpenChange={(next) => !next && close()}
        workerId={workerId}
        event={formEvent}
        onSaved={close}
      />
      <IssueActionPane.Component
        open={open.form === "issue-action" && canDiscipline}
        onOpenChange={(next) => !next && close()}
        workerId={workerId}
        workerName={workerName}
        suggestedLevel={suggestedLevel}
        highestLevel={(ladder.highestLevel ?? null) as DisciplinaryLevel | null}
        event={formEvent}
        onLevelChange={setPicked}
        onSaved={close}
      />
      <RecognitionPane.Component
        open={open.form === "add-recognition" && canRecognise}
        onOpenChange={(next) => !next && close()}
        workerId={workerId}
        workerName={workerName}
        onSaved={close}
      />
      <InjuryPane.Component
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
      <DisciplinaryActionPane.Component
        open={open.action !== null && open.form === null && !actionRecord.isPending}
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
