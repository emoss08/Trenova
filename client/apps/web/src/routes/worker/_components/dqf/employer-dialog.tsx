import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  recordEmploymentVerification,
  updateEmploymentVerification,
  type EmploymentVerificationRow,
} from "@/lib/graphql/worker-dqf";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { verificationMethodLabel, verificationStatusLabel } from "@trenova/shared/lib/dqf";
import {
  employmentVerificationFormSchema,
  employmentVerificationMethodSchema,
  employmentVerificationStatusSchema,
  type EmploymentVerificationFormValues,
} from "@trenova/shared/types/worker-dqf";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useDqfInvalidation } from "./use-dqf-invalidation";

const STATUS_OPTIONS = employmentVerificationStatusSchema.options.map((value) => ({
  value,
  label: verificationStatusLabel(value),
}));
const METHOD_OPTIONS = employmentVerificationMethodSchema.options.map((value) => ({
  value,
  label: verificationMethodLabel(value),
}));

export type EmployerDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  verification: EmploymentVerificationRow | null;
};

function defaultsFor(
  verification: EmploymentVerificationRow | null,
): EmploymentVerificationFormValues {
  if (!verification) {
    return {
      employerName: "",
      employerDotNumber: null,
      employerMcNumber: null,
      contactName: null,
      contactPhone: null,
      contactEmail: null,
      employedFrom: null,
      employedTo: null,
      wasDotRegulated: true,
      status: "Pending",
      method: "Email",
      requestedAt: null,
      responseReceivedAt: null,
      drugAlcoholResponseReceivedAt: null,
      hadAccidents: false,
      accidentCount: 0,
      hadDrugAlcoholViolations: false,
      findings: null,
      notes: null,
    };
  }
  return {
    employerName: verification.employerName,
    employerDotNumber: verification.employerDotNumber ?? null,
    employerMcNumber: verification.employerMcNumber ?? null,
    contactName: verification.contactName ?? null,
    contactPhone: verification.contactPhone ?? null,
    contactEmail: verification.contactEmail ?? null,
    employedFrom: verification.employedFrom ?? null,
    employedTo: verification.employedTo ?? null,
    wasDotRegulated: verification.wasDotRegulated,
    status: verification.status,
    method: verification.method,
    requestedAt: verification.requestedAt ?? null,
    responseReceivedAt: verification.responseReceivedAt ?? null,
    drugAlcoholResponseReceivedAt: verification.drugAlcoholResponseReceivedAt ?? null,
    hadAccidents: verification.hadAccidents,
    accidentCount: verification.accidentCount,
    hadDrugAlcoholViolations: verification.hadDrugAlcoholViolations,
    findings: verification.findings ?? null,
    notes: verification.notes ?? null,
  };
}

export function EmployerDialog({
  open,
  onOpenChange,
  workerId,
  verification,
}: EmployerDialogProps) {
  const t = useT();

  const invalidate = useDqfInvalidation(workerId);
  const isEdit = Boolean(verification);
  const form = useForm<EmploymentVerificationFormValues>({
    resolver: zodResolver(
      employmentVerificationFormSchema,
    ) as Resolver<EmploymentVerificationFormValues>,
    defaultValues: defaultsFor(verification),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(verification));
  }, [open, verification, reset]);

  const [status, wasDotRegulated, hadAccidents] = useWatch({
    control,
    name: ["status", "wasDotRegulated", "hadAccidents"],
  });
  const answered = status === "Received";

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    EmploymentVerificationFormValues,
    unknown,
    EmploymentVerificationFormValues
  >({
    form,
    resourceName: "Previous employer",
    mutationFn: (values) => {
      const shared = {
        employerName: values.employerName,
        employerDotNumber: values.employerDotNumber ?? undefined,
        employerMcNumber: values.employerMcNumber ?? undefined,
        contactName: values.contactName ?? undefined,
        contactPhone: values.contactPhone ?? undefined,
        contactEmail: values.contactEmail ?? undefined,
        employedFrom: values.employedFrom ?? undefined,
        employedTo: values.employedTo ?? undefined,
        wasDotRegulated: values.wasDotRegulated,
        status: values.status,
        method: values.method,
        requestedAt: values.requestedAt ?? undefined,
        notes: values.notes ?? undefined,
      };
      return verification
        ? updateEmploymentVerification({
            ...shared,
            verificationId: verification.id,
            responseReceivedAt: values.responseReceivedAt ?? undefined,
            drugAlcoholResponseReceivedAt: values.drugAlcoholResponseReceivedAt ?? undefined,
            hadAccidents: values.hadAccidents,
            accidentCount: values.accidentCount,
            hadDrugAlcoholViolations: values.hadDrugAlcoholViolations,
            findings: values.findings ?? undefined,
          })
        : recordEmploymentVerification({ ...shared, workerId });
    },
    onSuccess: () => {
      toast.success(isEdit ? "Employer updated" : "Employer recorded");
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? t("Edit previous employer") : t("Add a previous employer")}
          </DialogTitle>
          <DialogDescription>
            {t(
              "Every DOT-regulated employer in the three years before the application has to be investigated within thirty days of hire (49 CFR 391.23).",
            )}
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
              <FormControl cols="full">
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="employerName"
                  label={t("Employer")}
                  placeholder={t("e.g. Swift Transportation")}
                  description={t(
                    "The carrier or company as it appeared on the driver's application.",
                  )}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="employerDotNumber"
                  label={t("USDOT number")}
                  placeholder={t("e.g. 1234567")}
                  description={t("Identifies the carrier when the safety history request is sent.")}
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="employerMcNumber"
                  label={t("MC number")}
                  placeholder={t("e.g. MC-123456")}
                  description={t("Recorded when the carrier has one, alongside the USDOT number.")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<EmploymentVerificationFormValues>
                  control={control}
                  name="employedFrom"
                  label={t("Employed from")}
                  placeholder={t("First day there")}
                  description={t("When the driver started with this employer.")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<EmploymentVerificationFormValues>
                  control={control}
                  name="employedTo"
                  label={t("Employed to")}
                  placeholder={t("Last day there")}
                  description={t("When the driver left; cannot be before the start date.")}
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="contactName"
                  label={t("Contact")}
                  placeholder={t("e.g. Jane Doe, Safety Manager")}
                  description={t("Who at the employer the request is addressed to.")}
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="contactEmail"
                  label={t("Contact email")}
                  placeholder={t("e.g. safety@carrier.com")}
                  description={t("Where the request goes when it is sent by email.")}
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="contactPhone"
                  label={t("Contact phone")}
                  placeholder={t("e.g. (555) 123-4567")}
                  description={t("Used when the request is made or chased by phone.")}
                />
              </FormControl>
              <FormControl>
                <SelectField<EmploymentVerificationFormValues>
                  control={control}
                  name="method"
                  label={t("Requested by")}
                  options={METHOD_OPTIONS}
                  rules={{ required: true }}
                  placeholder={t("Pick a method")}
                  description={t("How the request was, or will be, sent to the employer.")}
                />
              </FormControl>
              <FormControl>
                <SelectField<EmploymentVerificationFormValues>
                  control={control}
                  name="status"
                  label={t("Status")}
                  options={STATUS_OPTIONS}
                  rules={{ required: true }}
                  placeholder={t("Pick a status")}
                  description={t(
                    "Where the request stands; anything past Pending needs the date it was sent.",
                  )}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<EmploymentVerificationFormValues>
                  control={control}
                  name="requestedAt"
                  label={t("Requested on")}
                  placeholder={t("Date the request went out")}
                  description={t(
                    "Required once the status moves past Pending; the response cannot pre-date it.",
                  )}
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<EmploymentVerificationFormValues>
                  control={control}
                  name="wasDotRegulated"
                  label={t("DOT-regulated employment")}
                  description={t(
                    "Turn off for a job with no safety-sensitive duties; there is then no testing record to ask about.",
                  )}
                />
              </FormControl>

              {isEdit && answered ? (
                <>
                  <FormControl>
                    <AutoCompleteDateField<EmploymentVerificationFormValues>
                      control={control}
                      name="responseReceivedAt"
                      label={t("Response received")}
                      placeholder={t("Date the reply arrived")}
                      description={t(
                        "Required for a received response; cannot be earlier than the request.",
                      )}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  {wasDotRegulated ? (
                    <FormControl>
                      <AutoCompleteDateField<EmploymentVerificationFormValues>
                        control={control}
                        name="drugAlcoholResponseReceivedAt"
                        label={t("Drug and alcohol history received")}
                        placeholder={t("Date the testing history arrived")}
                        description={t("49 CFR 382.413 asks for this specifically.")}
                      />
                    </FormControl>
                  ) : null}
                  <FormControl>
                    <SwitchField<EmploymentVerificationFormValues>
                      control={control}
                      name="hadAccidents"
                      label={t("Accidents reported")}
                      description={t(
                        "Turn on if the employer reported any accidents; the count is asked next.",
                      )}
                    />
                  </FormControl>
                  {hadAccidents ? (
                    <FormControl>
                      <NumberField<EmploymentVerificationFormValues>
                        control={control}
                        name="accidentCount"
                        label={t("How many")}
                        placeholder={t("e.g. 1")}
                        description={t(
                          "Accidents the employer reported for the driver's time there.",
                        )}
                        rules={{ required: true }}
                      />
                    </FormControl>
                  ) : null}
                  {wasDotRegulated ? (
                    <FormControl cols="full">
                      <SwitchField<EmploymentVerificationFormValues>
                        control={control}
                        name="hadDrugAlcoholViolations"
                        label={t("Drug or alcohol violations reported")}
                        description={t(
                          "Turn on when the employer's testing history reports a violation.",
                        )}
                      />
                    </FormControl>
                  ) : null}
                  <FormControl cols="full">
                    <TextareaField<EmploymentVerificationFormValues>
                      control={control}
                      name="findings"
                      label={t("What the employer reported")}
                      placeholder={t("e.g. Company driver, no accidents, eligible for rehire")}
                      description={t(
                        "The employer's answer, kept as the record of the investigation.",
                      )}
                      maxLength={4000}
                    />
                  </FormControl>
                </>
              ) : null}

              <FormControl cols="full">
                <TextareaField<EmploymentVerificationFormValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("e.g. Left a voicemail with HR")}
                  description={t(
                    "Office notes about this employer; not part of the response itself.",
                  )}
                  maxLength={2000}
                />
              </FormControl>

              {isEdit ? null : (
                <FormControl cols="full">
                  <Alert>
                    <AlertDescription>
                      {t(
                        "Record the response once it arrives by editing this employer. Chasing a silent employer is tracked separately — the count of chases is the record of good-faith effort.",
                      )}
                    </AlertDescription>
                  </Alert>
                </FormControl>
              )}
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending}>
                {isEdit ? t("Save") : t("Add")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
