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
import { openLeaveCase, updateLeaveCase, type LeaveCaseRow } from "@/lib/graphql/worker-leave";
import { zodResolver } from "@hookform/resolvers/zod";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import { leaveFrequencyLabel, leaveTypeLabel } from "@trenova/shared/lib/leave";
import { cn } from "@trenova/shared/lib/utils";
import {
  leaveCaseFormSchema,
  leaveFrequencySchema,
  leaveTypeFormSchema,
  type LeaveCaseFormValues,
} from "@trenova/shared/types/worker-leave";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { FMLA_ELIGIBILITY_HOURS } from "./leave-model";
import { useLeaveInvalidation } from "./use-leave-invalidation";

export type LeaveCasePaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  /** The case to edit; null to open a new one. */
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

/** Whether the hours worked meet the 1,250-hour half of the FMLA eligibility test. */
export function eligibilityNote(hours: number | null | undefined): "unknown" | "meets" | "short" {
  if (hours === null || hours === undefined || !Number.isFinite(hours)) return "unknown";
  return hours >= FMLA_ELIGIBILITY_HOURS ? "meets" : "short";
}

/**
 * Opening a leave case, or editing one, beside the cases it joins. A case is one qualifying
 * reason; it draws the entitlement down only as days are recorded against it.
 */
export function LeaveCasePane({ open, onOpenChange, workerId, leaveCase }: LeaveCasePaneProps) {
  const t = useT();
  const invalidate = useLeaveInvalidation(workerId);
  const isEdit = leaveCase !== null;
  const form = useForm<LeaveCaseFormValues>({
    resolver: zodResolver(leaveCaseFormSchema) as Resolver<LeaveCaseFormValues>,
    defaultValues: defaultsFor(leaveCase),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(leaveCase));
  }, [open, leaveCase, reset]);

  const [frequency, leaveType, hoursWorked] = useWatch({
    control,
    name: ["frequency", "leaveType", "eligibilityHoursWorked"],
  });
  const eligibility = eligibilityNote(hoursWorked);
  const typeOptions = leaveTypeFormSchema.options.map((value) => ({
    value,
    label: leaveTypeLabel(value),
  }));
  const frequencyOptions = leaveFrequencySchema.options.map((value) => ({
    value,
    label: leaveFrequencyLabel(value),
  }));

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    LeaveCaseFormValues,
    unknown,
    LeaveCaseFormValues
  >({
    form,
    resourceName: "Leave case",
    scope: { id: `worker-leave-case:${leaveCase?.id ?? workerId}` },
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
    onSuccess: async () => {
      await invalidate();
      onOpenChange(false);
    },
  });

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={isEdit ? t("Edit leave case") : t("Open a leave case")}
      description={t(
        "A case is one qualifying reason. It draws the entitlement down only as days are recorded against it.",
      )}
      contentKey={leaveCase?.id ?? "new"}
    >
      <WorkPaneDirtyTracker form={form} />
      <FormProvider {...form}>
        <Form
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            void handleSubmit((values) => mutateAsync(values))(event);
          }}
        >
          <WorkPaneBody>
            <FormGroup cols={2}>
              <FormControl>
                <SelectField<LeaveCaseFormValues>
                  control={control}
                  name="leaveType"
                  label={t("Type of leave")}
                  options={typeOptions}
                  required
                  placeholder={t("Pick a leave type")}
                  description={t("The law or policy the leave falls under.")}
                />
              </FormControl>
              <FormControl>
                <SelectField<LeaveCaseFormValues>
                  control={control}
                  name="frequency"
                  label={t("How it is taken")}
                  options={frequencyOptions}
                  required
                  placeholder={t("Pick a pattern")}
                  description={t("Continuous is one block; the others are recorded day by day.")}
                />
              </FormControl>
              <FormControl cols="full">
                <InputField<LeaveCaseFormValues>
                  control={control}
                  name="reason"
                  label={t("Qualifying reason")}
                  placeholder={t("e.g. Serious health condition of a parent")}
                  description={t(
                    "The one reason this case covers; open another case for a different reason.",
                  )}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<LeaveCaseFormValues>
                  control={control}
                  name="startsAt"
                  label={t("Leave begins")}
                  placeholder={t("First day of leave")}
                  required
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<LeaveCaseFormValues>
                  control={control}
                  name="endsAt"
                  label={t("Expected to end")}
                  placeholder={t("Leave empty if not yet known")}
                />
              </FormControl>
              {isEdit ? null : (
                <FormControl>
                  <AutoCompleteDateField<LeaveCaseFormValues>
                    control={control}
                    name="requestedAt"
                    label={t("Requested on")}
                    placeholder={t("Date the driver asked")}
                    required
                  />
                </FormControl>
              )}
              <FormControl>
                <NumberField<LeaveCaseFormValues>
                  control={control}
                  name="eligibilityHoursWorked"
                  label={t("Hours worked in the prior year")}
                  placeholder={t("e.g. 1800")}
                />
                <p
                  aria-live="polite"
                  className={cn(
                    "mt-1 text-2xs",
                    eligibility === "meets"
                      ? "text-success-foreground"
                      : eligibility === "short" && leaveType === "FMLA"
                        ? "text-danger-foreground"
                        : "text-muted-foreground",
                  )}
                >
                  {eligibility === "unknown"
                    ? t(
                        "For the 1,250-hour eligibility test. There is no timeclock here to answer it.",
                      )
                    : eligibility === "meets"
                      ? t("Meets the 1,250-hour test.")
                      : leaveType === "FMLA"
                        ? t(
                            "Under 1,250 hours, so not eligible for FMLA. Record it as another type.",
                          )
                        : t("Under 1,250 hours.")}
                </p>
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
                  <Alert
                    variant="info"
                    size="sm"
                    className="animate-expand-in motion-reduce:animate-none"
                  >
                    <AlertDescription>
                      {t(
                        "Intermittent leave is recorded day by day in hours, in the smallest increment used for any other absence (29 CFR 825.205).",
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
                  description={t("Kept on the case and shown on their leave record.")}
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending}>
              {isEdit ? t("Save case") : t("Open case")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
