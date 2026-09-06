import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  createWorkerSafetyEvent,
  fetchDefaultSafetyPoints,
  updateWorkerSafetyEvent,
  type WorkerSafetyEventRow,
} from "@/lib/graphql/worker-safety";
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
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { FormProvider, useForm, useFormState, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useSafetyInvalidation } from "./use-safety-invalidation";

const KIND_OPTIONS = safetyEventKindSchema.options.map((value) => ({
  value,
  label: SAFETY_EVENT_KIND_LABELS[value],
}));
const SEVERITY_OPTIONS = safetySeveritySchema.options.map((value) => ({
  value,
  label: SAFETY_SEVERITY_LABELS[value],
}));
const RESULT_OPTIONS = inspectionResultSchema.options.map((value) => ({
  value,
  label: INSPECTION_RESULT_LABELS[value],
}));
const LEVEL_OPTIONS = [1, 2, 3, 4, 5, 6].map((level) => ({
  value: String(level),
  label: `Level ${level}`,
}));

export type SafetyEventDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  event?: WorkerSafetyEventRow | null;
};

function defaultsFor(event?: WorkerSafetyEventRow | null): SafetyEventFormValues {
  if (!event) {
    return {
      kind: "Incident",
      severity: "Minor",
      occurredAt: getTodayDate(),
      location: null,
      description: "",
      preventable: false,
      points: 1,
      referenceNumber: null,
      shipmentId: null,
      inspectionLevel: null,
      inspectionResult: null,
      fineAmount: null,
      costAmount: null,
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
  };
}

export function SafetyEventDialog({ open, onOpenChange, workerId, event }: SafetyEventDialogProps) {
  const invalidate = useSafetyInvalidation(workerId);
  const isEdit = Boolean(event);
  const form = useForm<SafetyEventFormValues>({
    resolver: zodResolver(safetyEventFormSchema) as Resolver<SafetyEventFormValues>,
    defaultValues: defaultsFor(event),
  });
  const { control, handleSubmit, reset, setValue } = form;
  // Once someone types a number of their own, the suggestion stops moving it.
  const { dirtyFields } = useFormState({ control, name: "points" });
  const pointsTouched = isEdit || Boolean(dirtyFields.points);

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(event));
  }, [open, event, reset]);

  const [kind, severity, preventable, inspectionResult] = useWatch({
    control,
    name: ["kind", "severity", "preventable", "inspectionResult"],
  });
  const isInspection = kind === "Inspection";
  const isAccident = kind === "Accident";

  // Only an inspection's outcome changes the suggestion, so the key carries
  // the value the query actually asks with.
  const effectiveResult = isInspection ? inspectionResult : null;
  const suggested = useQuery({
    queryKey: ["default-safety-points", kind, severity, preventable, effectiveResult],
    queryFn: ({ signal }) =>
      fetchDefaultSafetyPoints(
        {
          kind,
          severity,
          preventable,
          inspectionResult: effectiveResult,
        },
        { signal },
      ),
    enabled: open,
    staleTime: 60 * 60 * 1000,
  });

  useEffect(() => {
    if (pointsTouched || suggested.data == null) return;
    setValue("points", suggested.data);
  }, [pointsTouched, suggested.data, setValue]);

  useEffect(() => {
    if (isInspection) return;
    setValue("inspectionResult", null);
    setValue("inspectionLevel", null);
  }, [isInspection, setValue]);

  const { mutateAsync, isPending } = useApiMutation<
    WorkerSafetyEventRow,
    SafetyEventFormValues,
    unknown,
    SafetyEventFormValues
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
    onSuccess: (saved) => {
      toast.success(isEdit ? "Safety event updated" : "Safety event recorded", {
        description:
          saved.activePoints > 0
            ? `${saved.activePoints} point${saved.activePoints === 1 ? "" : "s"} added to the scorecard.`
            : "No points added to the scorecard.",
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  const description = useMemo(() => {
    if (isInspection) {
      return "Roadside inspections carry points only when they fail or put the driver out of service.";
    }
    if (isAccident) {
      return "Preventable accidents cost more on the scorecard than non-preventable ones.";
    }
    return "Points come off the scorecard automatically two years after the event.";
  }, [isAccident, isInspection]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit safety event" : "Record a safety event"}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup className="pb-2" cols={2}>
              <FormControl>
                <SelectField<SafetyEventFormValues>
                  control={control}
                  name="kind"
                  label="What happened"
                  options={KIND_OPTIONS}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SelectField<SafetyEventFormValues>
                  control={control}
                  name="severity"
                  label="Severity"
                  options={SEVERITY_OPTIONS}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<SafetyEventFormValues>
                  control={control}
                  name="occurredAt"
                  label="When"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <InputField<SafetyEventFormValues>
                  control={control}
                  name="location"
                  label="Where"
                  placeholder="e.g. I-80 WB, Joliet IL"
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<SafetyEventFormValues>
                  control={control}
                  name="description"
                  label="What happened"
                  placeholder="Plain description of the event, as it would read in a file review"
                  rules={{ required: true }}
                  maxLength={4000}
                />
              </FormControl>

              {isInspection ? (
                <>
                  <FormControl>
                    <SelectField<SafetyEventFormValues>
                      control={control}
                      name="inspectionResult"
                      label="Outcome"
                      options={RESULT_OPTIONS}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <SelectField<SafetyEventFormValues>
                      control={control}
                      name="inspectionLevel"
                      label="Level"
                      options={LEVEL_OPTIONS}
                      isClearable
                    />
                  </FormControl>
                </>
              ) : null}

              {isAccident ? (
                <FormControl cols="full">
                  <SwitchField<SafetyEventFormValues>
                    control={control}
                    name="preventable"
                    label="Preventable"
                    description="Could the driver reasonably have avoided it? This drives the scorecard penalty."
                    position="left"
                    outlined
                  />
                </FormControl>
              ) : null}

              <FormControl>
                <NumberField<SafetyEventFormValues>
                  control={control}
                  name="points"
                  label="Points"
                  min={0}
                  description={
                    suggested.data == null
                      ? "Points count for two years."
                      : `Suggested ${suggested.data} for this kind and severity.`
                  }
                />
              </FormControl>
              <FormControl>
                <InputField<SafetyEventFormValues>
                  control={control}
                  name="referenceNumber"
                  label="Reference"
                  placeholder="Citation or report number"
                />
              </FormControl>
              <FormControl>
                <InputField<SafetyEventFormValues>
                  control={control}
                  name="fineAmount"
                  label="Fine"
                  placeholder="0.00"
                  sideText="$"
                />
              </FormControl>
              <FormControl>
                <InputField<SafetyEventFormValues>
                  control={control}
                  name="costAmount"
                  label="Cost to the carrier"
                  placeholder="0.00"
                  sideText="$"
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending} loadingText="Saving...">
                {isEdit ? "Save changes" : "Record event"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
