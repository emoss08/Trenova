import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  OSHA_LOG_KEY,
  recordWorkerInjury,
  updateWorkerInjury,
  WORKER_INJURIES_KEY,
  type WorkerInjuryRow,
} from "@/lib/graphql/worker-injury";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon, InfoCircleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  caseClassificationLabel,
  claimStatusLabel,
  classificationIsRecordable,
  illnessTypeLabel,
  injuryTreatmentLabel,
  suggestClassification,
} from "@trenova/shared/lib/injury";
import {
  injuryCaseStatusSchema,
  injuryFormSchema,
  injuryTreatmentSchema,
  oshaCaseClassificationSchema,
  oshaIllnessTypeSchema,
  workersCompClaimStatusSchema,
  type InjuryFormValues,
} from "@trenova/shared/types/worker-injury";
import { useEffect } from "react";
import { useForm, useWatch, type Resolver, type UseFormReturn } from "react-hook-form";

const STATUS_LABELS = defineLabels({
  Open: "Open, days may still accrue",
  Closed: "Closed",
});

/** What both mutations return: enough to say which line of the log this became. */
type SavedCase = {
  id: string;
  caseNumber: number;
  caseYear: number;
  recordable: boolean;
};

export function injuryDefaults(
  injury: WorkerInjuryRow | null,
  safetyEventId: string | null,
): InjuryFormValues {
  if (!injury) {
    return {
      occurredAt: getTodayDate(),
      reportedAt: getTodayDate(),
      returnedToWorkAt: null,
      description: "",
      location: null,
      bodyPart: null,
      harmfulAgent: null,
      classification: "NotRecordable",
      illnessType: "Injury",
      treatment: "None",
      status: "Open",
      daysAway: 0,
      daysRestricted: 0,
      privacyCase: false,
      claimStatus: "NotFiled",
      claimNumber: null,
      claimCarrier: null,
      claimFiledAt: null,
      claimClosedAt: null,
      safetyEventId,
      notes: null,
    };
  }
  return {
    occurredAt: injury.occurredAt,
    reportedAt: injury.reportedAt ?? null,
    returnedToWorkAt: injury.returnedToWorkAt ?? null,
    description: injury.description,
    location: injury.location ?? null,
    bodyPart: injury.bodyPart ?? null,
    harmfulAgent: injury.harmfulAgent ?? null,
    classification: injury.classification,
    illnessType: injury.illnessType,
    treatment: injury.treatment,
    status: injury.status,
    daysAway: injury.daysAway,
    daysRestricted: injury.daysRestricted,
    privacyCase: injury.privacyCase,
    claimStatus: injury.claimStatus,
    claimNumber: injury.claimNumber ?? null,
    claimCarrier: injury.claimCarrier ?? null,
    claimFiledAt: injury.claimFiledAt ?? null,
    claimClosedAt: injury.claimClosedAt ?? null,
    safetyEventId: injury.safetyEventId ?? null,
    notes: injury.notes ?? null,
  };
}

export type InjuryFormOptions = {
  workerId: string;
  injury: WorkerInjuryRow | null;
  /** Set when the case is opened from a safety event. */
  safetyEventId?: string | null;
  onSaved: () => void;
};

export type InjuryFormController = {
  form: UseFormReturn<InjuryFormValues>;
  isEdit: boolean;
  isPending: boolean;
  submit: (event: React.FormEvent<HTMLFormElement>) => void;
};

/**
 * The injury and illness case form behind both its hosts, the OSHA log's dialog and the worker
 * page's work pane: its schema, the classification suggestion that follows the facts until
 * somebody makes the judgement, and the record or update mutation.
 *
 * Both hosts mount it fresh at each opening (the pane is mounted per opening, the dialog's form
 * lives inside its content), so its values are the case's at mount and a refetch of the case
 * while it is open never replaces what is typed.
 */
export function useInjuryForm({
  workerId,
  injury,
  safetyEventId = null,
  onSaved,
}: InjuryFormOptions): InjuryFormController {
  const queryClient = useQueryClient();
  const isEdit = injury !== null;
  const form = useForm<InjuryFormValues>({
    resolver: zodResolver(injuryFormSchema) as Resolver<InjuryFormValues>,
    defaultValues: injuryDefaults(injury, safetyEventId),
  });
  const { handleSubmit, setValue } = form;

  // Until somebody makes the judgement, the classification follows the treatment and the days
  // as they change: driven by the change (a subscription), not by a render.
  useEffect(
    () =>
      form.subscribe({
        name: ["treatment", "daysAway", "daysRestricted"],
        formState: { values: true },
        callback: ({ values }) => {
          if (isEdit || form.getFieldState("classification").isDirty) return;
          setValue(
            "classification",
            suggestClassification(
              values.treatment,
              values.daysAway,
              values.daysRestricted,
            ) as InjuryFormValues["classification"],
          );
        },
      }),
    [form, isEdit, setValue],
  );

  const { mutateAsync, isPending } = useApiMutation<
    SavedCase,
    InjuryFormValues,
    unknown,
    InjuryFormValues
  >({
    form,
    resourceName: "Case",
    mutationFn: (values) => {
      const shared = {
        occurredAt: values.occurredAt,
        reportedAt: values.reportedAt ?? undefined,
        returnedToWorkAt: values.returnedToWorkAt ?? undefined,
        description: values.description,
        location: values.location ?? undefined,
        bodyPart: values.bodyPart ?? undefined,
        harmfulAgent: values.harmfulAgent ?? undefined,
        classification: values.classification,
        illnessType: values.illnessType,
        treatment: values.treatment,
        daysAway: values.daysAway,
        daysRestricted: values.daysRestricted,
        privacyCase: values.privacyCase,
        claimStatus: values.claimStatus,
        claimNumber: values.claimNumber ?? undefined,
        claimCarrier: values.claimCarrier ?? undefined,
        claimFiledAt: values.claimFiledAt ?? undefined,
        safetyEventId: values.safetyEventId ?? undefined,
        notes: values.notes ?? undefined,
      };
      return injury
        ? updateWorkerInjury({
            ...shared,
            injuryId: injury.id,
            status: values.status,
            claimClosedAt: values.claimClosedAt ?? undefined,
          })
        : recordWorkerInjury({ ...shared, workerId });
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: [WORKER_INJURIES_KEY, workerId] }),
        queryClient.invalidateQueries({ queryKey: [OSHA_LOG_KEY] }),
      ]);
      onSaved();
    },
  });

  const submit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    event.stopPropagation();
    void handleSubmit((values) => mutateAsync(values))(event);
  };

  return { form, isEdit, isPending, submit };
}

/** The case's fields, in the order the 300 log and the 301 form ask for them. */
export function InjuryFormFields({
  form,
  isEdit,
}: {
  form: UseFormReturn<InjuryFormValues>;
  isEdit: boolean;
}) {
  const t = useT();
  const { control } = form;
  const [classification, claimStatus] = useWatch({
    control,
    name: ["classification", "claimStatus"],
  });
  const recordable = classificationIsRecordable(classification);
  const claimFiled = claimStatus !== "NotFiled";

  const classificationOptions = oshaCaseClassificationSchema.options.map((value) => ({
    value,
    label: caseClassificationLabel(value),
  }));
  const illnessOptions = oshaIllnessTypeSchema.options.map((value) => ({
    value,
    label: illnessTypeLabel(value),
  }));
  const treatmentOptions = injuryTreatmentSchema.options.map((value) => ({
    value,
    label: injuryTreatmentLabel(value),
  }));
  const statusOptions = injuryCaseStatusSchema.options.map((value) => ({
    value,
    label: STATUS_LABELS[value],
  }));
  const claimOptions = workersCompClaimStatusSchema.options.map((value) => ({
    value,
    label: claimStatusLabel(value),
  }));

  return (
    <FormGroup cols={2}>
      <FormControl>
        <AutoCompleteDateField<InjuryFormValues>
          control={control}
          name="occurredAt"
          label={t("When it happened")}
          placeholder={t("MM/DD/YYYY")}
          description={t("Sets the log year the case is numbered in.")}
          required
        />
      </FormControl>
      <FormControl>
        <AutoCompleteDateField<InjuryFormValues>
          control={control}
          name="reportedAt"
          label={t("When it was reported")}
          placeholder={t("MM/DD/YYYY")}
          description={t("When it was first reported to the company.")}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField<InjuryFormValues>
          control={control}
          name="description"
          label={t("What happened")}
          placeholder={t("How the injury occurred, as it would read on the 301 form")}
          required
          maxLength={4000}
        />
      </FormControl>
      <FormControl>
        <InputField<InjuryFormValues>
          control={control}
          name="location"
          label={t("Where")}
          placeholder={t("e.g. Dock 4, Joliet terminal")}
        />
      </FormControl>
      <FormControl>
        <InputField<InjuryFormValues>
          control={control}
          name="bodyPart"
          label={t("Body part")}
          placeholder={t("e.g. Lower back")}
        />
      </FormControl>
      <FormControl>
        <InputField<InjuryFormValues>
          control={control}
          name="harmfulAgent"
          label={t("What harmed them")}
          placeholder={t("Object or substance")}
        />
      </FormControl>
      <FormControl>
        <SelectField<InjuryFormValues>
          control={control}
          name="illnessType"
          label={t("Injury or illness")}
          options={illnessOptions}
          placeholder={t("Pick a type")}
          description={t("Which column of the 300 log it is tallied in.")}
          required
        />
      </FormControl>
      <FormControl>
        <SelectField<InjuryFormValues>
          control={control}
          name="treatment"
          label={t("Treatment given")}
          options={treatmentOptions}
          placeholder={t("Pick the treatment")}
          description={t("First aid alone does not make a case recordable.")}
          required
        />
      </FormControl>
      <FormControl>
        <NumberField<InjuryFormValues>
          control={control}
          name="daysAway"
          label={t("Days away from work")}
          placeholder="0"
        />
      </FormControl>
      <FormControl>
        <NumberField<InjuryFormValues>
          control={control}
          name="daysRestricted"
          label={t("Days on restriction or transfer")}
          placeholder="0"
        />
      </FormControl>
      <FormControl>
        <SelectField<InjuryFormValues>
          control={control}
          name="classification"
          label={t("OSHA classification")}
          options={classificationOptions}
          placeholder={t("Pick a classification")}
          description={t(
            "The employer's judgement; suggested from the treatment and days until you pick one.",
          )}
          required
        />
      </FormControl>
      {isEdit ? (
        <FormControl>
          <SelectField<InjuryFormValues>
            control={control}
            name="status"
            label={t("Case status")}
            options={statusOptions}
            placeholder={t("Pick a status")}
            required
          />
        </FormControl>
      ) : null}
      <FormControl cols="full">
        <Alert
          key={String(recordable)}
          size="sm"
          variant={recordable ? "warning" : "default"}
          className="motion-safe:animate-rise"
        >
          {recordable ? <AlertTriangleIcon /> : <InfoCircleIcon />}
          <AlertDescription>
            {recordable
              ? t(
                  "This case goes on the OSHA 300 log. It counts toward the year the injury happened.",
                )
              : t(
                  "This case is kept on file but is not on the 300 log. First aid alone is not recordable (29 CFR 1904.7(b)(5)(ii)).",
                )}
          </AlertDescription>
        </Alert>
      </FormControl>
      <FormControl>
        <AutoCompleteDateField<InjuryFormValues>
          control={control}
          name="returnedToWorkAt"
          label={t("Returned to work")}
          placeholder={t("MM/DD/YYYY")}
          description={t("When the employee came back to full duty.")}
        />
      </FormControl>
      <FormControl cols="full">
        <SwitchField<InjuryFormValues>
          control={control}
          name="privacyCase"
          label={t("Privacy concern case")}
          description={t(
            "The name is left off the posted 300 log and written as a privacy case (29 CFR 1904.29(b)(6)).",
          )}
          position="left"
          outlined
        />
      </FormControl>
      <FormControl cols="full">
        <SelectField<InjuryFormValues>
          control={control}
          name="claimStatus"
          label={t("Workers' compensation claim")}
          options={claimOptions}
          placeholder={t("Pick a status")}
          required
        />
      </FormControl>
      {claimFiled ? (
        <>
          <FormControl className="motion-safe:animate-expand-in">
            <AutoCompleteDateField<InjuryFormValues>
              control={control}
              name="claimFiledAt"
              label={t("Claim filed")}
              placeholder={t("MM/DD/YYYY")}
              required
            />
          </FormControl>
          <FormControl className="motion-safe:animate-expand-in">
            <InputField<InjuryFormValues>
              control={control}
              name="claimNumber"
              label={t("Claim number")}
              placeholder={t("e.g. WC-2026-001234")}
            />
          </FormControl>
          <FormControl cols="full" className="motion-safe:animate-expand-in">
            <InputField<InjuryFormValues>
              control={control}
              name="claimCarrier"
              label={t("Carrier")}
              placeholder={t("Insurer handling the claim")}
            />
          </FormControl>
          {claimStatus === "Closed" ? (
            <FormControl className="motion-safe:animate-expand-in">
              <AutoCompleteDateField<InjuryFormValues>
                control={control}
                name="claimClosedAt"
                label={t("Claim closed")}
                placeholder={t("MM/DD/YYYY")}
                required
              />
            </FormControl>
          ) : null}
        </>
      ) : null}
      <FormControl cols="full">
        <TextareaField<InjuryFormValues>
          control={control}
          name="notes"
          label={t("Notes")}
          placeholder={t("Anything else the case file should carry")}
          maxLength={2000}
        />
      </FormControl>
    </FormGroup>
  );
}
