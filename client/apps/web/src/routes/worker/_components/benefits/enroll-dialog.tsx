import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { MoneyField } from "@/components/fields/money-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  BENEFIT_ENROLLMENTS_KEY,
  enrollBenefit,
  fetchBenefitPlans,
  TOTAL_COMPENSATION_KEY,
} from "@/lib/graphql/benefits";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
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
import {
  benefitPlanTypeLabel,
  COVERAGE_TIER_ORDER,
  coverageTierLabel,
} from "@trenova/shared/lib/benefits";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  benefitEnrollmentFormSchema,
  type BenefitEnrollmentFormValues,
} from "@trenova/shared/types/benefits";
import { useEffect, useMemo } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const TIER_OPTIONS = COVERAGE_TIER_ORDER.map((value) => ({
  value,
  label: coverageTierLabel(value),
}));

export type EnrollDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
};

function emptyEnrollment(): BenefitEnrollmentFormValues {
  return {
    benefitPlanId: "",
    coverageTier: "Employee",
    waive: false,
    waivedReason: null,
    employeeCostMinor: null,
    effectiveFrom: getTodayDate(),
    notes: null,
  };
}

export function EnrollDialog({ open, onOpenChange, workerId }: EnrollDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const form = useForm<BenefitEnrollmentFormValues>({
    resolver: zodResolver(benefitEnrollmentFormSchema) as Resolver<BenefitEnrollmentFormValues>,
    defaultValues: emptyEnrollment(),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(emptyEnrollment());
  }, [open, reset]);

  const waive = useWatch({ control, name: "waive" });

  const plansQuery = useQuery({
    queryKey: ["benefit-plans", "active"],
    queryFn: ({ signal }) => fetchBenefitPlans({ activeOnly: true }, { signal }),
    enabled: open,
  });

  const planOptions = useMemo(
    () =>
      (plansQuery.data ?? []).map((plan) => ({
        value: plan.id,
        label: `${plan.name} (${plan.planYear})`,
        description: benefitPlanTypeLabel(plan.planType),
      })),
    [plansQuery.data],
  );

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
    onSuccess: (_, values) => {
      toast.success(values.waive ? "Declination recorded" : "Enrolled", {
        description: values.waive
          ? "No deduction is created — a waiver takes nothing."
          : "A settlement deduction now collects their contribution.",
      });
      void queryClient.invalidateQueries({ queryKey: [BENEFIT_ENROLLMENTS_KEY, workerId] });
      void queryClient.invalidateQueries({ queryKey: [TOTAL_COMPENSATION_KEY, workerId] });
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Enroll or decline")}</DialogTitle>
          <DialogDescription>
            {t("Enrolling opens a settlement deduction for the employee contribution. Declining records the decision and takes nothing.")}
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
                <SelectField<BenefitEnrollmentFormValues>
                  control={control}
                  name="benefitPlanId"
                  label={t("Plan")}
                  options={planOptions}
                  rules={{ required: true }}
                  placeholder={t("Pick a plan")}
                  description={t("The plan they are joining or declining; only plans open for enrollment are listed.")}
                />
              </FormControl>
              <FormControl>
                <SelectField<BenefitEnrollmentFormValues>
                  control={control}
                  name="coverageTier"
                  label={t("Coverage")}
                  options={TIER_OPTIONS}
                  isReadOnly={waive}
                  placeholder={t("Pick a tier")}
                  description={t("The plan's employee price is scaled for a wider tier.")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<BenefitEnrollmentFormValues>
                  control={control}
                  name="effectiveFrom"
                  label={t("Effective from")}
                  rules={{ required: true }}
                  placeholder={t("e.g. First of next month")}
                  description={t("The day the cover starts; the deduction is taken from the first settlement on or after it.")}
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<BenefitEnrollmentFormValues>
                  control={control}
                  name="waive"
                  label={t("They declined the cover")}
                  description={t("Recorded rather than left blank: declined and nobody-asked are different facts at audit.")}
                />
              </FormControl>
              {waive ? (
                <FormControl cols="full">
                  <InputField<BenefitEnrollmentFormValues>
                    control={control}
                    name="waivedReason"
                    label={t("Why")}
                    placeholder={t("e.g. Covered by a spouse's plan")}
                    rules={{ required: true }}
                    description={t("Kept on the enrollment as the record of why the cover was declined.")}
                  />
                </FormControl>
              ) : (
                <>
                  <FormControl cols="full">
                    <MoneyField<BenefitEnrollmentFormValues>
                      control={control}
                      name="employeeCostMinor"
                      label={t("Employee cost per period")}
                      placeholder="0.00"
                      description={t("Leave empty to take the plan's own arithmetic for the tier.")}
                    />
                  </FormControl>
                  <FormControl cols="full">
                    <Alert variant="info">
                      <AlertDescription>
                        {t("The price is copied onto the enrollment, so repricing the plan next year cannot restate what they were charged this year.")}
                      </AlertDescription>
                    </Alert>
                  </FormControl>
                </>
              )}
              <FormControl cols="full">
                <TextareaField<BenefitEnrollmentFormValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("e.g. Enrolled during open enrollment")}
                  description={t("Kept with the enrollment and shown on the worker's benefits record.")}
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending}>
                {waive ? t("Record") : t("Enroll")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
