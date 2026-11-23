import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { VirtualRows } from "@/components/virtual-rows";
import { lazyWorkPane, preloadOn, useWorkPane } from "@/components/work-pane";
import { useOpenedOnce } from "@/hooks/use-opened-once";
import { lazyComponent } from "@/lib/lazy-component";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  closeLeaveCase,
  decideLeaveCase,
  deleteLeaveDay,
  fetchWorkerLeaveFile,
  recordLeaveCertification,
  requestLeaveCertification,
  WORKER_LEAVE_KEY,
  type LeaveCaseRow,
  type LeaveEntitlement,
  type LeaveEntryRow,
} from "@/lib/graphql/worker-leave";
import type { WorkerTabProps } from "@/routes/worker/use-worker-state";
import { useQuery } from "@tanstack/react-query";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { ChevronRightIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatRange, formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  certificationStatusLabel,
  certificationTone,
  formatLeaveHours,
  leaveCaseStatusLabel,
  leaveCaseStatusTone,
  leaveFrequencyLabel,
  leaveTypeLabel,
  measurementMethodLabel,
} from "@trenova/shared/lib/leave";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useId, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { TimeSkeleton } from "../time-skeleton";
import { EntitlementBar } from "./entitlement-bar";
import { caseHours } from "./leave-model";
import {
  BOUNDED_CELL_CLASS,
  BoundedTable,
  type BoundedColumn,
  type BoundedRowProps,
} from "../../bounded-table";
import { useWorkerInvalidation } from "../../use-worker-invalidation";
import { WORKER_FORM_PANES } from "../../worker-form-panes";

const LeaveCasePane = lazyWorkPane(
  WORKER_FORM_PANES["open-leave-case"],
  (module) => module.LeaveCasePane,
  {
    openKey: (props) => props.leaveCase?.id ?? "new",
    title: (props, t) => (props.leaveCase ? t("Edit leave case") : t("Open a leave case")),
    fields: 8,
  },
);
const LeaveDayDialog = lazyComponent(
  () => import("./leave-day-dialog"),
  (module) => module.LeaveDayDialog,
);

type LeaveViewProps = Pick<WorkerTabProps, "workerId" | "open" | "onOpen">;

function CasesSketch() {
  return (
    <div className="flex flex-col gap-3 rounded-md border border-border/70 bg-card p-3 text-left">
      {[0, 1].map((row) => (
        <div key={row} className="flex flex-col gap-1.5">
          <GhostLine className="w-1/3" />
          <GhostLine className="w-2/3" />
        </div>
      ))}
    </div>
  );
}

/**
 * Leave: the FMLA entitlement measured from the days recorded, and every case with its decision,
 * certification and the days drawn against it. It carries why someone is off work, so a reader
 * without the grant is told so rather than shown an empty screen.
 */
export default function LeaveView({ workerId, open, onOpen }: LeaveViewProps) {
  const t = useT();
  const pane = useWorkPane();
  const { allowed: canRead } = usePermission(Resource.WorkerLeave, Operation.Read);
  const { allowed: canRecord } = usePermission(Resource.WorkerLeave, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerLeave, Operation.Update);
  const fileQuery = useQuery({
    queryKey: [WORKER_LEAVE_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerLeaveFile(workerId, { signal }),
    enabled: canRead,
  });
  // The case stays set after the dialog closes, so its content does not empty while it leaves.
  const [recording, setRecording] = useState<{ open: boolean; leaveCase: LeaveCaseRow | null }>({
    open: false,
    leaveCase: null,
  });
  const dayDialogOpened = useOpenedOnce(recording.open);

  if (!canRead) {
    return (
      <EmptySheet
        sketch={<CasesSketch />}
        title={t("Leave is not shared with you")}
        description={t(
          "Leave records say why someone is off work and hold their medical certification, so they need the leave permission to read.",
        )}
      />
    );
  }
  if (fileQuery.isPending) return <TimeSkeleton view="leave" />;
  if (fileQuery.isError) {
    return (
      <ErrorState
        layout="sheet"
        error={fileQuery.error}
        onRetry={() => void fileQuery.refetch()}
        title={t("The leave could not be loaded")}
      />
    );
  }

  const { entitlement, cases } = fileQuery.data;
  const editing = open.leave ? (cases.find((row) => row.id === open.leave) ?? null) : null;
  const openCase = () =>
    pane.closeThen(() => onOpen({ tab: "time", view: "leave", form: "open-leave-case" }));

  return (
    <div className="grid min-w-0 items-start gap-4 xl:grid-cols-[minmax(0,1fr)_24rem]">
      <SectionPanel title={t("Cases")} count={cases.length} hint={t("Every case on file")}>
        {cases.length === 0 ? (
          <SectionPanelQuiet>
            {t("No leave case has been opened for this worker.")}
          </SectionPanelQuiet>
        ) : (
          <VirtualRows
            rows={cases.map((leaveCase) => ({
              key: leaveCase.id,
              render: () => (
                <CaseRow
                  workerId={workerId}
                  leaveCase={leaveCase}
                  canRecord={canRecord}
                  canUpdate={canUpdate}
                  onRecordDay={() => setRecording({ open: true, leaveCase })}
                  onEdit={() =>
                    pane.closeThen(() =>
                      onOpen({ tab: "time", view: "leave", leave: leaveCase.id }),
                    )
                  }
                />
              ),
            }))}
            aria-label={t("Cases")}
            mask="card"
            className="max-h-160"
            contentClassName="px-1.5 py-1"
            estimateSize={120}
          />
        )}
      </SectionPanel>
      <EntitlementPanel entitlement={entitlement} canRecord={canRecord} onOpenCase={openCase} />
      {canRecord || canUpdate ? (
        <LeaveCasePane.Component
          open={open.form === "open-leave-case" || editing !== null}
          onOpenChange={(next) => {
            if (!next) onOpen({ tab: "time", view: "leave" });
          }}
          workerId={workerId}
          leaveCase={open.form === "open-leave-case" ? null : editing}
        />
      ) : null}
      {dayDialogOpened ? (
        <LazyComponent fallback={null}>
          <LeaveDayDialog.Component
            open={recording.open}
            onOpenChange={(next) => setRecording((current) => ({ ...current, open: next }))}
            workerId={workerId}
            leaveCase={recording.leaveCase}
          />
        </LazyComponent>
      ) : null}
    </div>
  );
}

function EntitlementPanel({
  entitlement,
  canRecord,
  onOpenCase,
}: {
  entitlement: LeaveEntitlement;
  canRecord: boolean;
  onOpenCase: () => void;
}) {
  const t = useT();
  return (
    <SectionPanel
      title={t("FMLA entitlement")}
      help={
        <>
          <p>
            {t(
              "Measured from the days recorded against this worker's cases, inside the window the measurement method sets. Only days on a case designated as FMLA draw it down: approving a case and designating it are separate decisions.",
            )}
          </p>
          <p>
            {t(
              "Exhausted means nothing is left. Days designated after the fact can push use past the entitlement; the remainder then reads zero rather than negative.",
            )}
          </p>
        </>
      }
    >
      <div className="flex flex-col gap-2.5 px-3 pt-3 pb-3.5">
        <p className="flex flex-wrap items-baseline gap-x-1 gap-y-1">
          <b className="text-xl font-semibold tabular-nums">
            {formatLeaveHours(entitlement.remainingHours)}
          </b>
          <span className="text-sm text-muted-foreground tabular-nums">
            {t("of {0} hours left", formatLeaveHours(entitlement.totalHours))}
          </span>
          {entitlement.exhausted ? (
            <Badge variant="danger" dot className="ml-1">
              {t("Exhausted")}
            </Badge>
          ) : null}
        </p>
        {entitlement.militaryCaregiver || !entitlement.eligibleOnTenure ? (
          <p className="flex flex-wrap gap-1.5">
            {entitlement.militaryCaregiver ? (
              <Badge variant="info">{t("Military caregiver, 26 weeks")}</Badge>
            ) : null}
            {entitlement.eligibleOnTenure ? null : (
              <Badge variant="warning" dot>
                {t("Under 12 months' service")}
              </Badge>
            )}
          </p>
        ) : null}
        <EntitlementBar entitlement={entitlement} />
      </div>
      <DescriptionList layout="ruled" columns={2} className="rounded-none border-0 border-t">
        <DescriptionItem label={t("Remaining")} numeric>
          {t("{0} weeks", formatLeaveHours(entitlement.remainingWeeks))}
        </DescriptionItem>
        <DescriptionItem label={t("Used")} numeric>
          {t("{0} h", formatLeaveHours(entitlement.usedHours))}
        </DescriptionItem>
        <DescriptionItem label={t("Entitlement")} numeric>
          {t("{0} h", formatLeaveHours(entitlement.totalHours))}
        </DescriptionItem>
        <DescriptionItem label={t("Months employed")} numeric>
          {String(entitlement.monthsEmployed)}
        </DescriptionItem>
      </DescriptionList>
      <div className="flex flex-col items-start justify-between gap-3 border-t border-border-subtle px-3 py-2.5 sm:flex-row sm:items-center">
        <p className="text-xs text-muted-foreground">
          {t(
            "{0}, {1}.",
            measurementMethodLabel(entitlement.method),
            formatRange(entitlement.window.from, entitlement.window.through, {
              year: "always",
              join: "words",
            }),
          )}
        </p>
        {canRecord ? (
          <Button size="sm" className="shrink-0" {...preloadOn(LeaveCasePane)} onClick={onOpenCase}>
            {t("Open a leave case")}
          </Button>
        ) : null}
      </div>
    </SectionPanel>
  );
}

type CaseConfirm = "deny" | "close" | null;

function CaseRow({
  workerId,
  leaveCase,
  canRecord,
  canUpdate,
  onRecordDay,
  onEdit,
}: {
  workerId: string;
  leaveCase: LeaveCaseRow;
  canRecord: boolean;
  canUpdate: boolean;
  onRecordDay: () => void;
  onEdit: () => void;
}) {
  const t = useT();
  const { allowed: canDecide } = usePermission(Resource.WorkerLeave, Operation.Approve);
  const invalidate = useWorkerInvalidation(workerId, "leave");
  const [expanded, setExpanded] = useState(false);
  const [confirm, setConfirm] = useState<CaseConfirm>(null);
  const daysId = useId();
  const scope = { id: `worker-leave-case-action:${leaveCase.id}` };
  const decide = useApiMutation({
    resourceName: "Leave case",
    scope,
    mutationFn: (input: { approve: boolean; designate: boolean }) =>
      decideLeaveCase({ caseId: leaveCase.id, ...input }),
    onSuccess: () => invalidate(),
  });
  const close = useApiMutation({
    resourceName: "Leave case",
    scope,
    mutationFn: () => closeLeaveCase(leaveCase.id),
    onSuccess: () => invalidate(),
  });
  const requestCert = useApiMutation({
    resourceName: "Leave case",
    scope,
    mutationFn: () => requestLeaveCertification(leaveCase.id),
    onSuccess: () => invalidate(),
  });
  const receiveCert = useApiMutation({
    resourceName: "Leave case",
    scope,
    mutationFn: () => recordLeaveCertification({ caseId: leaveCase.id, status: "Received" }),
    onSuccess: () => invalidate(),
  });
  const busy =
    decide.isPending || close.isPending || requestCert.isPending || receiveCert.isPending;
  const hours = caseHours(leaveCase);
  const meta = [
    leaveCase.endsAt
      ? formatRange(leaveCase.startsAt, leaveCase.endsAt, { year: "always", join: "words" })
      : t("From {0}, end not known", formatUnixDateMedium(leaveCase.startsAt)),
    t("{0} hours", formatLeaveHours(String(hours))),
    leaveCase.certificationReceivedAt
      ? t("certification received {0}", formatUnixDateMedium(leaveCase.certificationReceivedAt))
      : leaveCase.certificationDueAt
        ? t("certification due {0}", formatUnixDateMedium(leaveCase.certificationDueAt))
        : null,
  ].filter((part): part is string => part !== null);
  const pendingDecision = canDecide && leaveCase.status === "Pending";
  const closable = canDecide && (leaveCase.status === "Approved" || leaveCase.status === "Denied");

  return (
    <div className="border-b border-border-subtle px-2 py-2.5">
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
        <span className="font-medium">
          {t(
            "{0} · {1}",
            leaveTypeLabel(leaveCase.leaveType),
            leaveFrequencyLabel(leaveCase.frequency),
          )}
        </span>
        <Badge variant={leaveCaseStatusTone(leaveCase.status)} dot>
          {leaveCaseStatusLabel(leaveCase.status)}
        </Badge>
        {leaveCase.fmlaDesignated ? <Badge variant="info">{t("Designated FMLA")}</Badge> : null}
        {leaveCase.certificationStatus === "NotRequired" ? null : (
          <Badge variant={certificationTone(leaveCase.certificationStatus)} dot>
            {t(
              "Certification: {0}",
              certificationStatusLabel(leaveCase.certificationStatus).toLowerCase(),
            )}
          </Badge>
        )}
        {leaveCase.certificationLate ? (
          <Badge variant="danger" dot>
            {t("Past the deadline")}
          </Badge>
        ) : null}
      </div>
      <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">{meta.join(" · ")}</p>
      {leaveCase.reason ? <p className="mt-1">{leaveCase.reason}</p> : null}
      {leaveCase.notes ? (
        <p className="mt-0.5 text-xs text-muted-foreground">{leaveCase.notes}</p>
      ) : null}
      <div className="mt-2 flex flex-wrap items-center gap-1.5">
        {pendingDecision ? (
          <>
            <Button
              size="xs"
              isLoading={decide.isPending && decide.variables?.designate === true}
              disabled={busy}
              onClick={() =>
                void decide.mutateAsync({ approve: true, designate: true }).catch(() => undefined)
              }
            >
              {t("Approve and designate FMLA")}
            </Button>
            <Button
              size="xs"
              variant="outline"
              isLoading={
                decide.isPending &&
                decide.variables?.approve === true &&
                decide.variables.designate === false
              }
              disabled={busy}
              onClick={() =>
                void decide.mutateAsync({ approve: true, designate: false }).catch(() => undefined)
              }
            >
              {t("Approve only")}
            </Button>
            <Button size="xs" variant="ghost" disabled={busy} onClick={() => setConfirm("deny")}>
              {t("Deny")}
            </Button>
          </>
        ) : null}
        {canRecord && leaveCase.status === "Approved" ? (
          <Button
            size="xs"
            variant="outline"
            disabled={busy}
            {...preloadOn(LeaveDayDialog)}
            onClick={onRecordDay}
          >
            {t("Record a day")}
          </Button>
        ) : null}
        {canUpdate && leaveCase.certificationStatus === "NotRequired" ? (
          <Button
            size="xs"
            variant="ghost"
            isLoading={requestCert.isPending}
            disabled={busy}
            onClick={() => void requestCert.mutateAsync().catch(() => undefined)}
          >
            {t("Request certification")}
          </Button>
        ) : null}
        {canUpdate && leaveCase.certificationStatus === "Requested" ? (
          <Button
            size="xs"
            variant="ghost"
            isLoading={receiveCert.isPending}
            disabled={busy}
            onClick={() => void receiveCert.mutateAsync().catch(() => undefined)}
          >
            {t("Mark received")}
          </Button>
        ) : null}
        {canUpdate ? (
          <Button
            size="xs"
            variant="ghost"
            disabled={busy}
            {...preloadOn(LeaveCasePane)}
            onClick={onEdit}
          >
            {t("Edit")}
          </Button>
        ) : null}
        {closable ? (
          <Button size="xs" variant="ghost" disabled={busy} onClick={() => setConfirm("close")}>
            {t("Close")}
          </Button>
        ) : null}
      </div>
      {leaveCase.entries.length > 0 ? (
        <>
          <button
            type="button"
            aria-expanded={expanded}
            aria-controls={daysId}
            onClick={() => setExpanded((value) => !value)}
            className="mt-2 inline-flex items-center gap-1 rounded-sm ui-focus-ring pr-1 text-xs font-medium text-muted-foreground hover:text-foreground"
          >
            <ChevronRightIcon
              aria-hidden
              className={cn(
                "size-3 transition-transform duration-200 ease-settle",
                expanded && "rotate-90",
              )}
            />
            {t("Day entries")}
            <span className="font-normal text-foreground-subtle tabular-nums">
              {leaveCase.entries.length}
            </span>
          </button>
          {expanded ? (
            <div
              id={daysId}
              className="mt-2 animate-expand-in overflow-hidden rounded-md border border-border-subtle motion-reduce:animate-none"
            >
              <DayEntries workerId={workerId} entries={leaveCase.entries} canUpdate={canUpdate} />
            </div>
          ) : null}
        </>
      ) : null}
      <ConfirmDialog
        open={confirm === "deny"}
        onOpenChange={(next) => !next && setConfirm(null)}
        title={t("Deny this leave case?")}
        description={t(
          "The case is marked denied, no days can be recorded against it and nothing is drawn from the entitlement.",
        )}
        confirmLabel={t("Deny case")}
        pending={decide.isPending}
        onConfirm={() => decide.mutateAsync({ approve: false, designate: false })}
      />
      <ConfirmDialog
        variant="default"
        open={confirm === "close"}
        onOpenChange={(next) => !next && setConfirm(null)}
        title={t("Close this leave case?")}
        description={t(
          "No more days can be recorded against it. Closing fills in the expected end if it is still blank.",
        )}
        confirmLabel={t("Close case")}
        pending={close.isPending}
        onConfirm={() => close.mutateAsync()}
      />
    </div>
  );
}

function dayColumns(t: TranslateFn, canUpdate: boolean): BoundedColumn[] {
  return [
    { key: "day", label: t("Day") },
    { key: "hours", label: t("Hours"), align: "end" },
    { key: "entry", label: t("Entry") },
    ...(canUpdate
      ? [{ key: "actions", label: <span className="sr-only">{t("Actions")}</span>, shrink: true }]
      : []),
  ];
}

function DayEntries({
  workerId,
  entries,
  canUpdate,
}: {
  workerId: string;
  entries: readonly LeaveEntryRow[];
  canUpdate: boolean;
}) {
  const t = useT();
  const invalidate = useWorkerInvalidation(workerId, "leave");
  const [removing, setRemoving] = useState<LeaveEntryRow | null>(null);
  const remove = useApiMutation({
    resourceName: "Day",
    scope: { id: `worker-leave-day-remove:${workerId}` },
    mutationFn: (id: string) => deleteLeaveDay(id),
    onSuccess: () => invalidate(),
  });
  const sorted = [...entries].sort((a, b) => a.usedOn - b.usedOn);

  return (
    <>
      <BoundedTable
        label={t("Day entries")}
        columns={dayColumns(t, canUpdate)}
        maxHeightClassName="max-h-72"
        rows={sorted.map((entry) => ({
          key: entry.id,
          render: (props) => (
            <DayEntryRow
              props={props}
              entry={entry}
              canUpdate={canUpdate}
              onRemove={() => setRemoving(entry)}
            />
          ),
        }))}
      />
      <ConfirmDialog
        open={removing !== null}
        onOpenChange={(next) => !next && setRemoving(null)}
        title={t("Remove this day?")}
        description={
          removing === null
            ? ""
            : removing.countsAgainstEntitlement
              ? t(
                  "{0} hours on {1} come off the case and go back onto the entitlement.",
                  formatLeaveHours(removing.hours),
                  formatUnixDateMedium(removing.usedOn),
                )
              : t(
                  "{0} hours on {1} come off the case. They were not counted against the entitlement.",
                  formatLeaveHours(removing.hours),
                  formatUnixDateMedium(removing.usedOn),
                )
        }
        confirmLabel={t("Remove day")}
        pending={remove.isPending}
        onConfirm={() => (removing ? remove.mutateAsync(removing.id) : Promise.resolve())}
      />
    </>
  );
}

function DayEntryRow({
  props,
  entry,
  canUpdate,
  onRemove,
}: {
  props: BoundedRowProps;
  entry: LeaveEntryRow;
  canUpdate: boolean;
  onRemove: () => void;
}) {
  const t = useT();
  return (
    <tr {...props} data-glide-target>
      <td className={cn(BOUNDED_CELL_CLASS, "font-mono text-xs tabular-nums")}>
        {formatUnixDateMedium(entry.usedOn)}
      </td>
      <td className={cn(BOUNDED_CELL_CLASS, "text-right font-mono text-xs tabular-nums")}>
        {t("{0} h", formatLeaveHours(entry.hours))}
      </td>
      <td className={BOUNDED_CELL_CLASS}>
        <span className="flex flex-wrap items-center gap-1.5">
          {entry.notes ? <span>{entry.notes}</span> : null}
          {entry.countsAgainstEntitlement ? null : (
            <Badge variant="neutral">{t("Not counted")}</Badge>
          )}
        </span>
      </td>
      {canUpdate ? (
        <td className={cn(BOUNDED_CELL_CLASS, "text-right")}>
          <Button size="xs" variant="ghost" onClick={onRemove}>
            {t("Remove")}
          </Button>
        </td>
      ) : null}
    </tr>
  );
}
