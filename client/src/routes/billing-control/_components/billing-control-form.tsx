import { BetaTag } from "@/components/beta-tag";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormSaveDock } from "@/components/form-save-dock";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { paymentTermChoices, transferScheduleChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { BillingControl } from "@/types/billing-control";
import { billingControlSchema } from "@/types/billing-control";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import {
  FormProvider,
  type Resolver,
  useForm,
  useFormContext,
  useWatch,
} from "react-hook-form";

export default function BillingControlForm() {
  const { data } = useSuspenseQuery({
    ...queries.billingControl.get(),
  });

  const form = useForm<BillingControl>({
    resolver: zodResolver(billingControlSchema) as Resolver<BillingControl>,
    defaultValues: data,
  });

  const { handleSubmit, setError, reset } = form;

  const { mutateAsync } = useOptimisticMutation<
    BillingControl,
    BillingControl,
    unknown,
    BillingControl
  >({
    queryKey: queries.billingControl.get._def,
    mutationFn: async (values: BillingControl) =>
      apiService.billingControlService.update(values),
    resourceName: "Billing Control",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [queries.billingControl.get._def],
  });

  const onSubmit = useCallback(
    async (values: BillingControl) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <InvoiceSettingsForm />
          <BillingValidationSettings />
          <TransferSettingsForm />
          <BillingAutomationForm />
          <ConsolidationSettingsForm />
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function InvoiceSettingsForm() {
  const { control } = useFormContext<BillingControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Invoice Document Configuration</CardTitle>
        <CardDescription>
          Define how invoices are formatted, what information they contain, and
          how they are presented to customers. These settings determine the
          professional appearance and content of your billing documents,
          ensuring clarity and consistency while facilitating prompt payment
          processing and maintaining compliance with financial documentation
          standards.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="invoiceNumberPrefix"
              rules={{ required: true }}
              maxLength={10}
              label="Invoice Number Prefix"
              placeholder="Enter the prefix for the invoice number"
              description="Establishes the standardized identifier that precedes the sequential number in all invoices."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="creditMemoNumberPrefix"
              rules={{ required: true }}
              maxLength={10}
              label="Credit Memo Number Prefix"
              placeholder="Enter the prefix for the credit memo number"
              description="Defines the standardized identifier that precedes the sequential number in all credit memos."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="showInvoiceDueDate"
              label="Show Invoice Due Date"
              description="When enabled, the payment due date is prominently displayed on all invoices."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="showAmountDue"
              label="Show Amount Due"
              description="When enabled, the total amount due is prominently displayed on all invoices."
            />
          </FormControl>
          <FormControl cols="full">
            <SelectField
              control={control}
              rules={{ required: true }}
              name="paymentTerm"
              label="Default Payment Terms"
              description="Establishes the standard timeframe for customer payment that applies when no specific terms have been negotiated."
              options={paymentTermChoices}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="invoiceTerms"
              label="Invoice Terms & Conditions"
              placeholder="Invoice Terms & Conditions"
              description="Establishes the legally binding payment conditions, grace periods, penalties for late payment, and other contractual stipulations that appear on all invoices."
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="invoiceFooter"
              label="Invoice Footer"
              placeholder="Invoice Footer"
              description="Defines supplementary information displayed at the bottom of all invoices, including company contact details, payment methods, electronic remittance instructions, and legal notices."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function TransferSettingsForm() {
  const { control } = useFormContext<BillingControl>();
  const autoTransfer = useWatch({ control, name: "autoTransfer" });

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          Operational-to-Financial Transfer Gateway <BetaTag />
        </CardTitle>
        <CardDescription>
          Define the criteria that govern when completed shipments transition
          from operational status to financial processing. This critical gateway
          bridges your operational and accounting systems, ensuring that only
          properly documented and validated shipments enter your revenue cycle.
          Effective transfer controls accelerate revenue recognition while
          maintaining billing accuracy and compliance with customer-specific
          requirements.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="autoMarkReadyToBill"
              label="Automate Ready-to-Bill Designation"
              description="When enabled, shipments that satisfy all transfer criteria are automatically flagged as 'Ready to Bill' without requiring manual verification."
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="autoTransfer"
              label="Enable Automatic Transfers"
              description="When enabled, shipments that satisfy all transfer criteria are automatically transferred to the billing system without requiring manual verification."
              position="left"
            />
          </FormControl>
          {autoTransfer && (
            <>
              <FormControl className="pl-10">
                <NumberField
                  control={control}
                  rules={{ required: autoTransfer }}
                  name="transferBatchSize"
                  label="Transfer Batch Size"
                  placeholder="Enter maximum number of shipments per batch"
                  description="Defines the maximum number of shipments processed in a single transfer operation, optimizing system performance by balancing transfer efficiency with resource utilization while preventing processing bottlenecks during high-volume periods."
                />
              </FormControl>
              <FormControl className="pl-10">
                <SelectField
                  control={control}
                  rules={{ required: autoTransfer }}
                  name="transferSchedule"
                  label="Transfer Schedule"
                  description="Establishes when automated transfers from operations to billing occur, balancing timely revenue recognition with operational efficiency and system resource optimization."
                  options={transferScheduleChoices}
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function BillingAutomationForm() {
  const { control } = useFormContext<BillingControl>();
  const autoBill = useWatch({ control, name: "autoBill" });

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          Invoice Generation Automation <BetaTag />
        </CardTitle>
        <CardDescription>
          Configure the intelligent automation system that determines when
          shipments are converted into customer invoices without manual
          intervention. This autonomous billing framework minimizes human
          touchpoints in the revenue cycle, reduces days-to-invoice metrics, and
          ensures consistent application of billing practices across your
          organization.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="autoBill"
              label="Enable Autonomous Invoice Generation"
              description="When enabled, the system will automatically convert qualified shipments into finalized invoices without manual review when predefined criteria are met."
              position="left"
            />
          </FormControl>
          {autoBill && (
            <>
              <FormControl className="pl-10">
                <SwitchField
                  control={control}
                  name="sendAutoBillNotifications"
                  label="Send Automated Billing Notifications"
                  description="When enabled, the system automatically notifies relevant stakeholders when invoices are generated through the automated billing process."
                  position="left"
                />
              </FormControl>
              <FormControl className="pl-10">
                <NumberField
                  control={control}
                  name="autoBillBatchSize"
                  rules={{ required: autoBill }}
                  label="Automated Billing Batch Size"
                  placeholder="Enter maximum invoices per batch"
                  description="Establishes the maximum number of invoices generated in a single automated billing run, optimizing system performance by balancing processing efficiency with resource utilization and preventing system slowdowns during high-volume periods."
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function BillingValidationSettings() {
  const { control } = useFormContext<BillingControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Pre-Billing Validation Framework</CardTitle>
        <CardDescription>
          Configure the comprehensive validation checks that shipments must pass
          before entering the invoicing process. These validation controls
          prevent billing errors, ensure compliance with customer-specific
          requirements, and verify rate accuracy before invoices are generated.
          A robust validation framework minimizes billing disputes, accelerates
          payment collection, and maintains strong customer relationships by
          ensuring billing accuracy and contractual compliance.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="enforceCustomerBillingReq"
              label="Enforce Customer-Specific Billing Requirements"
              description="When enabled, the system verifies that all customer-mandated documentation, reference numbers, and special handling instructions are fulfilled before allowing shipment transfer to billing."
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="validateCustomerRates"
              label="Validate Contractual Rate Compliance"
              description="When enabled, the system compares all applied charges against authorized customer rate agreements before allowing transfer to billing."
              position="left"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ConsolidationSettingsForm() {
  const { control } = useFormContext<BillingControl>();
  const allowInvoiceConsolidation = useWatch({
    control,
    name: "allowInvoiceConsolidation",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Invoice Consolidation & Grouping Strategy</CardTitle>
        <CardDescription>
          Define how multiple shipments and services are combined into unified
          invoices for your customers. Effective consolidation strategies reduce
          billing administrative costs, minimize the volume of payment
          transactions, and provide customers with comprehensive invoices that
          align with their accounting preferences and payment processing
          capabilities.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="allowInvoiceConsolidation"
              label="Allow Invoice Consolidation"
              description="Allow combining multiple shipments into a single invoice."
              position="left"
            />
          </FormControl>
          {allowInvoiceConsolidation && (
            <div className="flex flex-col pl-10">
              <FormControl className="min-h-[3em] max-w-[400px]">
                <NumberField
                  control={control}
                  name="consolidationPeriodDays"
                  label="Consolidation Period"
                  description="Number of days to consolidate shipments into a single invoice."
                  placeholder="7"
                  sideText="days"
                  rules={{ required: allowInvoiceConsolidation }}
                />
              </FormControl>
              <FormControl className="min-h-[3em]">
                <SwitchField
                  className="px-0!"
                  control={control}
                  name="groupConsolidatedInvoices"
                  label="Group Line Items"
                  description="Group line items by service type in consolidated invoices."
                  position="left"
                />
              </FormControl>
            </div>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}
