import { PayCodeSelectField } from "@/components/fields/pay-code-select-field";
import { InputField } from "@/components/fields/input-field";
import { MoneyField } from "@/components/fields/money-field";
import { NumberField } from "@/components/fields/number-field";
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
          <DialogTitle>{isEdit ? "Edit the plan" : "Add a plan"}</DialogTitle>
          <DialogDescription>
            Each plan year is its own row. The pay code is what a contribution shows up as on a
            settlement, which is why it is required.
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
                <InputField<BenefitPlanFormValues>
                  control={control}
                  name="code"
                  label="Code"
                  placeholder="e.g. MED-PPO"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <InputField<BenefitPlanFormValues>
                  control={control}
                  name="name"
                  label="Name"
                  placeholder="e.g. Medical PPO"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SelectField<BenefitPlanFormValues>
                  control={control}
                  name="planType"
                  label="Type"
                  options={TYPE_OPTIONS}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <NumberField<BenefitPlanFormValues>
                  control={control}
                  name="planYear"
                  label="Plan year"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <PayCodeSelectField<BenefitPlanFormValues>
                  control={control}
                  name="payCodeId"
                  label="Pay code"
                  required
                  description="What a contribution shows up as on a settlement."
                />
              </FormControl>
              <FormControl>
                <SelectField<BenefitPlanFormValues>
                  control={control}
                  name="status"
                  label="Status"
                  options={STATUS_OPTIONS}
                  description="Archiving is refused while anybody is still enrolled."
                />
              </FormControl>
              <FormControl>
                <MoneyField<BenefitPlanFormValues>
                  control={control}
                  name="employeeCostMinor"
                  label="Employee cost per period"
                  description="Deducted from each settlement."
                />
              </FormControl>
              <FormControl>
                <MoneyField<BenefitPlanFormValues>
                  control={control}
                  name="employerCostMinor"
                  label="Employer cost per period"
                  description="Never deducted. Carried so a total-compensation statement can show what the job is worth."
                />
              </FormControl>
              <FormControl>
                <InputField<BenefitPlanFormValues>
                  control={control}
                  name="carrier"
                  label="Carrier"
                />
              </FormControl>
              <FormControl>
                <InputField<BenefitPlanFormValues>
                  control={control}
                  name="policyNumber"
                  label="Policy number"
                />
              </FormControl>
              <FormControl>
                <NumberField<BenefitPlanFormValues>
                  control={control}
                  name="waitingPeriodDays"
                  label="Waiting period (days)"
                  description="How long after hire somebody becomes eligible."
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<BenefitPlanFormValues>
                  control={control}
                  name="description"
                  label="Description"
                  maxLength={2000}
                />
              </FormControl>
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
