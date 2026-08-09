import { GLAccountAutocompleteField } from "@/components/autocomplete-fields";
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
import {
  fetchCarrierSettlementControl,
  updateCarrierSettlementControl,
} from "@/lib/graphql/carrier-settlement";
import {
  carrierSettlementControlFormSchema,
  type CarrierSettlementControlFormValues,
} from "@trenova/shared/types/carrier-settlement";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm, useFormContext, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export default function CarrierSettlementControlForm() {
  const queryClient = useQueryClient();
  const { data } = useSuspenseQuery({
    queryKey: ["carrier-settlement-control"],
    queryFn: fetchCarrierSettlementControl,
  });

  const form = useForm<CarrierSettlementControlFormValues>({
    resolver: zodResolver(
      carrierSettlementControlFormSchema,
    ) as Resolver<CarrierSettlementControlFormValues>,
    defaultValues: {
      payTrigger: data.payTrigger,
      payPeriodFrequency: data.payPeriodFrequency,
      periodEndDayOfWeek: data.periodEndDayOfWeek,
      payDelayDays: data.payDelayDays,
      autoGenerateBatches: data.autoGenerateBatches,
      autoPostOnApprove: data.autoPostOnApprove,
      varianceTolerance: data.varianceToleranceMinor / 100,
      defaultApAccountId: data.defaultApAccountId ?? null,
      defaultPurchasedTransportationAccountId: data.defaultPurchasedTransportationAccountId ?? null,
    },
  });
  const { handleSubmit, reset } = form;

  const mutation = useApiMutation({
    mutationFn: (values: CarrierSettlementControlFormValues) =>
      updateCarrierSettlementControl({
        version: data.version,
        payTrigger: values.payTrigger,
        payPeriodFrequency: values.payPeriodFrequency,
        periodEndDayOfWeek: values.periodEndDayOfWeek,
        payDelayDays: values.payDelayDays,
        autoGenerateBatches: values.autoGenerateBatches,
        autoPostOnApprove: values.autoPostOnApprove,
        varianceToleranceMinor: Math.round(values.varianceTolerance * 100),
        defaultApAccountId: values.defaultApAccountId || undefined,
        defaultPurchasedTransportationAccountId:
          values.defaultPurchasedTransportationAccountId || undefined,
      }),
    onSuccess: (_, values) => {
      toast.success("Carrier settlement control updated");
      reset(values);
      void queryClient.invalidateQueries({ queryKey: ["carrier-settlement-control"] });
    },
    form,
    resourceName: "Carrier Settlement Control",
  });

  const onSubmit = useCallback(
    (values: CarrierSettlementControlFormValues) => mutation.mutate(values),
    [mutation],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <PayPeriodCard />
          <WorkflowCard />
          <MatchingCard />
          <PostingAccountsCard />
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function PayPeriodCard() {
  const { control } = useFormContext<CarrierSettlementControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>Pay Period</CardTitle>
        <CardDescription>
          Defines the carrier settlement cycle and when purchased-transportation cost accrues.
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
              description="How often carriers are settled — weekly is the industry norm for brokered freight."
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
              description="The shipment milestone at which carrier cost accrues into the settlement pool."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function WorkflowCard() {
  const { control } = useFormContext<CarrierSettlementControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>Workflow Automation</CardTitle>
        <CardDescription>
          Automate the routine AP run so reviewers focus on exceptions.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="autoGenerateBatches"
              label="Auto-Generate Batches"
              description="Generate a carrier settlement batch automatically when each pay period closes."
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
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function MatchingCard() {
  const { control } = useFormContext<CarrierSettlementControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>Invoice Matching</CardTitle>
        <CardDescription>
          Carrier invoices are compared to the negotiated buy rate; gaps beyond the tolerance flag a
          variance.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="varianceTolerance"
              label="Variance Tolerance"
              sideText="USD"
              decimalScale={2}
              fixedDecimalScale
              description="Invoice totals within this amount of the buy rate auto-match; larger gaps flag a variance for review."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function PostingAccountsCard() {
  const { control } = useFormContext<CarrierSettlementControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>Posting Accounts</CardTitle>
        <CardDescription>
          GL defaults for carrier settlement postings — posting debits purchased transportation and
          credits accounts payable; blank falls back to the accounting control defaults.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <GLAccountAutocompleteField
              control={control}
              name="defaultApAccountId"
              label="Accounts Payable Account"
              clearable
              description="The AP account credited when a settlement posts and debited when it is paid."
            />
          </FormControl>
          <FormControl>
            <GLAccountAutocompleteField
              control={control}
              name="defaultPurchasedTransportationAccountId"
              label="Purchased Transportation Account"
              clearable
              description="The expense account debited for carrier cost when a settlement posts."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
