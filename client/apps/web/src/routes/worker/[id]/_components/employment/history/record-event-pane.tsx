import {
  FleetCodeAutocompleteField,
  UserAutocompleteField,
} from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSingleFlight } from "@/hooks/use-single-flight";
import { driverTypeChoices, workerTypeChoices } from "@/lib/choices";
import {
  recordWorkerEmploymentEvent,
  type EmploymentEventOptions,
  type RecordEmploymentEventResult,
} from "@/lib/graphql/worker-employment";
import type { WorkerRecord } from "@/routes/worker/use-worker-state";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import type { RecordWorkerEmploymentEventInput } from "@trenova/graphql/generated/graphql";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { toast } from "@trenova/shared/components/ui/toast";
import { translate, type TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import { WORKER_LEAVE_TYPE_LABELS, type WorkerLeaveType } from "@trenova/shared/types/worker";
import {
  employmentEventFormSchema,
  employmentEventLabel,
  type EmploymentEventFormValues,
  type EmploymentEventKind,
} from "@trenova/shared/types/worker-employment";
import { FormProvider, useForm, useWatch, type FieldErrors, type Resolver } from "react-hook-form";
import { employmentOptionsQuery } from "../employment-reads";
import { describeEmploymentEffect, failedEffects } from "./employment-effects";
import { employmentEventHint, isEmploymentEventKind } from "./employment-event-meta";
import { EventPreview } from "./event-preview";
import { useWorkerInvalidation } from "../../use-worker-invalidation";

const LEAVE_TYPES = Object.keys(WORKER_LEAVE_TYPE_LABELS) as WorkerLeaveType[];

function emptyValues(kind: EmploymentEventKind | null): EmploymentEventFormValues {
  return {
    kind: (kind ?? "") as EmploymentEventKind,
    effectiveAt: getTodayDate(),
    reason: null,
    notes: null,
    fleetCodeId: null,
    managerId: null,
    driverType: null,
    workerType: null,
    rate: null,
    rateUnit: null,
    leaveType: null,
  };
}

/**
 * The shared schema, held to the server's rules for this worker: no kind is chosen until one
 * is picked, the reason is required for the kinds the server says need one, and a transfer
 * needs a fleet code or a manager (either one will do).
 */
export function employmentEventResolver(
  reasonRequired: ReadonlySet<string>,
): Resolver<EmploymentEventFormValues> {
  const base = zodResolver(employmentEventFormSchema) as Resolver<EmploymentEventFormValues>;
  return async (values, context, options) => {
    if (!values.kind) {
      return {
        values: {},
        errors: { kind: { type: "required", message: translate("Choose an event") } },
      };
    }
    const result = await base(values, context, options);
    const errors: FieldErrors<EmploymentEventFormValues> = { ...result.errors };
    if (reasonRequired.has(values.kind) && !values.reason?.trim()) {
      errors.reason = {
        type: "required",
        message: translate(
          "A reason is required to record “{0}”",
          employmentEventLabel(values.kind),
        ),
      };
    }
    if (values.kind === "Transferred" && !values.fleetCodeId && !values.managerId) {
      const message = translate("Choose a fleet code or a manager");
      errors.fleetCodeId = { type: "required", message };
      errors.managerId = { type: "required", message };
    } else if (values.kind === "Transferred") {
      delete errors.fleetCodeId;
      delete errors.managerId;
    }
    return Object.keys(errors).length > 0 ? { values: {}, errors } : result;
  };
}

/** The input the write takes, carrying only the fields the chosen kind uses. */
export function recordEventInput(
  workerId: string,
  values: EmploymentEventFormValues,
): RecordWorkerEmploymentEventInput {
  const kind = values.kind;
  return {
    workerId,
    kind,
    effectiveAt: values.effectiveAt,
    reason: values.reason?.trim() || null,
    notes: values.notes?.trim() || null,
    documentId: null,
    fleetCodeId: kind === "Transferred" ? (values.fleetCodeId ?? undefined) : undefined,
    managerId: kind === "Transferred" ? (values.managerId ?? undefined) : undefined,
    driverType: kind === "Promoted" ? (values.driverType ?? undefined) : undefined,
    workerType: kind === "Promoted" ? (values.workerType ?? undefined) : undefined,
    rate: kind === "RateChanged" ? values.rate?.trim() || undefined : undefined,
    rateUnit: kind === "RateChanged" ? values.rateUnit?.trim() || undefined : undefined,
    leaveType: kind === "LeaveStarted" ? (values.leaveType ?? undefined) : undefined,
  };
}

/**
 * Whether the kind's own fields say enough to plan the event; until they do, the preview
 * waits rather than reading a plan the server would refuse.
 */
export function previewReady(values: Partial<EmploymentEventFormValues>): boolean {
  if (!values.kind || !values.effectiveAt) return false;
  switch (values.kind) {
    case "Transferred":
      return Boolean(values.fleetCodeId || values.managerId);
    case "Promoted":
      return Boolean(values.driverType || values.workerType);
    case "RateChanged":
      return Boolean(values.rate?.trim());
    case "LeaveStarted":
      return Boolean(values.leaveType);
    default:
      return true;
  }
}

/** What the person is told after a save: the effects carried out, out of sight of the history. */
export function announceRecorded(
  kind: EmploymentEventKind,
  result: RecordEmploymentEventResult,
  t: TranslateFn,
) {
  const carried = result.effects.filter((effect) => !effect.failed);
  if (carried.length > 0) {
    toast.success(t("{0} recorded", employmentEventLabel(kind)), {
      description: carried
        .map((effect) => describeEmploymentEffect(effect, t, "did").text)
        .join(" "),
    });
  }
  for (const effect of failedEffects(result.effects)) {
    toast.warning(describeEmploymentEffect(effect, t, "did").text, {
      description: t("The event stands; this part needs doing by hand."),
    });
  }
}

function currentDriverType(value: string): string {
  const choice = driverTypeChoices.find((option) => option.value === value);
  return choice ? translate("Currently {0}.", translate(choice.label)) : "";
}

function currentWorkerType(value: string): string {
  const choice = workerTypeChoices.find((option) => option.value === value);
  return choice ? translate("Currently {0}.", translate(choice.label)) : "";
}

export type RecordEventPaneProps = {
  open: boolean;
  /** `kind` from the address: presets the event when the worker may record it. */
  presetKind: string | null;
  workerId: string;
  worker: WorkerRecord;
  onClose: () => void;
};

/**
 * Record an employment event, beside the history it adds to. The kinds on offer and which of
 * them need a reason come from the server's own rule; "What this will do" is read from the
 * plan the write runs, before anything is saved.
 */
export function RecordEventPane({
  open,
  presetKind,
  workerId,
  worker,
  onClose,
}: RecordEventPaneProps) {
  const t = useT();
  const optionsQuery = useQuery({ ...employmentOptionsQuery(workerId), enabled: open });
  const options: EmploymentEventOptions | undefined = optionsQuery.data;
  const reasonRequired = new Set<string>(options?.reasonRequiredKinds ?? []);
  const preset =
    presetKind && isEmploymentEventKind(presetKind) && options?.recordableKinds.includes(presetKind)
      ? presetKind
      : null;

  return (
    <WorkPane
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={t("Record employment event")}
      description={t("Moves {0}'s employment on and writes it to the history.", worker.wholeName)}
    >
      {options ? (
        <RecordEventForm
          preset={preset}
          workerId={workerId}
          worker={worker}
          options={options}
          reasonRequired={reasonRequired}
          onDone={onClose}
        />
      ) : (
        <>
          <WorkPaneBody>
            {optionsQuery.isError ? (
              <ErrorState
                layout="compact"
                error={optionsQuery.error}
                title={t("The events that can be recorded could not be loaded")}
                onRetry={() => void optionsQuery.refetch()}
              />
            ) : (
              <div
                aria-busy
                aria-label={t("Loading the event form")}
                className="flex flex-col gap-4"
              >
                {[0, 1, 2, 3].map((row) => (
                  <div key={row} aria-hidden className="flex flex-col gap-1.5">
                    <span className="block h-3 w-20 ui-shimmer rounded-sm" />
                    <span className="block h-7 w-full ui-shimmer rounded-md" />
                  </div>
                ))}
              </div>
            )}
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="button" size="sm" disabled>
              {t("Record event")}
            </Button>
          </WorkPaneFooter>
        </>
      )}
    </WorkPane>
  );
}

function RecordEventForm({
  preset,
  workerId,
  worker,
  options,
  reasonRequired,
  onDone,
}: {
  preset: EmploymentEventKind | null;
  workerId: string;
  worker: WorkerRecord;
  options: EmploymentEventOptions;
  reasonRequired: ReadonlySet<string>;
  onDone: () => void;
}) {
  const t = useT();
  const invalidate = useWorkerInvalidation(workerId, "employment");
  const form = useForm<EmploymentEventFormValues>({
    resolver: employmentEventResolver(reasonRequired),
    defaultValues: emptyValues(preset),
  });
  const { control, handleSubmit } = form;

  const values = useWatch({ control }) as EmploymentEventFormValues;
  const kind = values.kind as EmploymentEventKind | "";
  const requiresReason = kind !== "" && reasonRequired.has(kind);
  const ready = previewReady(values);

  const save = useApiMutation<
    RecordEmploymentEventResult,
    EmploymentEventFormValues,
    unknown,
    EmploymentEventFormValues
  >({
    form,
    resourceName: "Employment event",
    scope: { id: `worker-employment-event:${workerId}` },
    mutationFn: (submitted) => recordWorkerEmploymentEvent(recordEventInput(workerId, submitted)),
    onSuccess: async (result, submitted) => {
      announceRecorded(submitted.kind, result, t);
      await invalidate();
      onDone();
    },
  });
  const submit = useSingleFlight((values: EmploymentEventFormValues) => save.mutateAsync(values));

  const kindOptions = options.recordableKinds.map((value) => ({
    value,
    label: employmentEventLabel(value),
  }));
  const leaveOptions = LEAVE_TYPES.map((value) => ({
    value,
    label: WORKER_LEAVE_TYPE_LABELS[value],
  }));
  const terminating = kind === "Terminated";

  return (
    <FormProvider {...form}>
      <WorkPaneDirtyTracker form={form} />
      <Form
        onSubmit={(event) => {
          event.preventDefault();
          event.stopPropagation();
          void handleSubmit((submitted) => submit.run(submitted))(event);
        }}
      >
        <WorkPaneBody>
          <FormGroup cols={1}>
            <FormControl>
              <SelectField<EmploymentEventFormValues>
                control={control}
                name="kind"
                label={t("Event")}
                placeholder={t("Choose an event")}
                options={kindOptions}
                required
                clearable={false}
                description={
                  kind === ""
                    ? t("The events {0} can move through now.", worker.wholeName)
                    : employmentEventHint(kind)
                }
              />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField<EmploymentEventFormValues>
                control={control}
                name="effectiveAt"
                label={t("Effective")}
                required
                placeholder={t("Today")}
              />
            </FormControl>
          </FormGroup>

          {kind === "Transferred" ? (
            <div key="Transferred" className="animate-expand-in motion-reduce:animate-none">
              <FormGroup cols={2}>
                <FormControl>
                  <FleetCodeAutocompleteField<EmploymentEventFormValues>
                    control={control}
                    name="fleetCodeId"
                    label={t("Fleet code")}
                    placeholder={t("Search fleet codes")}
                    clearable
                    description={
                      worker.fleetCode?.code
                        ? t("Currently {0}.", worker.fleetCode.code)
                        : t("Not on a fleet now.")
                    }
                  />
                </FormControl>
                <FormControl>
                  <UserAutocompleteField<EmploymentEventFormValues>
                    control={control}
                    name="managerId"
                    label={t("Manager")}
                    placeholder={t("Search people")}
                    clearable
                    description={
                      worker.manager?.name
                        ? t("Currently {0}.", worker.manager.name)
                        : t("No manager now.")
                    }
                  />
                </FormControl>
              </FormGroup>
            </div>
          ) : null}
          {kind === "LeaveStarted" ? (
            <div key="LeaveStarted" className="animate-expand-in motion-reduce:animate-none">
              <SelectField<EmploymentEventFormValues>
                control={control}
                name="leaveType"
                label={t("Leave type")}
                options={leaveOptions}
                required
                placeholder={t("Choose a leave type")}
              />
            </div>
          ) : null}
          {kind === "Promoted" ? (
            <div key="Promoted" className="animate-expand-in motion-reduce:animate-none">
              <FormGroup cols={2}>
                <FormControl>
                  <SelectField<EmploymentEventFormValues>
                    control={control}
                    name="driverType"
                    label={t("Driver type")}
                    options={driverTypeChoices}
                    clearable
                    placeholder={t("Unchanged")}
                    description={currentDriverType(worker.driverType)}
                  />
                </FormControl>
                <FormControl>
                  <SelectField<EmploymentEventFormValues>
                    control={control}
                    name="workerType"
                    label={t("Worker type")}
                    options={workerTypeChoices}
                    clearable
                    placeholder={t("Unchanged")}
                    description={currentWorkerType(worker.type)}
                  />
                </FormControl>
              </FormGroup>
            </div>
          ) : null}
          {kind === "RateChanged" ? (
            <div key="RateChanged" className="animate-expand-in motion-reduce:animate-none">
              <FormGroup cols={2}>
                <FormControl>
                  <InputField<EmploymentEventFormValues>
                    control={control}
                    name="rate"
                    label={t("New rate")}
                    placeholder={t("e.g. 0.64")}
                    required
                    description={t("Recorded on the event as typed.")}
                  />
                </FormControl>
                <FormControl>
                  <InputField<EmploymentEventFormValues>
                    control={control}
                    name="rateUnit"
                    label={t("Unit")}
                    placeholder={t("e.g. a loaded mile")}
                    description={t("What the rate is per.")}
                  />
                </FormControl>
              </FormGroup>
            </div>
          ) : null}

          <FormGroup cols={1}>
            <FormControl>
              <InputField<EmploymentEventFormValues>
                control={control}
                name="reason"
                label={t("Reason")}
                placeholder={t("What happened, in a line")}
                required={requiresReason}
                maxLength={255}
              />
            </FormControl>
            <FormControl>
              <TextareaField<EmploymentEventFormValues>
                control={control}
                name="notes"
                label={t("Notes")}
                placeholder={t("Anything the next person should know")}
                maxLength={4000}
              />
            </FormControl>
          </FormGroup>

          <EventPreview
            input={ready ? recordEventInput(workerId, values) : null}
            waiting={
              kind === ""
                ? t("Choose an event to see what it changes.")
                : t("Fill in the event's details to see what it changes.")
            }
            workerName={worker.wholeName}
          />
        </WorkPaneBody>
        <WorkPaneFooter>
          <WorkPaneCancel />
          <Button
            type="submit"
            size="sm"
            variant={terminating ? "destructive" : "default"}
            isLoading={submit.pending}
          >
            {terminating ? t("End employment") : t("Record event")}
          </Button>
        </WorkPaneFooter>
      </Form>
    </FormProvider>
  );
}
