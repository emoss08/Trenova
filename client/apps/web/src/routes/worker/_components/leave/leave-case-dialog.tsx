import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { openLeaveCase, updateLeaveCase, type LeaveCaseRow } from "@/lib/graphql/worker-leave";
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
import { getTodayDate } from "@trenova/shared/lib/date";
import { leaveFrequencyLabel, leaveTypeLabel } from "@trenova/shared/lib/leave";
import {
  leaveCaseFormSchema,
  leaveFrequencySchema,
  leaveTypeFormSchema,
  type LeaveCaseFormValues,
} from "@trenova/shared/types/worker-leave";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useLeaveInvalidation } from "./use-leave-invalidation";

const TYPE_OPTIONS = leaveTypeFormSchema.options.map((value) => ({
  value,
  label: leaveTypeLabel(value),
}));
const FREQUENCY_OPTIONS = leaveFrequencySchema.options.map((value) => ({
  value,
  label: leaveFrequencyLabel(value),
}));

export type LeaveCaseDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  leaveCase: LeaveCaseRow | null;
};

function defaultsFor(leaveCase: LeaveCaseRow | null): LeaveCaseFormValues {
  if (!leaveCase) {
    return {
      leaveType: "FMLA",
      frequency: "Continuous",
      reason: null,
      militaryCaregiver: false,
      requestedAt: getTodayDate(),
      startsAt: getTodayDate(),
      endsAt: null,
      eligibilityHoursWorked: null,
      notes: null,
    };
  }
  return {
    leaveType: leaveCase.leaveType as LeaveCaseFormValues["leaveType"],
    frequency: leaveCase.frequency,
    reason: leaveCase.reason ?? null,
    militaryCaregiver: leaveCase.militaryCaregiver,
    requestedAt: leaveCase.requestedAt,
    startsAt: leaveCase.startsAt,
    endsAt: leaveCase.endsAt ?? null,
    eligibilityHoursWorked: leaveCase.eligibilityHoursWorked ?? null,
    notes: leaveCase.notes ?? null,
  };
}

export function LeaveCaseDialog({ open, onOpenChange, workerId, leaveCase }: LeaveCaseDialogProps) {
  const t = useT();

  const invalidate = useLeaveInvalidation(workerId);
  const isEdit = Boolean(leaveCase);
  const form = useForm<LeaveCaseFormValues>({
    resolver: zodResolver(leaveCaseFormSchema) as Resolver<LeaveCaseFormValues>,
    defaultValues: defaultsFor(leaveCase),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(leaveCase));
  }, [open, leaveCase, reset]);

  const [frequency] = useWatch({ control, name: ["frequency"] });

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    LeaveCaseFormValues,
    unknown,
    LeaveCaseFormValues
  >({
    form,
    resourceName: "Leave case",
    mutationFn: (values) => {
      const shared = {
        leaveType: values.leaveType,
        frequency: values.frequency,
        reason: values.reason ?? undefined,
        militaryCaregiver: values.militaryCaregiver,
        startsAt: values.startsAt,
        endsAt: values.endsAt ?? undefined,
        eligibilityHoursWorked: values.eligibilityHoursWorked ?? undefined,
        notes: values.notes ?? undefined,
      };
      return leaveCase
        ? updateLeaveCase({ ...shared, caseId: leaveCase.id })
        : openLeaveCase({ ...shared, workerId, requestedAt: values.requestedAt });
    },
    onSuccess: () => {
      toast.success(isEdit ? "Leave case updated" : "Leave case opened", {
        description: isEdit
          ? undefined
          : "Approving and designating are separate decisions — nothing is drawn down yet.",
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? t("Edit leave case") : t("Open a leave case")}</DialogTitle>
          <DialogDescription>
            {t(
              "A case is one qualifying reason. The entitlement is drawn down by the days recorded against it, so a case open for months has used nothing until days are.",
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
              <FormControl>
                <SelectField<LeaveCaseFormValues>
                  control={control}
                  name="leaveType"
                  label={t("Type of leave")}
                  options={TYPE_OPTIONS}
                  rules={{ required: true }}
                  placeholder={t("Pick a leave type")}
                  description={t("The law or policy the leave falls under.")}
                />
              </FormControl>
              <FormControl>
                <SelectField<LeaveCaseFormValues>
                  control={control}
                  name="frequency"
                  label={t("How it is taken")}
                  options={FREQUENCY_OPTIONS}
                  rules={{ required: true }}
                  placeholder={t("Pick a pattern")}
                  description={t(
                    "Continuous is one block of time; intermittent and reduced schedule are recorded day by day.",
                  )}
                />
              </FormControl>
              <FormControl cols="full">
                <InputField<LeaveCaseFormValues>
                  control={control}
                  name="reason"
                  label={t("Qualifying reason")}
                  placeholder={t("e.g. Serious health condition of a parent")}
                  description={t(
                    "The one qualifying reason this case covers; open another case for a different reason.",
                  )}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<LeaveCaseFormValues>
                  control={control}
                  name="startsAt"
                  label={t("Leave begins")}
                  placeholder={t("First day of leave")}
                  description={t(
                    "The first day of the leave; the expected end cannot be before it.",
                  )}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<LeaveCaseFormValues>
                  control={control}
                  name="endsAt"
                  label={t("Expected to end")}
                  placeholder={t("Leave empty if not yet known")}
                  description={t(
                    "When the leave is expected to finish; closing the case fills it in if still blank.",
                  )}
                />
              </FormControl>
              {isEdit ? null : (
                <FormControl>
                  <AutoCompleteDateField<LeaveCaseFormValues>
                    control={control}
                    name="requestedAt"
                    label={t("Requested on")}
                    placeholder={t("Date the driver asked")}
                    description={t("When the driver asked for the leave; defaults to today.")}
                    rules={{ required: true }}
                  />
                </FormControl>
              )}
              <FormControl>
                <NumberField<LeaveCaseFormValues>
                  control={control}
                  name="eligibilityHoursWorked"
                  label={t("Hours worked in the prior year")}
                  placeholder={t("e.g. 1800")}
                  description={t(
                    "For the 1,250-hour eligibility test. There is no timeclock here, so it is recorded by hand.",
                  )}
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<LeaveCaseFormValues>
                  control={control}
                  name="militaryCaregiver"
                  label={t("Military caregiver leave")}
                  description={t(
                    "Raises the entitlement to 26 weeks for the period (29 CFR 825.127).",
                  )}
                />
              </FormControl>
              {frequency === "Intermittent" || frequency === "ReducedSchedule" ? (
                <FormControl cols="full">
                  <Alert>
                    <AlertDescription>
                      {t(
                        "Intermittent leave is recorded day by day in hours, in the smallest increment the organisation uses for any other absence (29 CFR 825.205).",
                      )}
                    </AlertDescription>
                  </Alert>
                </FormControl>
              ) : null}
              <FormControl cols="full">
                <TextareaField<LeaveCaseFormValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("e.g. Certification requested from the physician")}
                  description={t("Kept on the case and shown on the driver's leave record.")}
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending}>
                {isEdit ? t("Save") : t("Open case")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
