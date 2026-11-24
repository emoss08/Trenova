import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  recordEmploymentVerification,
  updateEmploymentVerification,
  type EmploymentVerificationRow,
} from "@/lib/graphql/worker-dqf";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Form,
  FormControl,
  FormGroup,
} from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  verificationMethodLabel,
  verificationStatusLabel,
} from "@trenova/shared/lib/dqf";
import {
  employmentVerificationFormSchema,
  employmentVerificationMethodSchema,
  employmentVerificationStatusSchema,
  type EmploymentVerificationFormValues,
} from "@trenova/shared/types/worker-dqf";
import {
  FormProvider,
  useForm,
  useWatch,
  type Resolver,
} from "react-hook-form";
import { PaneFormSection } from "../../pane-section";
import { useWorkerInvalidation } from "../../use-worker-invalidation";

export type EmployerFormPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  verification: EmploymentVerificationRow | null;
  onSaved?: (verificationId: string) => void;
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
    drugAlcoholResponseReceivedAt:
      verification.drugAlcoholResponseReceivedAt ?? null,
    hadAccidents: verification.hadAccidents,
    accidentCount: verification.accidentCount,
    hadDrugAlcoholViolations: verification.hadDrugAlcoholViolations,
    findings: verification.findings ?? null,
    notes: verification.notes ?? null,
  };
}

export function EmployerFormPane({
  open,
  onOpenChange,
  workerId,
  verification,
  onSaved,
}: EmployerFormPaneProps) {
  const t = useT();

  const invalidate = useWorkerInvalidation(workerId, "dq-file");
  const isEdit = Boolean(verification);
  const statusOptions = employmentVerificationStatusSchema.options.map(
    (value) => ({
      value,
      label: verificationStatusLabel(value),
    }),
  );
  const methodOptions = employmentVerificationMethodSchema.options.map(
    (value) => ({
      value,
      label: verificationMethodLabel(value),
    }),
  );
  const form = useForm<EmploymentVerificationFormValues>({
    resolver: zodResolver(
      employmentVerificationFormSchema,
    ) as Resolver<EmploymentVerificationFormValues>,
    defaultValues: defaultsFor(verification),
  });
  const { control, handleSubmit, reset } = form;

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
            drugAlcoholResponseReceivedAt:
              values.drugAlcoholResponseReceivedAt ?? undefined,
            hadAccidents: values.hadAccidents,
            accidentCount: values.accidentCount,
            hadDrugAlcoholViolations: values.hadDrugAlcoholViolations,
            findings: values.findings ?? undefined,
          })
        : recordEmploymentVerification({ ...shared, workerId });
    },
    onSuccess: (saved) => {
      reset(form.getValues());
      void invalidate();
      onSaved?.(saved.id);
      onOpenChange(false);
    },
  });

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={isEdit ? t("Edit previous employer") : t("Add previous employer")}
      description={t(
        "Every DOT-regulated employer in the three years before hire gets a safety history request within thirty days (49 CFR 391.23).",
      )}
    >
      <WorkPaneDirtyTracker form={form} />
      <FormProvider {...form}>
        <Form
          showFormRefusal={false}
          onSubmit={(submitEvent) => {
            submitEvent.preventDefault();
            submitEvent.stopPropagation();
            void handleSubmit(async (values) => {
              await mutateAsync(values);
            })(submitEvent);
          }}
        >
          <WorkPaneBody>
            <PaneFormSection title={t("The employer")}>
              <FormControl cols="full">
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="employerName"
                  label={t("Employer")}
                  placeholder={t("e.g. Swift Transportation")}
                  required
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="employerDotNumber"
                  label={t("USDOT number")}
                  placeholder={t("e.g. 1234567")}
                  inputClassProps="font-mono"
                  description={t(
                    "Identifies the carrier when the request is sent.",
                  )}
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="employerMcNumber"
                  label={t("MC number")}
                  placeholder={t("e.g. MC-123456")}
                  inputClassProps="font-mono"
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<EmploymentVerificationFormValues>
                  control={control}
                  name="employedFrom"
                  label={t("Employed from")}
                  placeholder={t("First day there")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<EmploymentVerificationFormValues>
                  control={control}
                  name="employedTo"
                  label={t("Employed to")}
                  placeholder={t("Last day there")}
                  description={t("Cannot be before the start date.")}
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
                  outlined
                />
              </FormControl>
            </PaneFormSection>

            <PaneFormSection title={t("Contact")}>
              <FormControl cols="full">
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="contactName"
                  label={t("Contact")}
                  placeholder={t("e.g. Jane Doe, Safety Manager")}
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="contactEmail"
                  label={t("Contact email")}
                  placeholder={t("e.g. safety@carrier.com")}
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="contactPhone"
                  label={t("Contact phone")}
                  placeholder={t("e.g. (555) 123-4567")}
                />
              </FormControl>
            </PaneFormSection>

            <PaneFormSection title={t("Request")}>
              <FormControl>
                <SelectField<EmploymentVerificationFormValues>
                  control={control}
                  name="method"
                  label={t("Requested by")}
                  options={methodOptions}
                  required
                  placeholder={t("Pick a method")}
                />
              </FormControl>
              <FormControl>
                <SelectField<EmploymentVerificationFormValues>
                  control={control}
                  name="status"
                  label={t("Status")}
                  options={statusOptions}
                  required
                  placeholder={t("Pick a status")}
                />
              </FormControl>
              <FormControl cols="full">
                <AutoCompleteDateField<EmploymentVerificationFormValues>
                  control={control}
                  required
                  name="requestedAt"
                  label={t("Requested on")}
                  placeholder={t("Date the request went out")}
                  description={t(
                    "Needed once the status moves past not yet requested.",
                  )}
                />
              </FormControl>
              {answered ? (
                <>
                  <FormControl>
                    <AutoCompleteDateField<EmploymentVerificationFormValues>
                      control={control}
                      name="responseReceivedAt"
                      label={t("Response received")}
                      placeholder={t("Date the reply arrived")}
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
                        description={t(
                          "49 CFR 382.413 asks for this specifically.",
                        )}
                      />
                    </FormControl>
                  ) : null}
                </>
              ) : null}
            </PaneFormSection>

            {isEdit && answered ? (
              <PaneFormSection title={t("What they reported")}>
                <FormControl>
                  <SwitchField<EmploymentVerificationFormValues>
                    control={control}
                    name="hadAccidents"
                    label={t("Accidents reported")}
                  />
                </FormControl>
                {hadAccidents ? (
                  <FormControl>
                    <NumberField<EmploymentVerificationFormValues>
                      control={control}
                      name="accidentCount"
                      label={t("How many")}
                      placeholder={t("e.g. 1")}
                      required
                    />
                  </FormControl>
                ) : null}
                {wasDotRegulated ? (
                  <FormControl cols="full">
                    <SwitchField<EmploymentVerificationFormValues>
                      control={control}
                      name="hadDrugAlcoholViolations"
                      label={t("Drug or alcohol violations reported")}
                    />
                  </FormControl>
                ) : null}
                <FormControl cols="full">
                  <TextareaField<EmploymentVerificationFormValues>
                    control={control}
                    name="findings"
                    label={t("What the employer reported")}
                    placeholder={t(
                      "e.g. Company driver, no accidents, eligible for rehire",
                    )}
                    maxLength={4000}
                  />
                </FormControl>
              </PaneFormSection>
            ) : null}

            <FormGroup cols={1}>
              <FormControl cols="full">
                <TextareaField<EmploymentVerificationFormValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("e.g. Left a voicemail with HR")}
                  maxLength={2000}
                  description={
                    isEdit
                      ? undefined
                      : t(
                          "Record the response by editing this employer once it arrives. Each chase is kept as the record of a good-faith effort.",
                        )
                  }
                />
              </FormControl>
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending}>
              {isEdit ? t("Save changes") : t("Add previous employer")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
