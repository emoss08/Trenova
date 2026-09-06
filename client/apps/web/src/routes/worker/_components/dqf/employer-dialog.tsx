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
          <DialogTitle>{isEdit ? "Edit previous employer" : "Add a previous employer"}</DialogTitle>
          <DialogDescription>
            Every DOT-regulated employer in the three years before the application has to be
            investigated within thirty days of hire (49 CFR 391.23).
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
                  label="Employer"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="employerDotNumber"
                  label="USDOT number"
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="employerMcNumber"
                  label="MC number"
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<EmploymentVerificationFormValues>
                  control={control}
                  name="employedFrom"
                  label="Employed from"
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<EmploymentVerificationFormValues>
                  control={control}
                  name="employedTo"
                  label="Employed to"
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="contactName"
                  label="Contact"
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="contactEmail"
                  label="Contact email"
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentVerificationFormValues>
                  control={control}
                  name="contactPhone"
                  label="Contact phone"
                />
              </FormControl>
              <FormControl>
                <SelectField<EmploymentVerificationFormValues>
                  control={control}
                  name="method"
                  label="Requested by"
                  options={METHOD_OPTIONS}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SelectField<EmploymentVerificationFormValues>
                  control={control}
                  name="status"
                  label="Status"
                  options={STATUS_OPTIONS}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<EmploymentVerificationFormValues>
                  control={control}
                  name="requestedAt"
                  label="Requested on"
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<EmploymentVerificationFormValues>
                  control={control}
                  name="wasDotRegulated"
                  label="DOT-regulated employment"
                  description="Turn off for a job with no safety-sensitive duties; there is then no testing record to ask about."
                />
              </FormControl>

              {isEdit && answered ? (
                <>
                  <FormControl>
                    <AutoCompleteDateField<EmploymentVerificationFormValues>
                      control={control}
                      name="responseReceivedAt"
                      label="Response received"
                      rules={{ required: true }}
                    />
                  </FormControl>
                  {wasDotRegulated ? (
                    <FormControl>
                      <AutoCompleteDateField<EmploymentVerificationFormValues>
                        control={control}
                        name="drugAlcoholResponseReceivedAt"
                        label="Drug and alcohol history received"
                        description="49 CFR 382.413 asks for this specifically."
                      />
                    </FormControl>
                  ) : null}
                  <FormControl>
                    <SwitchField<EmploymentVerificationFormValues>
                      control={control}
                      name="hadAccidents"
                      label="Accidents reported"
                    />
                  </FormControl>
                  {hadAccidents ? (
                    <FormControl>
                      <NumberField<EmploymentVerificationFormValues>
                        control={control}
                        name="accidentCount"
                        label="How many"
                        rules={{ required: true }}
                      />
                    </FormControl>
                  ) : null}
                  {wasDotRegulated ? (
                    <FormControl cols="full">
                      <SwitchField<EmploymentVerificationFormValues>
                        control={control}
                        name="hadDrugAlcoholViolations"
                        label="Drug or alcohol violations reported"
                      />
                    </FormControl>
                  ) : null}
                  <FormControl cols="full">
                    <TextareaField<EmploymentVerificationFormValues>
                      control={control}
                      name="findings"
                      label="What the employer reported"
                      maxLength={4000}
                    />
                  </FormControl>
                </>
              ) : null}

              <FormControl cols="full">
                <TextareaField<EmploymentVerificationFormValues>
                  control={control}
                  name="notes"
                  label="Notes"
                  maxLength={2000}
                />
              </FormControl>

              {isEdit ? null : (
                <FormControl cols="full">
                  <Alert>
                    <AlertDescription>
                      Record the response once it arrives by editing this employer. Chasing a silent
                      employer is tracked separately — the count of chases is the record of
                      good-faith effort.
                    </AlertDescription>
                  </Alert>
                </FormControl>
              )}
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending}>
                {isEdit ? "Save" : "Add"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
