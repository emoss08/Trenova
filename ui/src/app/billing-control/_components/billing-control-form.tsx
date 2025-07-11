"use no memo";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormSaveDock } from "@/components/form";
import { BetaTag } from "@/components/ui/beta-tag";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import {
  billingExceptionHandlingChoices,
  paymentTermChoices,
  transferScheduleChoices,
} from "@/lib/choices";
import { queries } from "@/lib/queries";
import {
  BillingControlSchema,
  billingControlSchema,
} from "@/lib/schemas/billing-schema";
import { api } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useState } from "react";
import { FormProvider, useForm, useFormContext } from "react-hook-form";

export default function BillingControlForm() {
  const billingControl = useSuspenseQuery({
    ...queries.organization.getBillingControl(),
  });

  const form = useForm({
    resolver: zodResolver(billingControlSchema),
    defaultValues: billingControl.data,
  });

  const { handleSubmit, setError, reset } = form;
  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.organization.getBillingControl._def,
    mutationFn: async (values: BillingControlSchema) =>
      api.billingControl.update(values),
    successMessage: "Billing control updated successfully",
    resourceName: "Billing Control",
    setFormError: setError,
    resetForm: reset,
    invalidateQueries: [queries.organization.getShipmentControl._def],
  });

  const onSubmit = useCallback(
    async (values: BillingControlSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <InvoiceSettings />
          <BillingValidationSettings />
          <ShipmentTransferSettings />
          <AutomatedBillingSettings />
          <ExceptionHandlingSettings />
          <ConsolidationSettings />
          <FormSaveDock />
        </div>
      </Form>
    </FormProvider>
  );
}

function ShipmentTransferSettings() {
  const { control, watch } = useFormContext<BillingControlSchema>();

  const autoTransfer = watch("autoTransfer");

  const [showTransferCriteria, setShowTransferCriteria] =
    useState<boolean>(false);

  useEffect(() => {
    if (autoTransfer) {
      setShowTransferCriteria(true);
    } else {
      setShowTransferCriteria(false);
    }
  }, [autoTransfer]);

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
          {showTransferCriteria && (
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

function BillingValidationSettings() {
  const { control } = useFormContext<BillingControlSchema>();

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

function AutomatedBillingSettings() {
  const { control, watch } = useFormContext<BillingControlSchema>();
  const [showAutoBillCriteria, setShowAutoBillCriteria] =
    useState<boolean>(false);

  const autoBill = watch("autoBill");

  useEffect(() => {
    if (autoBill) {
      setShowAutoBillCriteria(true);
    } else {
      setShowAutoBillCriteria(false);
    }
  }, [autoBill]);

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
          {showAutoBillCriteria && (
            <>
              <FormControl className="pl-10">
                <SwitchField
                  control={control}
                  name="sendAutoBillNotifications"
                  label="Send Automated Billing Notifications"
                  description="When enabled, the system automatically notifies relevant stakeholders when invoices are generated through the automated billing process."
                  position="left"
                  size="sm"
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

function InvoiceSettings() {
  const { control } = useFormContext<BillingControlSchema>();

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

function ExceptionHandlingSettings() {
  const { control } = useFormContext<BillingControlSchema>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Billing Exception Management Framework</CardTitle>
        <CardDescription>
          Configure how the system identifies, routes, and resolves
          discrepancies and exceptions that occur during the billing process. A
          robust exception handling framework maintains billing accuracy and
          operational efficiency while preventing revenue leakage and ensuring
          that anomalies receive appropriate levels of scrutiny based on their
          financial impact and complexity.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl>
            <SelectField
              control={control}
              name="billingExceptionHandling"
              label="Exception Processing Strategy"
              description="Determines the workflow for managing billing exceptions, defining how discrepancies are routed, who receives notifications, and what approval thresholds apply."
              rules={{ required: true }}
              options={billingExceptionHandlingChoices}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="rateDiscrepancyThreshold"
              label="Rate Discrepancy Threshold"
              placeholder="Enter the rate discrepancy threshold"
              description="Establishes the monetary or percentage variance between quoted and applied rates that triggers exception handling workflows."
              rules={{ required: true, min: 0 }}
              sideText="%"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="autoResolveMinorDiscrepancies"
              label="Automatically Resolve Minor Discrepancies"
              description="When enabled, the system will automatically resolve rate variances below the defined threshold without manual intervention."
              position="left"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ConsolidationSettings() {
  const { control, watch } = useFormContext<BillingControlSchema>();

  const [showConsolidationOptions, setShowConsolidationOptions] =
    useState<boolean>(false);

  const allowInvoiceConsolidation = watch("allowInvoiceConsolidation");

  useEffect(() => {
    if (allowInvoiceConsolidation) {
      console.log("allowInvoiceConsolidation", allowInvoiceConsolidation);
      setShowConsolidationOptions(true);
    } else {
      console.log("allowInvoiceConsolidation", allowInvoiceConsolidation);
      setShowConsolidationOptions(false);
    }
  }, [allowInvoiceConsolidation]);

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
              position="left"
              label="Enable Invoice Consolidation"
              description="When enabled, multiple shipments for the same customer can be combined into a single invoice document."
            />
          </FormControl>
          {showConsolidationOptions && (
            <>
              <FormControl className="pl-10 min-h-[3em]">
                <SwitchField
                  control={control}
                  position="left"
                  name="groupConsolidatedInvoices"
                  label="Group by Service Category"
                  description="When enabled, consolidated invoices organize charges by service type or category rather than by individual shipment."
                  size="sm"
                />
              </FormControl>
              <FormControl className="pl-10 min-h-[3em]">
                <NumberField
                  control={control}
                  name="consolidationPeriodDays"
                  label="Consolidation Period Duration"
                  placeholder="Enter the consolidation period days"
                  description="Defines the timeframe (in days) during which shipments are grouped into a single invoice."
                  rules={{ required: true }}
                  sideText="days"
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}
