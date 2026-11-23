import { BenefitPlanAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { MoneyField } from "@/components/fields/money-field";
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
import { enrollBenefit } from "@/lib/graphql/benefits";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { COVERAGE_TIER_ORDER, coverageTierLabel } from "@trenova/shared/lib/benefits";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  benefitEnrollmentFormSchema,
  type BenefitEnrollmentFormValues,
} from "@trenova/shared/types/benefits";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { useWorkerInvalidation } from "../../use-worker-invalidation";

function startEnrollment(
  planId: string | null,
  coverageTier: string | null,
): BenefitEnrollmentFormValues {
  return {
    benefitPlanId: planId ?? "",
    coverageTier: (coverageTier ?? "Employee") as BenefitEnrollmentFormValues["coverageTier"],
    waive: false,
    waivedReason: null,
    employeeCostMinor: null,
    effectiveFrom: getTodayDate(),
    notes: null,
  };
}

export type EnrollPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  /** The plan the pane starts on, from a row's "Enroll or decline" or "Change coverage". */
  planId: string | null;
  /** The tier the worker holds now, when changing coverage. */
  coverageTier: string | null;
  onSaved: () => void;
};

/**
 * Records an election beside the enrolments: the plan and tier, or a decline with its reason,
 * which auditors ask for. Enrolling opens the settlement deduction for the employee's share.
 */
export function EnrollPane({
  open,
  onOpenChange,
  workerId,
  planId,
  coverageTier,
  onSaved,
}: EnrollPaneProps) {
  const t = useT();
  const invalidate = useWorkerInvalidation(workerId, "benefits");
  const form = useForm<BenefitEnrollmentFormValues>({
    resolver: zodResolver(benefitEnrollmentFormSchema) as Resolver<BenefitEnrollmentFormValues>,
    defaultValues: startEnrollment(planId, coverageTier),
  });
  const { control, handleSubmit } = form;

  const waive = useWatch({ control, name: "waive" });
  const tierOptions = COVERAGE_TIER_ORDER.map((value) => ({
    value,
    label: coverageTierLabel(value),
  }));

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    BenefitEnrollmentFormValues,
    unknown,
    BenefitEnrollmentFormValues
  >({
    form,
    resourceName: "Enrollment",
    mutationFn: (values) =>
      enrollBenefit({
        workerId,
        benefitPlanId: values.benefitPlanId,
        coverageTier: values.coverageTier,
        waive: values.waive,
        waivedReason: values.waivedReason ?? undefined,
        employeeCostMinor: values.employeeCostMinor ?? undefined,
        effectiveFrom: values.effectiveFrom,
        notes: values.notes ?? undefined,
      }),
    onSuccess: async () => {
      await invalidate();
      onSaved();
    },
  });

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={coverageTier ? t("Change coverage") : t("Enroll or decline")}
      description={t(
        "Records the election for the plan year. A decline is kept on file with its reason.",
      )}
      contentKey={planId ?? "any"}
    >
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
            <FormGroup cols={1}>
              <FormControl cols="full">
                <BenefitPlanAutocompleteField<BenefitEnrollmentFormValues>
                  control={control}
                  name="benefitPlanId"
                  label={t("Plan")}
                  required
                  placeholder={t("Pick a plan")}
                  description={t("Only plans open for enrollment are listed.")}
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<BenefitEnrollmentFormValues>
                  control={control}
                  name="waive"
                  label={t("They declined the cover")}
                  description={t("Keeps the decline on file, which the plan's auditors ask for.")}
                  position="left"
                  outlined
                />
              </FormControl>
              {waive ? (
                <FormControl cols="full" className="motion-safe:animate-expand-in">
                  <InputField<BenefitEnrollmentFormValues>
                    control={control}
                    name="waivedReason"
                    label={t("Why")}
                    placeholder={t("e.g. Covered by a spouse's plan")}
                    required
                  />
                </FormControl>
              ) : (
                <>
                  <FormControl cols="full" className="motion-safe:animate-expand-in">
                    <SelectField<BenefitEnrollmentFormValues>
                      control={control}
                      name="coverageTier"
                      label={t("Coverage")}
                      options={tierOptions}
                      placeholder={t("Pick a tier")}
                      description={t("The plan's employee price is scaled for a wider tier.")}
                      required
                    />
                  </FormControl>
                  <FormControl cols="full" className="motion-safe:animate-expand-in">
                    <MoneyField<BenefitEnrollmentFormValues>
                      control={control}
                      name="employeeCostMinor"
                      label={t("Employee cost per period")}
                      placeholder="0.00"
                      description={t("Leave empty to take the plan's own arithmetic for the tier.")}
                    />
                  </FormControl>
                </>
              )}
              <FormControl cols="full">
                <AutoCompleteDateField<BenefitEnrollmentFormValues>
                  control={control}
                  name="effectiveFrom"
                  label={t("Effective from")}
                  required
                  placeholder={t("e.g. First of next month")}
                  description={t(
                    "The deduction is taken from the first settlement on or after it.",
                  )}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<BenefitEnrollmentFormValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("e.g. Enrolled during open enrollment")}
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending}>
              {waive ? t("Record") : t("Enroll")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
