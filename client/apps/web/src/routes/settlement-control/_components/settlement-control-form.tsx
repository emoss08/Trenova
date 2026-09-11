import { useT } from "@trenova/shared/i18n/use-t";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  payPeriodFrequencyChoices,
  settlementPayTriggerChoices,
  weekdayChoices,
} from "@/lib/choices";
import { fetchSettlementControl, updateSettlementControl } from "@/lib/graphql/driver-settlement";
import {
  settlementControlFormSchema,
  type SettlementControlFormValues,
} from "@trenova/shared/types/driver-pay";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm, useFormContext, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export default function SettlementControlForm() {
  const t = useT();

  const queryClient = useQueryClient();
  const { data } = useSuspenseQuery({
    queryKey: ["settlement-control"],
    queryFn: ({ signal }) => fetchSettlementControl({ signal }),
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
  const { handleSubmit, reset } = form;

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
      toast.success(t("Settlement control updated"));
      reset(values);
      void queryClient.invalidateQueries({ queryKey: ["settlement-control"] });
    },
    form,
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
          <FormSaveDock saveButtonContent={t("Save Changes")} />
        </div>
      </Form>
    </FormProvider>
  );
}

function PayPeriodCard() {
  const t = useT();

  const { control } = useFormContext<SettlementControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Pay Period")}</CardTitle>
        <CardDescription>
          {t("Defines the settlement cycle and when drivers earn pay for a shipment.")}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="payPeriodFrequency"
              label={t("Frequency")}
              options={payPeriodFrequencyChoices}
              rules={{ required: true }}
              description={t("How often drivers are settled — weekly is the industry norm for asset carriers.")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="periodEndDayOfWeek"
              label={t("Period End Day")}
              options={weekdayChoices}
              rules={{ required: true }}
              description={t("The pay period closes at the start of this day.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="payDelayDays"
              label={t("Pay Delay (days)")}
              description={t("Days between the period end and the settlement pay date.")}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="payTrigger"
              label={t("Pay Trigger")}
              options={settlementPayTriggerChoices}
              rules={{ required: true }}
              description={t("The milestone at which driver pay accrues. Move Completed pays each driver as soon as their own move finishes — the most accurate option when drivers split a load.")}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function WorkflowCard() {
  const t = useT();

  const { control } = useFormContext<SettlementControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Workflow Automation")}</CardTitle>
        <CardDescription>
          {t("Exception-driven review: automate the clean 90% and focus reviewers on anomalies.")}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="autoGenerateBatches"
              label={t("Auto-Generate Batches")}
              description={t("Generate a settlement batch automatically when each pay period closes.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="autoApproveClean"
              label={t("Auto-Approve Clean Settlements")}
              description={t("Settlements without exceptions skip manual review and go straight to approved.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="autoAttachAccruals"
              label={t("Auto-Attach New Pay to Open Drafts")}
              description={t("As drivers complete work, new pay events flow into their open draft settlement automatically — no manual transfer needed.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="autoPostOnApprove"
              label={t("Auto-Post on Approval")}
              description={t("Approving a settlement immediately posts it to the general ledger, collapsing two steps into one.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowNegativeNet"
              label={t("Allow Negative Net (Carry Forward)")}
              description={t("When deductions exceed earnings, carry the balance to the next settlement instead of capping recoveries.")}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ExceptionCard() {
  const t = useT();

  const { control } = useFormContext<SettlementControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Exception Detection")}</CardTitle>
        <CardDescription>
          {t("Settlements deviating from a driver's recent history are flagged for review.")}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="varianceThresholdPct"
              label={t("Variance Threshold")}
              sideText="%"
              description={t("Flag when net pay deviates from the trailing average by more than this percentage.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="varianceLookbackWeeks"
              label={t("Lookback (settlements)")}
              description={t("Number of prior settlements used to compute the trailing average.")}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function EscrowCard() {
  const t = useT();

  const { control } = useFormContext<SettlementControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Escrow Interest")}</CardTitle>
        <CardDescription>
          {t("49 CFR 376.12(k) requires interest on owner-operator escrow at least quarterly.")}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="defaultEscrowInterestRate"
              label={t("Default Annual Interest Rate")}
              sideText="%"
              decimalScale={2}
              fixedDecimalScale
              description={t("Applied to new escrow accounts unless overridden per account.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="escrowInterestFrequencyMonths"
              label={t("Accrual Frequency (months)")}
              description={t("1–3 months; quarterly is the regulatory maximum interval.")}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
