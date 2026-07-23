import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  payPeriodFrequencyChoices,
  settlementPayTriggerChoices,
  weekdayChoices,
} from "@/lib/choices";
import { fetchSettlementControl, updateSettlementControl } from "@/lib/graphql/driver-settlement";
import { settlementControlFormSchema, type SettlementControlFormValues } from "@trenova/shared/types/driver-pay";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm, useFormContext, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export default function SettlementControlForm() {
  const queryClient = useQueryClient();
  const { data } = useSuspenseQuery({
    queryKey: ["settlement-control"],
    queryFn: fetchSettlementControl,
  });

  const form = useForm<SettlementControlFormValues>({
    resolver: zodResolver(settlementControlFormSchema) as Resolver<SettlementControlFormValues>,
    defaultValues: {
      payPeriodFrequency: data.payPeriodFrequency,
      periodEndDayOfWeek: data.periodEndDayOfWeek,
      payDelayDays: data.payDelayDays,
      payTrigger: data.payTrigger,
      autoGenerateBatches: data.autoGenerateBatches,
      autoApproveClean: data.autoApproveClean,
      autoAttachAccruals: data.autoAttachAccruals,
      autoPostOnApprove: data.autoPostOnApprove,
      allowNegativeNet: data.allowNegativeNet,
      varianceThresholdPct: Number(data.varianceThresholdPct),
      varianceLookbackWeeks: data.varianceLookbackWeeks,
      defaultEscrowInterestRate: Number(data.defaultEscrowInterestRate),
      escrowInterestFrequencyMonths: data.escrowInterestFrequencyMonths,
    },
  });
  const { handleSubmit, setError, reset } = form;

  const mutation = useApiMutation({
    mutationFn: (values: SettlementControlFormValues) =>
      updateSettlementControl({
        version: data.version,
        payPeriodFrequency: values.payPeriodFrequency,
        periodEndDayOfWeek: values.periodEndDayOfWeek,
        payDelayDays: values.payDelayDays,
        payTrigger: values.payTrigger,
        autoGenerateBatches: values.autoGenerateBatches,
        autoApproveClean: values.autoApproveClean,
        autoAttachAccruals: values.autoAttachAccruals,
        autoPostOnApprove: values.autoPostOnApprove,
        allowNegativeNet: values.allowNegativeNet,
        varianceThresholdPct: String(values.varianceThresholdPct),
        varianceLookbackWeeks: values.varianceLookbackWeeks,
        defaultEscrowInterestRate: String(values.defaultEscrowInterestRate),
        escrowInterestFrequencyMonths: values.escrowInterestFrequencyMonths,
      }),
    onSuccess: (_, values) => {
      toast.success("Settlement control updated");
      reset(values);
      void queryClient.invalidateQueries({ queryKey: ["settlement-control"] });
    },
    setFormError: setError,
    resourceName: "Settlement Control",
  });

  const onSubmit = useCallback(
    (values: SettlementControlFormValues) => mutation.mutate(values),
    [mutation],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <PayPeriodCard />
          <WorkflowCard />
          <ExceptionCard />
          <EscrowCard />
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function PayPeriodCard() {
  const { control } = useFormContext<SettlementControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>Pay Period</CardTitle>
        <CardDescription>
          Defines the settlement cycle and when drivers earn pay for a shipment.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="payPeriodFrequency"
              label="Frequency"
              options={payPeriodFrequencyChoices}
              rules={{ required: true }}
              description="How often drivers are settled — weekly is the industry norm for asset carriers."
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="periodEndDayOfWeek"
              label="Period End Day"
              options={weekdayChoices}
              rules={{ required: true }}
              description="The pay period closes at the start of this day."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="payDelayDays"
              label="Pay Delay (days)"
              description="Days between the period end and the settlement pay date."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="payTrigger"
              label="Pay Trigger"
              options={settlementPayTriggerChoices}
              rules={{ required: true }}
              description="The milestone at which driver pay accrues. Move Completed pays each driver as soon as their own move finishes — the most accurate option when drivers split a load."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function WorkflowCard() {
  const { control } = useFormContext<SettlementControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>Workflow Automation</CardTitle>
        <CardDescription>
          Exception-driven review: automate the clean 90% and focus reviewers on anomalies.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="autoGenerateBatches"
              label="Auto-Generate Batches"
              description="Generate a settlement batch automatically when each pay period closes."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="autoApproveClean"
              label="Auto-Approve Clean Settlements"
              description="Settlements without exceptions skip manual review and go straight to approved."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="autoAttachAccruals"
              label="Auto-Attach New Pay to Open Drafts"
              description="As drivers complete work, new pay events flow into their open draft settlement automatically — no manual transfer needed."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="autoPostOnApprove"
              label="Auto-Post on Approval"
              description="Approving a settlement immediately posts it to the general ledger, collapsing two steps into one."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowNegativeNet"
              label="Allow Negative Net (Carry Forward)"
              description="When deductions exceed earnings, carry the balance to the next settlement instead of capping recoveries."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ExceptionCard() {
  const { control } = useFormContext<SettlementControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>Exception Detection</CardTitle>
        <CardDescription>
          Settlements deviating from a driver&apos;s recent history are flagged for review.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="varianceThresholdPct"
              label="Variance Threshold"
              sideText="%"
              description="Flag when net pay deviates from the trailing average by more than this percentage."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="varianceLookbackWeeks"
              label="Lookback (settlements)"
              description="Number of prior settlements used to compute the trailing average."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function EscrowCard() {
  const { control } = useFormContext<SettlementControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>Escrow Interest</CardTitle>
        <CardDescription>
          49 CFR 376.12(k) requires interest on owner-operator escrow at least quarterly.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="defaultEscrowInterestRate"
              label="Default Annual Interest Rate"
              sideText="%"
              decimalScale={2}
              fixedDecimalScale
              description="Applied to new escrow accounts unless overridden per account."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="escrowInterestFrequencyMonths"
              label="Accrual Frequency (months)"
              description="1–3 months; quarterly is the regulatory maximum interval."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
