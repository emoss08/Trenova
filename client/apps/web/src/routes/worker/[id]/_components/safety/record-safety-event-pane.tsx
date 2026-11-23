import { FileDropField, type AttachedFile } from "@/components/documents/file-drop-field";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  PaneFormLoading,
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  createWorkerSafetyEvent,
  updateWorkerSafetyEvent,
  type WorkerSafetyEventRow,
} from "@/lib/graphql/worker-safety";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { sourceLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  INSPECTION_RESULT_LABELS,
  SAFETY_EVENT_KIND_LABELS,
  SAFETY_SEVERITY_LABELS,
  inspectionResultSchema,
  safetyEventFormSchema,
  safetyEventKindSchema,
  safetySeveritySchema,
  type InspectionResult,
  type SafetyEventFormValues,
  type SafetyEventKind,
  type SafetySeverity,
} from "@trenova/shared/types/worker-safety";
import { useEffect, useState } from "react";
import { Controller, FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { useWorkerInvalidation } from "../use-worker-invalidation";
import { defaultSafetyPointsQuery, type SafetyPointsFacts } from "./safety-reads";

const EVENT_KIND_OPTIONS = safetyEventKindSchema.options.map((value) => ({
  value,
  label: sourceLabels(SAFETY_EVENT_KIND_LABELS)[value],
}));
const SEVERITY_OPTIONS = safetySeveritySchema.options.map((value) => ({
  value,
  label: sourceLabels(SAFETY_SEVERITY_LABELS)[value],
}));
const RESULT_OPTIONS = inspectionResultSchema.options.map((value) => ({
  value,
  label: sourceLabels(INSPECTION_RESULT_LABELS)[value],
}));
const LEVEL_OPTIONS = [
  { value: "1", label: "Level 1" },
  { value: "2", label: "Level 2" },
  { value: "3", label: "Level 3" },
  { value: "4", label: "Level 4" },
  { value: "5", label: "Level 5" },
  { value: "6", label: "Level 6" },
];

/** The form's values: the schema's, and the document the event is filed with. */
export type SafetyEventPaneValues = SafetyEventFormValues & { document: AttachedFile | null };

/** The schema checks the event; the document rides along beside it, outside the schema. */
const resolveSafetyEvent: Resolver<SafetyEventPaneValues> = async (values, context, options) => {
  const base = zodResolver(safetyEventFormSchema) as unknown as Resolver<SafetyEventFormValues>;
  const result = await base(values, context, options as never);
  if (Object.keys(result.errors).length > 0) {
    return { values: {}, errors: result.errors } as Awaited<
      ReturnType<Resolver<SafetyEventPaneValues>>
    >;
  }
  return {
    values: { ...(result.values as SafetyEventFormValues), document: values.document },
    errors: {},
  };
};

/** What a new event starts as, so its first points suggestion can be read before the form. */
const NEW_EVENT_FACTS: SafetyPointsFacts = {
  kind: "Incident",
  severity: "Minor",
  preventable: false,
  inspectionResult: null,
};

function factsOf(values: Pick<SafetyEventPaneValues, keyof SafetyPointsFacts>): SafetyPointsFacts {
  return {
    kind: values.kind,
    severity: values.severity,
    preventable: values.preventable,
    inspectionResult: values.inspectionResult,
  };
}

function sameFacts(a: SafetyPointsFacts, b: SafetyPointsFacts): boolean {
  return (
    a.kind === b.kind &&
    a.severity === b.severity &&
    a.preventable === b.preventable &&
    a.inspectionResult === b.inspectionResult
  );
}

/** The form's values at mount: the event's, or a new one's with the house scale's points. */
export function safetyEventDefaults(
  event: WorkerSafetyEventRow | null,
  suggestedPoints: number | null = null,
): SafetyEventPaneValues {
  if (!event) {
    return {
      kind: NEW_EVENT_FACTS.kind,
      severity: NEW_EVENT_FACTS.severity,
      occurredAt: getTodayDate(),
      location: null,
      description: "",
      preventable: NEW_EVENT_FACTS.preventable,
      points: suggestedPoints ?? 1,
      referenceNumber: null,
      shipmentId: null,
      inspectionLevel: null,
      inspectionResult: null,
      fineAmount: null,
      costAmount: null,
      document: null,
    };
  }
  return {
    kind: event.kind as SafetyEventKind,
    severity: event.severity as SafetySeverity,
    occurredAt: event.occurredAt,
    location: event.location ?? null,
    description: event.description,
    preventable: event.preventable,
    points: event.points,
    referenceNumber: event.referenceNumber ?? null,
    shipmentId: event.shipmentId ?? null,
    inspectionLevel: event.inspectionLevel ?? null,
    inspectionResult: (event.inspectionResult ?? null) as InspectionResult | null,
    fineAmount: event.fineAmount ?? null,
    costAmount: event.costAmount ?? null,
    document: event.document
      ? {
          id: event.document.id,
          name: event.document.originalName || event.document.fileName,
          size: event.document.fileSize ?? 0,
          fileType: event.document.fileType,
          previewStatus: event.document.previewStatus,
        }
      : null,
  };
}

export type RecordSafetyEventPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  /** The event being edited; null records a new one. */
  event: WorkerSafetyEventRow | null;
  onSaved: () => void;
};

/**
 * Records a safety event, or edits one, beside the events list. What happened decides the fields
 * that apply (an inspection's outcome and level, an accident's preventability) and where the
 * points suggestion starts; once somebody types points of their own the suggestion stops. A new
 * event's form mounts once the first suggestion is read, so it starts on it.
 */
export function RecordSafetyEventPane(props: RecordSafetyEventPaneProps) {
  const t = useT();
  const { open, onOpenChange, event } = props;
  const isEdit = event !== null;
  const initial = useQuery({ ...defaultSafetyPointsQuery(NEW_EVENT_FACTS), enabled: !isEdit });
  const waiting = !isEdit && initial.isPending;

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={isEdit ? t("Edit safety event") : t("Record a safety event")}
      description={
        isEdit
          ? t("Changes to the points move the score at once.")
          : t("Adds it to the events and counts its points toward the score.")
      }
      contentKey={event?.id ?? "new"}
    >
      {waiting ? (
        <PaneFormLoading fields={6} />
      ) : (
        <SafetyEventForm {...props} suggestedPoints={initial.data ?? null} />
      )}
    </WorkPane>
  );
}

function SafetyEventForm({
  workerId,
  event,
  onSaved,
  suggestedPoints,
}: RecordSafetyEventPaneProps & { suggestedPoints: number | null }) {
  const t = useT();
  const queryClient = useQueryClient();
  const invalidate = useWorkerInvalidation(workerId, "safety");
  const isEdit = event !== null;
  const form = useForm<SafetyEventPaneValues>({
    resolver: resolveSafetyEvent,
    defaultValues: safetyEventDefaults(event, suggestedPoints),
  });
  const { control, handleSubmit, setValue } = form;

  // When what happened changes, the points follow the house scale until somebody types their
  // own. Driven by the change itself (a subscription, not a render), and a suggestion that comes
  // back after the facts moved on is dropped.
  useEffect(
    () =>
      form.subscribe({
        name: ["kind", "severity", "preventable", "inspectionResult"],
        formState: { values: true },
        callback: ({ values }) => {
          if (isEdit || form.getFieldState("points").isDirty) return;
          const facts = factsOf(values);
          queryClient
            .fetchQuery(defaultSafetyPointsQuery(facts))
            .then((points) => {
              if (form.getFieldState("points").isDirty) return;
              if (!sameFacts(factsOf(form.getValues()), facts)) return;
              setValue("points", points);
            })
            .catch(() => undefined);
        },
      }),
    [form, isEdit, queryClient, setValue],
  );

  const [kind, severity, preventable, inspectionResult] = useWatch({
    control,
    name: ["kind", "severity", "preventable", "inspectionResult"],
  });
  const isInspection = kind === "Inspection";
  const isAccident = kind === "Accident";
  const suggested = useQuery(
    defaultSafetyPointsQuery({ kind, severity, preventable, inspectionResult }),
  );

  // Only an inspection carries an outcome and a level: changing what happened away from one
  // drops them, as the change, not after it.
  const changeKind = (value: string) => {
    if (value === "Inspection") return;
    setValue("inspectionResult", null);
    setValue("inspectionLevel", null);
  };

  const [uploading, setUploading] = useState(false);
  const { mutateAsync, isPending } = useApiMutation<
    WorkerSafetyEventRow,
    SafetyEventPaneValues,
    unknown,
    SafetyEventPaneValues
  >({
    form,
    resourceName: "Safety event",
    mutationFn: (values) => {
      const shared = {
        kind: values.kind,
        severity: values.severity,
        occurredAt: values.occurredAt,
        location: values.location ?? undefined,
        description: values.description,
        preventable: values.preventable,
        referenceNumber: values.referenceNumber ?? undefined,
        shipmentId: values.shipmentId ?? undefined,
        inspectionLevel: values.inspectionLevel ?? undefined,
        inspectionResult: values.inspectionResult ?? undefined,
        outOfService: values.inspectionResult === "OutOfService",
        fineAmount: values.fineAmount ?? undefined,
        costAmount: values.costAmount ?? undefined,
        documentId: values.document?.id ?? undefined,
      };
      return event
        ? updateWorkerSafetyEvent({
            ...shared,
            id: event.id,
            points: values.points,
            version: event.version,
          })
        : createWorkerSafetyEvent({ ...shared, workerId, points: values.points });
    },
    onSuccess: async () => {
      await invalidate();
      onSaved();
    },
  });

  return (
    <>
      <WorkPaneDirtyTracker form={form} />
      <FormProvider {...form}>
        <Form
          onSubmit={(submitEvent) => {
            submitEvent.preventDefault();
            submitEvent.stopPropagation();
            void handleSubmit((values) => mutateAsync(values))(submitEvent);
          }}
        >
          <WorkPaneBody className="text-sm">
            <FormGroup cols={2}>
              <FormControl>
                <SelectField<SafetyEventPaneValues>
                  control={control}
                  name="kind"
                  label={t("What happened")}
                  options={EVENT_KIND_OPTIONS}
                  placeholder={t("Pick an event type")}
                  onValueChange={changeKind}
                  required
                />
              </FormControl>
              <FormControl>
                <SelectField<SafetyEventPaneValues>
                  control={control}
                  name="severity"
                  label={t("Severity")}
                  options={SEVERITY_OPTIONS}
                  placeholder={t("Pick a severity")}
                  required
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<SafetyEventPaneValues>
                  control={control}
                  name="occurredAt"
                  label={t("When")}
                  placeholder={t("MM/DD/YYYY")}
                  required
                />
              </FormControl>
              <FormControl>
                <InputField<SafetyEventPaneValues>
                  control={control}
                  name="location"
                  label={t("Where")}
                  placeholder={t("e.g. I-80 WB, Joliet IL")}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<SafetyEventPaneValues>
                  control={control}
                  name="description"
                  label={t("Description")}
                  placeholder={t("What happened, as the report or the driver tells it")}
                  required
                  maxLength={4000}
                />
              </FormControl>
              {isInspection ? (
                <>
                  <FormControl className="motion-safe:animate-expand-in">
                    <SelectField<SafetyEventPaneValues>
                      control={control}
                      name="inspectionResult"
                      label={t("Outcome")}
                      options={RESULT_OPTIONS}
                      placeholder={t("Pick an outcome")}
                      required
                    />
                  </FormControl>
                  <FormControl className="motion-safe:animate-expand-in">
                    <SelectField<SafetyEventPaneValues>
                      control={control}
                      name="inspectionLevel"
                      label={t("Level")}
                      options={LEVEL_OPTIONS}
                      placeholder={t("Pick a level")}
                      clearable
                    />
                  </FormControl>
                </>
              ) : null}
              {isAccident ? (
                <FormControl cols="full" className="motion-safe:animate-expand-in">
                  <SwitchField<SafetyEventPaneValues>
                    control={control}
                    name="preventable"
                    label={t("Preventable")}
                    description={t(
                      "The driver could reasonably have avoided it. It costs more on the scorecard.",
                    )}
                    position="left"
                    outlined
                  />
                </FormControl>
              ) : null}
              <FormControl>
                <NumberField<SafetyEventPaneValues>
                  control={control}
                  name="points"
                  label={t("Points")}
                  placeholder="0"
                  min={0}
                  description={
                    suggested.data == null
                      ? isInspection && !inspectionResult
                        ? t("Pick the outcome to see the suggestion.")
                        : t("Points count for two years.")
                      : t("Suggested: {0}. Points roll off after two years.", suggested.data)
                  }
                />
              </FormControl>
              <FormControl>
                <InputField<SafetyEventPaneValues>
                  control={control}
                  name="referenceNumber"
                  label={t("Reference")}
                  placeholder={t("Citation or report number")}
                />
              </FormControl>
              <FormControl>
                <InputField<SafetyEventPaneValues>
                  control={control}
                  name="fineAmount"
                  label={t("Fine")}
                  placeholder="0.00"
                  sideText="$"
                />
              </FormControl>
              <FormControl>
                <InputField<SafetyEventPaneValues>
                  control={control}
                  name="costAmount"
                  label={t("Cost to the carrier")}
                  placeholder="0.00"
                  sideText="$"
                />
              </FormControl>
            </FormGroup>
            <Controller
              control={control}
              name="document"
              render={({ field }) => (
                <FileDropField
                  label={t("Document")}
                  value={field.value}
                  onChange={field.onChange}
                  resourceId={workerId}
                  resourceType="worker"
                  prompt={t("Drop the report or photos here")}
                  acceptLabel={t("PDF, JPG, PNG or WEBP.")}
                  disabled={isPending}
                  onUploadingChange={setUploading}
                />
              )}
            />
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending} disabled={uploading}>
              {isEdit ? t("Save changes") : t("Record event")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </>
  );
}
