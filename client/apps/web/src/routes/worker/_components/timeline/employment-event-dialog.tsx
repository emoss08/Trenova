import { FleetCodeAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { driverTypeChoices, workerTypeChoices } from "@/lib/choices";
import {
  amendWorkerEmploymentEvent,
  recordWorkerEmploymentEvent,
  type EmploymentCascade,
  type RecordEmploymentEventResult,
  type WorkerEmploymentEventRow,
} from "@/lib/graphql/worker-employment";
import { zodResolver } from "@hookform/resolvers/zod";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { getTodayDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { WORKER_LEAVE_TYPE_LABELS, type WorkerLeaveType } from "@trenova/shared/types/worker";
import {
  EMPLOYMENT_EVENT_LABELS,
  EMPLOYMENT_EVENT_REQUIRES_REASON,
  employmentEventAmendSchema,
  employmentEventFormSchema,
  type EmploymentEventAmendValues,
  type EmploymentEventFormValues,
  type EmploymentEventKind,
} from "@trenova/shared/types/worker-employment";
import { TriangleAlertIcon } from "lucide-react";
import { useEffect, useMemo } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { employmentEventMeta, employmentSnapshot, recordableKinds } from "./employment-event-meta";
import { useEmploymentInvalidation } from "./use-employment-invalidation";

export type EmploymentSheetWorker = {
  fleetCodeId?: string | null;
  driverType: string;
  type: string;
  status: string;
};

export type EmploymentEventSheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  worker: EmploymentSheetWorker;
  mode: "record" | "amend";
  event?: Pick<
    WorkerEmploymentEventRow,
    "id" | "kind" | "effectiveAt" | "reason" | "notes" | "version"
  >;
  history?: readonly Pick<WorkerEmploymentEventRow, "kind">[];
  onSaved?: (event: WorkerEmploymentEventRow) => void;
};

const EMPTY_HISTORY: readonly Pick<WorkerEmploymentEventRow, "kind">[] = [];

const LEAVE_TYPE_OPTIONS = (Object.keys(WORKER_LEAVE_TYPE_LABELS) as WorkerLeaveType[]).map(
  (value) => ({ value, label: WORKER_LEAVE_TYPE_LABELS[value] }),
);

function ptoDays(value: string): string {
  const parsed = Number(value);
  return Number.isFinite(parsed)
    ? parsed.toLocaleString(undefined, { maximumFractionDigits: 2 })
    : value;
}

export function describeCascade(cascade: EmploymentCascade): string {
  const parts: string[] = [];
  if (cascade.ptoAssignmentEnded) parts.push("Ended the PTO policy assignment");
  if (cascade.payAssignmentEnded) parts.push("Ended the pay assignment");
  if (cascade.upcomingPtoCancelled > 0) {
    parts.push(
      `Cancelled ${cascade.upcomingPtoCancelled} upcoming PTO request${cascade.upcomingPtoCancelled === 1 ? "" : "s"}`,
    );
  }
  if (cascade.defaultPolicyApplied) parts.push("Enrolled in the default PTO policy");
  if (Number(cascade.ptoPaidOutDays) > 0) {
    parts.push(`Paid out ${ptoDays(cascade.ptoPaidOutDays)} PTO days`);
  }
  if (Number(cascade.ptoForfeitedDays) > 0) {
    parts.push(`Forfeited ${ptoDays(cascade.ptoForfeitedDays)} PTO days`);
  }
  if (cascade.portalAccessRevoked) parts.push("Revoked the driver portal sign-in");
  if (cascade.trainingAssigned > 0) {
    parts.push(
      `Opened ${cascade.trainingAssigned} required course${cascade.trainingAssigned === 1 ? "" : "s"}`,
    );
  }
  if (cascade.checklistStarted) parts.push("Started the matching checklist");
  return parts.length > 0 ? parts.join(" · ") : "The timeline has been updated.";
}

export function EmploymentEventSheet(props: EmploymentEventSheetProps) {
  return props.mode === "amend" && props.event ? (
    <AmendSheet {...props} event={props.event} />
  ) : (
    <RecordSheet {...props} />
  );
}

function RecordSheet({
  open,
  onOpenChange,
  workerId,
  worker,
  history = EMPTY_HISTORY,
  onSaved,
}: EmploymentEventSheetProps) {
  const invalidate = useEmploymentInvalidation(workerId);
  const today = useMemo(() => getTodayDate(), []);
  const kinds = useMemo(
    () => recordableKinds(employmentSnapshot(worker.status, history)),
    [worker.status, history],
  );
  const firstKind = kinds[0] ?? "ProbationEnded";

  const form = useForm<EmploymentEventFormValues>({
    resolver: zodResolver(employmentEventFormSchema) as Resolver<EmploymentEventFormValues>,
    defaultValues: {
      kind: kinds[0] ?? "ProbationEnded",
      effectiveAt: today,
      reason: null,
      notes: null,
      fleetCodeId: null,
      managerId: null,
      driverType: null,
      workerType: null,
      rate: null,
      rateUnit: null,
      leaveType: null,
    },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset({
      kind: firstKind,
      effectiveAt: today,
      reason: null,
      notes: null,
      fleetCodeId: null,
      managerId: null,
      driverType: null,
      workerType: null,
      rate: null,
      rateUnit: null,
      leaveType: null,
    });
  }, [open, firstKind, today, reset]);

  const kind = useWatch({ control, name: "kind" }) as EmploymentEventKind;
  const meta = employmentEventMeta(kind);
  const requiresReason = EMPLOYMENT_EVENT_REQUIRES_REASON.has(kind);

  const { mutateAsync, isPending } = useApiMutation<
    RecordEmploymentEventResult,
    EmploymentEventFormValues,
    unknown,
    EmploymentEventFormValues
  >({
    form,
    resourceName: "Employment event",
    mutationFn: (values) =>
      recordWorkerEmploymentEvent({
        workerId,
        kind: values.kind,
        effectiveAt: values.effectiveAt,
        reason: values.reason,
        notes: values.notes,
        documentId: null,
        fleetCodeId: values.kind === "Transferred" ? (values.fleetCodeId ?? undefined) : undefined,
        managerId: values.kind === "Transferred" ? (values.managerId ?? undefined) : undefined,
        driverType: values.kind === "Promoted" ? (values.driverType ?? undefined) : undefined,
        workerType: values.kind === "Promoted" ? (values.workerType ?? undefined) : undefined,
        rate: values.kind === "RateChanged" ? (values.rate ?? undefined) : undefined,
        rateUnit: values.kind === "RateChanged" ? (values.rateUnit ?? undefined) : undefined,
        leaveType: values.kind === "LeaveStarted" ? (values.leaveType ?? undefined) : undefined,
      }),
    onSuccess: (result) => {
      toast.success(`${EMPLOYMENT_EVENT_LABELS[kind] ?? "Event"} recorded`, {
        description: describeCascade(result.cascade),
      });
      // The termination stands either way, so a portal failure is a follow-up
      // task for the recorder rather than a failed save.
      if (result.cascade.portalRevocationError) {
        toast.warning("Revoke the driver portal sign-in by hand", {
          description: `The worker is terminated but Dash access is still open: ${result.cascade.portalRevocationError}`,
        });
      }
      void invalidate();
      onSaved?.(result.event);
      onOpenChange(false);
    },
  });

  const kindOptions = useMemo(
    () => kinds.map((value) => ({ value, label: EMPLOYMENT_EVENT_LABELS[value] })),
    [kinds],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Record employment event</DialogTitle>
          <DialogDescription>
            Events are the only way employment status moves. Each one is kept on the worker&apos;s
            timeline with who recorded it.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(event);
            }}
          >
            <FormGroup className="pb-2" cols={2}>
              {kind === "Terminated" ? (
                <FormControl cols="full">
                  <Alert variant="destructive" className="py-2">
                    <TriangleAlertIcon className="size-4" />
                    <AlertTitle>This ends employment</AlertTitle>
                    <AlertDescription>
                      The worker becomes inactive, their PTO and pay assignments close on the
                      effective date, and any time off starting after it is cancelled.
                    </AlertDescription>
                  </Alert>
                </FormControl>
              ) : null}
              <FormControl cols="full">
                <SelectField<EmploymentEventFormValues>
                  control={control}
                  name="kind"
                  label="Event"
                  placeholder="Select an event"
                  options={kindOptions}
                  rules={{ required: true }}
                  description={meta.hint}
                />
              </FormControl>
              <FormControl cols="full">
                <AutoCompleteDateField<EmploymentEventFormValues>
                  control={control}
                  name="effectiveAt"
                  label="Effective"
                  rules={{ required: true }}
                  placeholder="Effective date"
                />
              </FormControl>
              {kind === "Transferred" ? (
                <FormControl cols="full">
                  <FleetCodeAutocompleteField<EmploymentEventFormValues>
                    control={control}
                    name="fleetCodeId"
                    label="Fleet code"
                    placeholder="Destination fleet"
                    description="The fleet the worker moves to."
                  />
                </FormControl>
              ) : null}
              {kind === "LeaveStarted" ? (
                <FormControl cols="full">
                  <SelectField<EmploymentEventFormValues>
                    control={control}
                    name="leaveType"
                    label="Leave type"
                    options={LEAVE_TYPE_OPTIONS}
                    rules={{ required: true }}
                    placeholder="Select a leave type"
                    description="Shown on the worker's status while the leave is open."
                  />
                </FormControl>
              ) : null}
              {kind === "Promoted" ? (
                <>
                  <FormControl>
                    <SelectField<EmploymentEventFormValues>
                      control={control}
                      name="driverType"
                      label="Driver type"
                      options={driverTypeChoices}
                      isClearable
                      placeholder="Select a driver type"
                      description={`Currently ${worker.driverType}.`}
                    />
                  </FormControl>
                  <FormControl>
                    <SelectField<EmploymentEventFormValues>
                      control={control}
                      name="workerType"
                      label="Worker type"
                      options={workerTypeChoices}
                      isClearable
                      placeholder="Select a worker type"
                      description={`Currently ${worker.type}.`}
                    />
                  </FormControl>
                </>
              ) : null}
              {kind === "RateChanged" ? (
                <>
                  <FormControl>
                    <InputField<EmploymentEventFormValues>
                      control={control}
                      name="rate"
                      label="New rate"
                      placeholder="e.g. 0.62"
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <InputField<EmploymentEventFormValues>
                      control={control}
                      name="rateUnit"
                      label="Unit"
                      placeholder="per mile, per hour, salary"
                    />
                  </FormControl>
                </>
              ) : null}
              <FormControl cols="full">
                <InputField<EmploymentEventFormValues>
                  control={control}
                  name="reason"
                  label="Reason"
                  placeholder={requiresReason ? "Required" : "Optional"}
                  rules={{ required: requiresReason }}
                  maxLength={255}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<EmploymentEventFormValues>
                  control={control}
                  name="notes"
                  label="Notes"
                  placeholder="Context for whoever reads this later"
                  maxLength={4000}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button
                type="submit"
                variant={kind === "Terminated" ? "destructive" : "default"}
                disabled={isPending}
                className={cn(kind === "Terminated" && "min-w-40")}
              >
                {isPending ? "Saving..." : `Record ${EMPLOYMENT_EVENT_LABELS[kind] ?? "event"}`}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function AmendSheet({
  open,
  onOpenChange,
  workerId,
  event,
  onSaved,
}: EmploymentEventSheetProps & { event: NonNullable<EmploymentEventSheetProps["event"]> }) {
  const invalidate = useEmploymentInvalidation(workerId);
  const meta = employmentEventMeta(event.kind);

  const form = useForm<EmploymentEventAmendValues>({
    resolver: zodResolver(employmentEventAmendSchema) as Resolver<EmploymentEventAmendValues>,
    defaultValues: {
      effectiveAt: event.effectiveAt,
      reason: event.reason ?? null,
      notes: event.notes ?? null,
      amendmentNote: "",
    },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset({
      effectiveAt: event.effectiveAt,
      reason: event.reason ?? null,
      notes: event.notes ?? null,
      amendmentNote: "",
    });
  }, [open, event, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    WorkerEmploymentEventRow,
    EmploymentEventAmendValues,
    unknown,
    EmploymentEventAmendValues
  >({
    form,
    resourceName: "Employment event",
    mutationFn: (values) =>
      amendWorkerEmploymentEvent({
        id: event.id,
        effectiveAt: values.effectiveAt,
        reason: values.reason,
        notes: values.notes,
        amendmentNote: values.amendmentNote,
        version: event.version,
      }),
    onSuccess: (saved) => {
      toast.success("Event amended", {
        description: "The correction is kept alongside the original record.",
      });
      void invalidate();
      onSaved?.(saved);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Amend {meta.label.toLowerCase()}</DialogTitle>
          <DialogDescription>
            Correct the date, reason or notes. What the event already did to the worker stays as it
            is — record a new event to change state again.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup cols={1}>
              <FormControl cols="full">
                <AutoCompleteDateField<EmploymentEventAmendValues>
                  control={control}
                  name="effectiveAt"
                  label="Effective"
                  rules={{ required: true }}
                  placeholder="Effective date"
                />
              </FormControl>
              <FormControl cols="full">
                <InputField<EmploymentEventAmendValues>
                  control={control}
                  name="reason"
                  label="Reason"
                  placeholder="Reason for the event being amended"
                  maxLength={255}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<EmploymentEventAmendValues>
                  control={control}
                  name="notes"
                  label="Notes"
                  placeholder="Context for whoever reads this later"
                  maxLength={4000}
                />
              </FormControl>
              <FormControl cols="full">
                <InputField<EmploymentEventAmendValues>
                  control={control}
                  name="amendmentNote"
                  label="Why is this being amended?"
                  placeholder="e.g. Wrong effective date was entered"
                  rules={{ required: true }}
                  maxLength={255}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter className="border-border border-t pt-3">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? "Saving..." : "Save amendment"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
