import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  recordWorkerInjury,
  updateWorkerInjury,
  WORKER_INJURIES_KEY,
  type WorkerInjuryRow,
} from "@/lib/graphql/worker-injury";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
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
import { toast } from "sonner";
import { FormProvider, useForm, useFormState, useWatch, type Resolver } from "react-hook-form";

const CLASSIFICATION_OPTIONS = oshaCaseClassificationSchema.options.map((value) => ({
  value,
  label: caseClassificationLabel(value),
}));
const ILLNESS_OPTIONS = oshaIllnessTypeSchema.options.map((value) => ({
  value,
  label: illnessTypeLabel(value),
}));
const TREATMENT_OPTIONS = injuryTreatmentSchema.options.map((value) => ({
  value,
  label: injuryTreatmentLabel(value),
}));
const STATUS_OPTIONS = injuryCaseStatusSchema.options.map((value) => ({
  value,
  label: value === "Open" ? "Open — days may still accrue" : "Closed",
}));
const CLAIM_OPTIONS = workersCompClaimStatusSchema.options.map((value) => ({
  value,
  label: claimStatusLabel(value),
}));

/** What both mutations return: enough to say which line of the log this became. */
type SavedCase = {
  id: string;
  caseNumber: number;
  caseYear: number;
  recordable: boolean;
};

export type InjuryDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  injury: WorkerInjuryRow | null;
  /** Set when the case is being opened from a safety event. */
  safetyEventId?: string | null;
};

function defaultsFor(
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

export function InjuryDialog({
  open,
  onOpenChange,
  workerId,
  injury,
  safetyEventId = null,
}: InjuryDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const isEdit = Boolean(injury);
  const form = useForm<InjuryFormValues>({
    resolver: zodResolver(injuryFormSchema) as Resolver<InjuryFormValues>,
    defaultValues: defaultsFor(injury, safetyEventId),
  });
  const { control, handleSubmit, reset, setValue } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(injury, safetyEventId));
  }, [open, injury, safetyEventId, reset]);

  const [treatment, daysAway, daysRestricted, classification, claimStatus] = useWatch({
    control,
    name: ["treatment", "daysAway", "daysRestricted", "classification", "claimStatus"],
  });

  // Recordability is the employer's judgement. The suggestion follows the facts
  // until somebody makes that judgement themselves, and then stops moving.
  const { dirtyFields } = useFormState({ control, name: "classification" });
  const chosen = isEdit || Boolean(dirtyFields.classification);
  const suggested = suggestClassification(treatment, daysAway, daysRestricted);

  useEffect(() => {
    if (chosen) return;
    setValue("classification", suggested as InjuryFormValues["classification"]);
  }, [chosen, suggested, setValue]);

  const claimFiled = claimStatus !== "NotFiled";

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
    onSuccess: (saved) => {
      toast.success(isEdit ? "Case updated" : "Case recorded", {
        description: `Case ${saved.caseYear}-${saved.caseNumber} · ${
          saved.recordable ? "on the OSHA 300 log" : "not recordable"
        }.`,
      });
      void queryClient.invalidateQueries({ queryKey: [WORKER_INJURIES_KEY, workerId] });
      void queryClient.invalidateQueries({ queryKey: ["osha-log"] });
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit case" : "Record an injury or illness"}</DialogTitle>
          <DialogDescription>
            {t("Every case is kept, recordable or not — the decision not to record one is itself worth a record.")}
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
            <FormGroup className="pb-2" cols={2}>
              <FormControl>
                <AutoCompleteDateField<InjuryFormValues>
                  control={control}
                  name="occurredAt"
                  label={t("When it happened")}
                  placeholder={t("MM/DD/YYYY")}
                  description={t("Sets the log year the case is numbered in.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<InjuryFormValues>
                  control={control}
                  name="reportedAt"
                  label={t("When it was reported")}
                  placeholder={t("MM/DD/YYYY")}
                  description={t("When the injury was first reported to the company.")}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<InjuryFormValues>
                  control={control}
                  name="description"
                  label={t("What happened")}
                  placeholder={t("How the injury occurred, as it would read on the 301 form")}
                  description={t("The narrative for the 301 incident report: what the employee was doing and how it happened.")}
                  rules={{ required: true }}
                  maxLength={4000}
                />
              </FormControl>
              <FormControl>
                <InputField<InjuryFormValues>
                  control={control}
                  name="location"
                  label={t("Where")}
                  placeholder={t("e.g. Dock 4, Joliet terminal")}
                  description={t("Where the event occurred; the 300 log asks for it.")}
                />
              </FormControl>
              <FormControl>
                <InputField<InjuryFormValues>
                  control={control}
                  name="bodyPart"
                  label={t("Body part")}
                  placeholder={t("e.g. Lower back")}
                  description={t("The part of the body affected; it goes in the 300 log description.")}
                />
              </FormControl>
              <FormControl>
                <InputField<InjuryFormValues>
                  control={control}
                  name="harmfulAgent"
                  label={t("What harmed them")}
                  placeholder={t("Object or substance")}
                  description={t("The object or substance that directly caused the injury, as the 300 log asks.")}
                />
              </FormControl>
              <FormControl>
                <SelectField<InjuryFormValues>
                  control={control}
                  name="illnessType"
                  label={t("Injury or illness")}
                  options={ILLNESS_OPTIONS}
                  placeholder={t("Pick a type")}
                  description={t("Which column of the 300 log the case is tallied in.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SelectField<InjuryFormValues>
                  control={control}
                  name="treatment"
                  label={t("Treatment given")}
                  options={TREATMENT_OPTIONS}
                  placeholder={t("Pick the treatment")}
                  description={t("Feeds the classification suggestion; first aid alone does not make a case recordable.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <NumberField<InjuryFormValues>
                  control={control}
                  name="daysAway"
                  label={t("Days away from work")}
                  placeholder="0"
                  description={t("Calendar days away from work; drives the classification suggestion and the log totals.")}
                />
              </FormControl>
              <FormControl>
                <NumberField<InjuryFormValues>
                  control={control}
                  name="daysRestricted"
                  label={t("Days on restriction or transfer")}
                  placeholder="0"
                  description={t("Calendar days on restricted work or job transfer; counted in the DART rate.")}
                />
              </FormControl>
              <FormControl>
                <SelectField<InjuryFormValues>
                  control={control}
                  name="classification"
                  label={t("OSHA classification")}
                  options={CLASSIFICATION_OPTIONS}
                  placeholder={t("Pick a classification")}
                  description={t("The employer's judgement; suggested from treatment and days lost until you choose one yourself.")}
                  rules={{ required: true }}
                />
              </FormControl>
              {isEdit ? (
                <FormControl>
                  <SelectField<InjuryFormValues>
                    control={control}
                    name="status"
                    label={t("Case status")}
                    options={STATUS_OPTIONS}
                    placeholder={t("Pick a status")}
                    description={t("An open case can still accrue days away or restricted; close it once the count is final.")}
                    rules={{ required: true }}
                  />
                </FormControl>
              ) : null}

              <FormControl cols="full">
                <Alert variant={classificationIsRecordable(classification) ? "default" : undefined}>
                  <AlertDescription>
                    {classificationIsRecordable(classification)
                      ? "This case goes on the OSHA 300 log."
                      : "This case is kept on file but is not on the 300 log. First aid alone is not recordable (29 CFR 1904.7(b)(5)(ii))."}
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
              <FormControl>
                <SwitchField<InjuryFormValues>
                  control={control}
                  name="privacyCase"
                  label={t("Privacy concern case")}
                  description={t("The name is withheld from the posted log (29 CFR 1904.29(b)(6)).")}
                />
              </FormControl>

              <FormControl>
                <SelectField<InjuryFormValues>
                  control={control}
                  name="claimStatus"
                  label={t("Workers' compensation")}
                  options={CLAIM_OPTIONS}
                  placeholder={t("Pick a status")}
                  description={t("Tracks the workers' compensation claim; the claim dates are dropped while it is not filed.")}
                  rules={{ required: true }}
                />
              </FormControl>
              {claimFiled ? (
                <>
                  <FormControl>
                    <AutoCompleteDateField<InjuryFormValues>
                      control={control}
                      name="claimFiledAt"
                      label={t("Claim filed")}
                      placeholder={t("MM/DD/YYYY")}
                      description={t("The date the claim was submitted to the carrier.")}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <InputField<InjuryFormValues>
                      control={control}
                      name="claimNumber"
                      label={t("Claim number")}
                      placeholder={t("e.g. WC-2026-001234")}
                      description={t("The carrier's claim number, for matching correspondence.")}
                    />
                  </FormControl>
                  <FormControl>
                    <InputField<InjuryFormValues>
                      control={control}
                      name="claimCarrier"
                      label={t("Carrier")}
                      placeholder={t("Insurer handling the claim")}
                      description={t("The workers' compensation insurer handling the claim.")}
                    />
                  </FormControl>
                  {claimStatus === "Closed" ? (
                    <FormControl>
                      <AutoCompleteDateField<InjuryFormValues>
                        control={control}
                        name="claimClosedAt"
                        label={t("Claim closed")}
                        placeholder={t("MM/DD/YYYY")}
                        description={t("When the carrier closed the claim.")}
                        rules={{ required: true }}
                      />
                    </FormControl>
                  ) : null}
                </>
              ) : null}

              <FormControl cols="full">
                <TextareaField<InjuryFormValues>
                  control={control}
                  name="notes"
                  placeholder={t("Anything else the case file should carry")}
                  description={t("Internal notes kept with the case.")}
                  label={t("Notes")}
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending}>
                {isEdit ? "Save" : "Record"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
