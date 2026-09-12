import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { MoneyField } from "@/components/fields/money-field";
import { NumberField } from "@/components/fields/number-field";
import { PayCodeSelectField } from "@/components/fields/pay-code-select-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  BENEFIT_COSTS_KEY,
  BENEFIT_PLANS_KEY,
  createBenefitPlan,
  updateBenefitPlan,
  type BenefitPlanRow,
} from "@/lib/graphql/benefits";
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
import { BENEFIT_PLAN_TYPE_ORDER, benefitPlanTypeLabel } from "@trenova/shared/lib/benefits";
import { getTodayDate } from "@trenova/shared/lib/date";
import { benefitPlanFormSchema, type BenefitPlanFormValues } from "@trenova/shared/types/benefits";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const TYPE_OPTIONS = BENEFIT_PLAN_TYPE_ORDER.map((value) => ({
  value,
  label: benefitPlanTypeLabel(value),
}));

const STATUS_OPTIONS = [
  { value: "Active", label: "Active" },
  { value: "Inactive", label: "Archived" },
];

function currentYear(): number {
  return new Date(getTodayDate() * 1000).getUTCFullYear();
}

export type BenefitPlanDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  plan: BenefitPlanRow | null;
};

function defaultsFor(plan: BenefitPlanRow | null): BenefitPlanFormValues {
  if (!plan) {
    return {
      code: "",
      name: "",
      description: null,
      planType: "Medical",
      carrier: null,
      policyNumber: null,
      payCodeId: "",
      planYear: currentYear(),
      employeeCostMinor: 0,
      employerCostMinor: 0,
      waitingPeriodDays: 0,
      status: "Active",
    };
  }
  return {
    code: plan.code,
    name: plan.name,
    description: plan.description ?? null,
    planType: plan.planType as BenefitPlanFormValues["planType"],
    carrier: plan.carrier ?? null,
    policyNumber: plan.policyNumber ?? null,
    payCodeId: plan.payCodeId,
    planYear: plan.planYear,
    employeeCostMinor: plan.employeeCostMinor,
    employerCostMinor: plan.employerCostMinor,
    waitingPeriodDays: plan.waitingPeriodDays,
    status: plan.status === "Active" ? "Active" : "Inactive",
  };
}

export function BenefitPlanDialog({ open, onOpenChange, plan }: BenefitPlanDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const isEdit = Boolean(plan);
  const form = useForm<BenefitPlanFormValues>({
    resolver: zodResolver(benefitPlanFormSchema) as Resolver<BenefitPlanFormValues>,
    defaultValues: defaultsFor(plan),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(plan));
  }, [open, plan, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    BenefitPlanFormValues,
    unknown,
    BenefitPlanFormValues
  >({
    form,
    resourceName: "Plan",
    mutationFn: (values) => {
      const shared = {
        code: values.code,
        name: values.name,
        description: values.description ?? undefined,
        planType: values.planType,
        carrier: values.carrier ?? undefined,
        policyNumber: values.policyNumber ?? undefined,
        payCodeId: values.payCodeId,
        planYear: values.planYear,
        employeeCostMinor: values.employeeCostMinor,
        employerCostMinor: values.employerCostMinor,
        waitingPeriodDays: values.waitingPeriodDays,
        status: values.status,
      };
      return plan
        ? updateBenefitPlan({ ...shared, id: plan.id, version: plan.version })
        : createBenefitPlan(shared);
    },
    onSuccess: () => {
      toast.success(isEdit ? "Plan updated" : "Plan added");
      void queryClient.invalidateQueries({ queryKey: [BENEFIT_PLANS_KEY] });
      void queryClient.invalidateQueries({ queryKey: [BENEFIT_COSTS_KEY] });
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? t("Edit the plan") : t("Add a plan")}</DialogTitle>
          <DialogDescription>
            {t("Each plan year is its own row. The pay code is what a contribution shows up as on a settlement, which is why it is required.")}
          </DialogDescription>
        </DialogHeader>
        {isEdit ? (
          <Alert variant="warning">
            <AlertDescription>
              {t("Each enrolment keeps the price it was made at, so changing a cost here only affects people enrolled from now on; existing deductions are not repriced.")}
            </AlertDescription>
          </Alert>
        ) : null}
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
                <InputField<BenefitPlanFormValues>
                  control={control}
                  name="code"
                  label={t("Code")}
                  placeholder={t("e.g. MED-PPO")}
                  rules={{ required: true }}
                  description={t("A short tag that identifies the plan on lists and enrolments, up to 20 characters.")}
                />
              </FormControl>
              <FormControl>
                <InputField<BenefitPlanFormValues>
                  control={control}
                  name="name"
                  label={t("Name")}
                  placeholder={t("e.g. Medical PPO")}
                  rules={{ required: true }}
                  description={t("The plan name workers see when they are enrolled in it or decline it.")}
                />
              </FormControl>
              <FormControl>
                <SelectField<BenefitPlanFormValues>
                  control={control}
                  name="planType"
                  label={t("Type")}
                  options={TYPE_OPTIONS}
                  placeholder={t("Pick a type")}
                  rules={{ required: true }}
                  description={t("The kind of cover this is; plans are grouped by it on the benefits page.")}
                />
              </FormControl>
              <FormControl>
                <NumberField<BenefitPlanFormValues>
                  control={control}
                  name="planYear"
                  label={t("Plan year")}
                  placeholder={t("e.g. 2026")}
                  rules={{ required: true }}
                  description={t("The year this pricing applies to; set up next year's plan as a new row instead of editing this one.")}
                />
              </FormControl>
              <FormControl>
                <PayCodeSelectField<BenefitPlanFormValues>
                  control={control}
                  name="payCodeId"
                  label={t("Pay code")}
                  required
                  description={t("The line a contribution shows up as on a settlement, so every deduction can be explained.")}
                />
              </FormControl>
              <FormControl>
                <SelectField<BenefitPlanFormValues>
                  control={control}
                  name="status"
                  label={t("Status")}
                  options={STATUS_OPTIONS}
                  placeholder={t("Pick a status")}
                  description={t("An archived plan cannot be enrolled in, and archiving is refused while anybody is still enrolled.")}
                />
              </FormControl>
              <FormControl>
                <MoneyField<BenefitPlanFormValues>
                  control={control}
                  name="employeeCostMinor"
                  label={t("Employee cost per period")}
                  placeholder="0.00"
                  description={t("What the worker pays each pay period; it is deducted from every settlement while they are enrolled.")}
                />
              </FormControl>
              <FormControl>
                <MoneyField<BenefitPlanFormValues>
                  control={control}
                  name="employerCostMinor"
                  label={t("Employer cost per period")}
                  placeholder="0.00"
                  description={t("What the company pays each pay period; never deducted, but carried so a total-compensation statement shows what the job is worth.")}
                />
              </FormControl>
              <FormControl>
                <InputField<BenefitPlanFormValues>
                  control={control}
                  name="carrier"
                  label={t("Carrier")}
                  placeholder={t("e.g. Blue Shield")}
                  description={t("The insurer or provider that underwrites the plan, shown on the plan card.")}
                />
              </FormControl>
              <FormControl>
                <InputField<BenefitPlanFormValues>
                  control={control}
                  name="policyNumber"
                  label={t("Policy number")}
                  placeholder={t("e.g. P-100")}
                  description={t("The carrier's policy or group number, so a query about cover can be matched to the right contract.")}
                />
              </FormControl>
              <FormControl cols="full">
                <NumberField<BenefitPlanFormValues>
                  control={control}
                  name="waitingPeriodDays"
                  label={t("Waiting period (days)")}
                  placeholder={t("e.g. 30")}
                  description={t("How long after hire somebody becomes eligible, up to a year; it is shown on the plan card but not checked when enrolling.")}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<BenefitPlanFormValues>
                  control={control}
                  name="description"
                  label={t("Description")}
                  maxLength={2000}
                  placeholder={t("e.g. PPO with a $500 deductible and a nationwide network")}
                  description={t("Anything an administrator should know about the plan that the fields above do not say.")}
                />
              </FormControl>
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
