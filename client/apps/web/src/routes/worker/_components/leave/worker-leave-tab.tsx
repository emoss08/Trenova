import { InfoPopover } from "@/components/info-popover";
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
} from "@/lib/graphql/worker-leave";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Progress } from "@trenova/shared/components/ui/progress";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  certificationStatusLabel,
  certificationTone,
  entitlementUsedPercent,
  formatLeaveHours,
  leaveCaseStatusLabel,
  leaveCaseStatusTone,
  leaveFrequencyLabel,
  leaveTypeLabel,
  measurementMethodLabel,
} from "@trenova/shared/lib/leave";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { toast } from "sonner";
import { LeaveCaseDialog } from "./leave-case-dialog";
import { LeaveDayDialog } from "./leave-day-dialog";
import { useLeaveInvalidation } from "./use-leave-invalidation";

type DialogState =
  | { kind: "case"; leaveCase: LeaveCaseRow | null }
  | { kind: "day"; leaveCase: LeaveCaseRow };

export default function WorkerLeaveTab({ workerId }: { workerId: string }) {
  const { allowed: canRead } = usePermission(Resource.WorkerLeave, Operation.Read);
  const { allowed: canRecord } = usePermission(Resource.WorkerLeave, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerLeave, Operation.Update);
  const { allowed: canDecide } = usePermission(Resource.WorkerLeave, Operation.Approve);
  const invalidate = useLeaveInvalidation(workerId);
  const [dialog, setDialog] = useState<DialogState | null>(null);

  const fileQuery = useQuery({
    queryKey: [WORKER_LEAVE_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerLeaveFile(workerId, { signal }),
    enabled: canRead,
  });

  const decide = useMutation({
    mutationFn: ({
      caseId,
      approve,
      designate,
    }: {
      caseId: string;
      approve: boolean;
      designate: boolean;
    }) => decideLeaveCase({ caseId, approve, designate }),
    onSuccess: (saved) => {
      toast.success(`Leave case ${leaveCaseStatusLabel(saved.status).toLowerCase()}`, {
        description: saved.fmlaDesignated
          ? "Designated as FMLA, so days recorded against it draw the entitlement down."
          : "Not designated as FMLA, so days recorded against it draw nothing down.",
      });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not decide the case", { description: error.message }),
  });

  const close = useMutation({
    mutationFn: (id: string) => closeLeaveCase(id),
    onSuccess: () => {
      toast.success("Leave case closed", {
        description: "The days recorded against it stay drawn down; they were taken.",
      });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not close the case", { description: error.message }),
  });

  const requestCert = useMutation({
    mutationFn: (id: string) => requestLeaveCertification(id),
    onSuccess: (saved) => {
      toast.success("Certification requested", {
        description: saved.certificationDueAt
          ? `Due ${formatUnixDate(saved.certificationDueAt)}.`
          : undefined,
      });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not request certification", { description: error.message }),
  });

  const receiveCert = useMutation({
    mutationFn: (id: string) => recordLeaveCertification({ caseId: id, status: "Received" }),
    onSuccess: () => {
      toast.success("Certification received");
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not record the certification", { description: error.message }),
  });

  const removeDay = useMutation({
    mutationFn: (id: string) => deleteLeaveDay(id),
    onSuccess: () => {
      toast.success("Day removed", { description: "The hours go back to the entitlement." });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not remove the day", { description: error.message }),
  });

  // Leave carries the reason somebody is off work and the certification behind
  // it. Without the grant the tab shows nothing at all.
  if (!canRead) return null;

  if (fileQuery.isLoading) {
    return (
      <div className="flex flex-col gap-3 p-4">
        {[0, 1].map((row) => (
          <Skeleton key={row} className="h-28 w-full" />
        ))}
      </div>
    );
  }

  const file = fileQuery.data;
  if (!file) return null;

  const { entitlement } = file;
  const usedPercent = entitlementUsedPercent(entitlement.usedHours, entitlement.totalHours);

  return (
    <div className="flex flex-col gap-4 p-4">
      <section
        className={cn(
          "rounded-lg border p-4",
          entitlement.exhausted && "border-amber-500/60 bg-amber-500/5",
        )}
      >
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="text-sm font-medium">FMLA entitlement</h3>
              <InfoPopover title="FMLA entitlement">
                <p>
                  Measured on read from the days recorded against this worker&apos;s cases, inside
                  the window the measurement method sets. Only days on a case designated as FMLA
                  draw it down: approving a case and designating it FMLA are separate decisions, so
                  approved leave that was not designated is recorded but counts for nothing.
                </p>
                <p>
                  Exhausted means nothing is left. Days designated after the fact can push use past
                  the entitlement; the remainder then reads zero rather than negative.
                </p>
              </InfoPopover>
              {entitlement.exhausted ? <Badge variant="inactive">Exhausted</Badge> : null}
              {entitlement.militaryCaregiver ? (
                <Badge variant="info">Military caregiver — 26 weeks</Badge>
              ) : null}
              {entitlement.eligibleOnTenure ? null : (
                <Badge variant="warning">Under 12 months&apos; service</Badge>
              )}
            </div>
            <p className="text-muted-foreground mt-1 text-xs">
              {measurementMethodLabel(entitlement.method)} ·{" "}
              {formatUnixDate(entitlement.window.from)} to{" "}
              {formatUnixDate(entitlement.window.through)}
            </p>
          </div>
          {canRecord ? (
            <Button size="sm" onClick={() => setDialog({ kind: "case", leaveCase: null })}>
              Open a case
            </Button>
          ) : null}
        </div>

        <Progress value={usedPercent} className="mt-4" />

        <dl className="mt-3 grid grid-cols-2 gap-3 text-xs sm:grid-cols-4">
          <Figure
            label="Remaining"
            value={`${formatLeaveHours(entitlement.remainingHours)} h`}
            detail={`${formatLeaveHours(entitlement.remainingWeeks)} weeks`}
          />
          <Figure
            label="Used"
            value={`${formatLeaveHours(entitlement.usedHours)} h`}
            detail={`${formatLeaveHours(entitlement.usedWeeks)} weeks`}
          />
          <Figure
            label="Entitlement"
            value={`${formatLeaveHours(entitlement.totalHours)} h`}
            detail={`${formatLeaveHours(entitlement.totalWeeks)} weeks`}
          />
          <Figure label="Months employed" value={String(entitlement.monthsEmployed)} />
        </dl>

        <p className="text-muted-foreground mt-3 text-[11px]">
          The 1,250-hour half of the eligibility test is recorded on each case: there is no
          timeclock here to answer it from.
        </p>
      </section>

      <section>
        <h3 className="cc-label text-foreground mb-2">Cases</h3>
        {file.cases.length === 0 ? (
          <p className="text-muted-foreground rounded-md border border-dashed p-3 text-xs">
            No leave case has been opened for this worker.
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {file.cases.map((leaveCase) => (
              <li key={leaveCase.id} className="rounded-md border px-3 py-2 text-xs">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="flex flex-wrap items-center gap-2">
                    <span className="font-medium">{leaveTypeLabel(leaveCase.leaveType)}</span>
                    <Badge variant={leaveCaseStatusTone(leaveCase.status)}>
                      {leaveCaseStatusLabel(leaveCase.status)}
                    </Badge>
                    {leaveCase.fmlaDesignated ? (
                      <Badge variant="info">Designated FMLA</Badge>
                    ) : null}
                    <Badge variant="secondary">{leaveFrequencyLabel(leaveCase.frequency)}</Badge>
                    {leaveCase.certificationStatus === "NotRequired" ? null : (
                      <Badge variant={certificationTone(leaveCase.certificationStatus)}>
                        Certification{" "}
                        {certificationStatusLabel(leaveCase.certificationStatus).toLowerCase()}
                      </Badge>
                    )}
                    {leaveCase.certificationLate ? (
                      <Badge variant="inactive">Past the deadline</Badge>
                    ) : null}
                  </span>
                  <span className="flex flex-wrap items-center gap-1">
                    {canDecide && leaveCase.status === "Pending" ? (
                      <>
                        <Button
                          size="xs"
                          isLoading={decide.isPending}
                          onClick={() =>
                            decide.mutate({
                              caseId: leaveCase.id,
                              approve: true,
                              designate: true,
                            })
                          }
                        >
                          Approve &amp; designate
                        </Button>
                        <Button
                          size="xs"
                          variant="outline"
                          onClick={() =>
                            decide.mutate({
                              caseId: leaveCase.id,
                              approve: true,
                              designate: false,
                            })
                          }
                        >
                          Approve only
                        </Button>
                        <Button
                          size="xs"
                          variant="ghost"
                          onClick={() =>
                            decide.mutate({
                              caseId: leaveCase.id,
                              approve: false,
                              designate: false,
                            })
                          }
                        >
                          Deny
                        </Button>
                      </>
                    ) : null}
                    {canRecord && leaveCase.status === "Approved" ? (
                      <Button
                        size="xs"
                        variant="outline"
                        onClick={() => setDialog({ kind: "day", leaveCase })}
                      >
                        Record a day
                      </Button>
                    ) : null}
                    {canUpdate && leaveCase.certificationStatus === "NotRequired" ? (
                      <Button
                        size="xs"
                        variant="ghost"
                        isLoading={requestCert.isPending}
                        onClick={() => requestCert.mutate(leaveCase.id)}
                      >
                        Request certification
                      </Button>
                    ) : null}
                    {canUpdate && leaveCase.certificationStatus === "Requested" ? (
                      <Button
                        size="xs"
                        variant="ghost"
                        isLoading={receiveCert.isPending}
                        onClick={() => receiveCert.mutate(leaveCase.id)}
                      >
                        Mark received
                      </Button>
                    ) : null}
                    {canUpdate ? (
                      <Button
                        size="xs"
                        variant="ghost"
                        onClick={() => setDialog({ kind: "case", leaveCase })}
                      >
                        Edit
                      </Button>
                    ) : null}
                    {canDecide &&
                    (leaveCase.status === "Approved" || leaveCase.status === "Denied") ? (
                      <Button
                        size="xs"
                        variant="ghost"
                        isLoading={close.isPending}
                        onClick={() => close.mutate(leaveCase.id)}
                      >
                        Close
                      </Button>
                    ) : null}
                  </span>
                </div>

                <p className="text-muted-foreground mt-1">
                  {formatUnixDate(leaveCase.startsAt)}
                  {leaveCase.endsAt ? ` – ${formatUnixDate(leaveCase.endsAt)}` : " – open"}
                  {leaveCase.reason ? ` · ${leaveCase.reason}` : ""}
                  {leaveCase.certificationDueAt
                    ? ` · certification due ${formatUnixDate(leaveCase.certificationDueAt)}`
                    : ""}
                </p>

                {leaveCase.entries.length > 0 ? (
                  <ul className="mt-2 flex flex-col gap-1">
                    {leaveCase.entries.map((entry) => (
                      <li
                        key={entry.id}
                        className="border-border/60 flex items-center justify-between border-t pt-1"
                      >
                        <span className="flex items-center gap-2">
                          <span className="tabular-nums">{formatUnixDate(entry.usedOn)}</span>
                          <span className="tabular-nums">{formatLeaveHours(entry.hours)} h</span>
                          {entry.countsAgainstEntitlement ? null : (
                            <Badge variant="secondary">Not counted</Badge>
                          )}
                        </span>
                        {canUpdate ? (
                          <Button
                            size="xs"
                            variant="ghost"
                            isLoading={removeDay.isPending}
                            onClick={() => removeDay.mutate(entry.id)}
                          >
                            Remove
                          </Button>
                        ) : null}
                      </li>
                    ))}
                  </ul>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </section>

      <LeaveCaseDialog
        open={dialog?.kind === "case"}
        onOpenChange={(open) => !open && setDialog(null)}
        workerId={workerId}
        leaveCase={dialog?.kind === "case" ? dialog.leaveCase : null}
      />
      <LeaveDayDialog
        open={dialog?.kind === "day"}
        onOpenChange={(open) => !open && setDialog(null)}
        workerId={workerId}
        leaveCase={dialog?.kind === "day" ? dialog.leaveCase : null}
      />
    </div>
  );
}

function Figure({ label, value, detail }: { label: string; value: string; detail?: string }) {
  return (
    <div>
      <dt className="text-muted-foreground text-[11px]">{label}</dt>
      <dd className="text-sm font-semibold tabular-nums">{value}</dd>
      {detail ? <dd className="text-muted-foreground text-[11px]">{detail}</dd> : null}
    </div>
  );
}
