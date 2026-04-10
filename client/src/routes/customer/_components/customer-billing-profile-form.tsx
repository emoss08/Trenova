import {
  DocumentTypeMultiSelectField,
  GLAccountAutocompleteField,
  UserAutocompleteField,
} from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import {
  billingCycleTypeChoices,
  consolidationGroupByChoices,
  creditStatusChoices,
  currencyChoices,
  customerPaymentTermChoices,
  invoiceAdjustmentSupportingDocumentPolicyChoices,
  invoiceMethodChoices,
  invoiceNumberFormatChoices,
} from "@/lib/choices";
import type { Customer } from "@/types/customer";
import {
  BanknoteIcon,
  ClockIcon,
  CreditCardIcon,
  FileTextIcon,
  GavelIcon,
  MailCheckIcon,
  SettingsIcon,
  StickyNoteIcon,
  UserCheckIcon,
  TruckIcon,
} from "lucide-react";
import { useFormContext, useWatch } from "react-hook-form";

function SectionHeader({
  icon: Icon,
  title,
  description,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
}) {
  return (
    <div className="flex items-center gap-3">
      <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
        <Icon className="size-4" />
      </div>
      <div>
        <h3 className="text-sm leading-none font-semibold tracking-tight">{title}</h3>
        <p className="mt-1 text-xs text-muted-foreground">{description}</p>
      </div>
    </div>
  );
}

export function CustomerBillingProfileForm() {
  const { control } = useFormContext<Customer>();

  const billingCycleType = useWatch({
    control,
    name: "billingProfile.billingCycleType",
  });
  const creditStatus = useWatch({
    control,
    name: "billingProfile.creditStatus",
  });
  const invoiceNumberFormat = useWatch({
    control,
    name: "billingProfile.invoiceNumberFormat",
  });
  const allowInvoiceConsolidation = useWatch({
    control,
    name: "billingProfile.allowInvoiceConsolidation",
  });
  const applyLateCharges = useWatch({
    control,
    name: "billingProfile.applyLateCharges",
  });
  const taxExempt = useWatch({ control, name: "billingProfile.taxExempt" });
  const detentionBillingEnabled = useWatch({
    control,
    name: "billingProfile.detentionBillingEnabled",
  });

  const showDayOfWeek = billingCycleType === "Weekly" || billingCycleType === "BiWeekly";
  const showCreditHoldReason = creditStatus === "Hold" || creditStatus === "Suspended";
  const showCustomPrefix = invoiceNumberFormat === "CustomPrefix";

  return (
    <div className="space-y-6">
      <SectionHeader
        icon={ClockIcon}
        title="Billing Cycle & Payment"
        description="Controls when invoices are generated and how long customers have to pay"
      />
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            name="billingProfile.billingCycleType"
            label="Billing Cycle"
            description="Determines invoice generation frequency. 'Immediate' creates an invoice per shipment; 'Monthly' batches all shipments into one monthly invoice."
            options={billingCycleTypeChoices}
          />
        </FormControl>
        {showDayOfWeek && (
          <FormControl>
            <NumberField
              control={control}
              name="billingProfile.billingCycleDayOfWeek"
              label="Day of Week"
              placeholder="0-6"
              description="Which day invoices are generated (0 = Sunday through 6 = Saturday). Only applies to weekly and bi-weekly cycles."
            />
          </FormControl>
        )}
        <FormControl>
          <SelectField
            control={control}
            name="billingProfile.paymentTerm"
            label="Payment Term"
            description="The number of days this customer has to pay after an invoice is issued. Overrides the organization default when set."
            options={customerPaymentTermChoices}
          />
        </FormControl>
        <FormControl cols="full">
          <SelectField
            control={control}
            name="billingProfile.billingCurrency"
            label="Currency"
            description="Currency used on all invoices for this customer. Determines how amounts are formatted and displayed on billing documents."
            options={currencyChoices}
          />
        </FormControl>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="billingProfile.hasBillingControlOverrides"
            label="Override Global Billing Settings"
            description="When enabled, this customer's billing profile takes precedence over your organization's global billing control settings."
            outlined
          />
        </FormControl>
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={CreditCardIcon}
        title="Credit Management"
        description="Set credit limits and automatic hold rules to manage financial exposure"
      />
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            name="billingProfile.creditStatus"
            label="Credit Status"
            description="Reflects this customer's current creditworthiness. 'Hold' and 'Suspended' block new shipments from being dispatched."
            options={creditStatusChoices}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="billingProfile.creditLimit"
            label="Credit Limit"
            placeholder="0.00"
            description="Maximum outstanding balance allowed before shipments are blocked. Leave empty for unlimited credit."
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="billingProfile.creditBalance"
            label="Outstanding Balance"
            placeholder="0.00"
            description="Current unpaid invoice total. Automatically updated as invoices are generated and payments received."
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="billingProfile.enforceCreditLimit"
            label="Enforce Credit Limit"
            description="When enabled, the system will prevent new shipments from being created once the outstanding balance exceeds the credit limit."
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="billingProfile.autoCreditHold"
            label="Auto Credit Hold"
            description="Automatically change credit status to 'Hold' when the outstanding balance exceeds the credit limit, without requiring manual intervention."
          />
        </FormControl>
        {showCreditHoldReason && (
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="billingProfile.creditHoldReason"
              label="Hold Reason"
              placeholder="e.g., Past due on Invoice #1234, awaiting payment..."
              description="Document why this customer is on hold or suspended. This is visible to dispatch and billing staff when they attempt to create shipments."
            />
          </FormControl>
        )}
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={FileTextIcon}
        title="Invoice Configuration"
        description="Control how invoices are formatted, numbered, and which GL accounts they post to"
      />
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            name="billingProfile.invoiceMethod"
            label="Invoice Method"
            description="'Individual' creates one invoice per shipment. 'Summary' combines multiple shipments. 'Summary with Detail' includes line-level shipment breakdowns."
            options={invoiceMethodChoices}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="billingProfile.invoiceNumberFormat"
            label="Invoice Number Format"
            description="How invoice numbers are generated. 'Custom Prefix' prepends a customer-specific string; 'PO Based' uses the customer's PO number as the invoice identifier."
            options={invoiceNumberFormatChoices}
          />
        </FormControl>
        {showCustomPrefix && (
          <FormControl>
            <InputField
              control={control}
              name="billingProfile.customerInvoicePrefix"
              label="Invoice Prefix"
              placeholder="e.g., ACME-"
              description="Custom string prepended to all invoice numbers for this customer, useful when customers require a specific format for their AP system."
            />
          </FormControl>
        )}
        <FormControl>
          <NumberField
            control={control}
            name="billingProfile.invoiceCopies"
            label="Invoice Copies"
            placeholder="1"
            description="Number of invoice copies to generate per billing run. Additional copies are often required for customers with multiple AP departments."
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="billingProfile.summaryTransmitOnGeneration"
            label="Auto-Transmit on Generation"
            description="Automatically send summary invoices to the customer as soon as they are generated, without requiring manual review first."
          />
        </FormControl>
        <FormControl>
          <GLAccountAutocompleteField
            control={control}
            name="billingProfile.revenueAccountId"
            label="Revenue Account"
            description="GL account where revenue from this customer's shipments is posted. Overrides the organization default revenue account."
            clearable
          />
        </FormControl>
        <FormControl>
          <GLAccountAutocompleteField
            control={control}
            name="billingProfile.arAccountId"
            label="Accounts Receivable"
            description="GL account for tracking this customer's outstanding invoices. Overrides the organization default AR account."
            clearable
          />
        </FormControl>
        <FormControl cols="full">
          <DocumentTypeMultiSelectField
            control={control}
            name="billingProfile.documentTypes"
            label="Required Document Types"
            description="Documents that must be attached before an invoice can be generated (e.g., signed BOL, proof of delivery). Missing documents will block billing."
          />
        </FormControl>
      </FormGroup>

      <Separator />

      <SectionHeader
        icon={MailCheckIcon}
        title="Invoice Consolidation"
        description="Combine multiple shipments into fewer invoices to reduce AP processing overhead"
      />
      <FormGroup cols={1}>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.allowInvoiceConsolidation"
            label="Allow Invoice Consolidation"
            description="When enabled, shipments within the consolidation period are combined into a single invoice instead of being billed individually."
            position="left"
          />
        </FormControl>
        {allowInvoiceConsolidation && (
          <div className="flex flex-col pl-10">
            <FormControl className="min-h-[3em] max-w-[400px]">
              <NumberField
                control={control}
                name="billingProfile.consolidationPeriodDays"
                label="Consolidation Window"
                placeholder="7"
                sideText="days"
                description="How many days of shipments to batch into a single consolidated invoice."
              />
            </FormControl>
            <FormControl className="min-h-[3em] max-w-[400px]">
              <SelectField
                control={control}
                name="billingProfile.consolidationGroupBy"
                label="Group By"
                description="How line items are organized within a consolidated invoice. Grouping by location or PO number makes it easier for the customer to reconcile."
                options={consolidationGroupByChoices}
              />
            </FormControl>
          </div>
        )}
      </FormGroup>

      <Separator />

      <SectionHeader
        icon={BanknoteIcon}
        title="Late Charges & Tax"
        description="Configure penalty rates for overdue invoices and tax exemption status"
      />
      <FormGroup cols={1}>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.applyLateCharges"
            label="Apply Late Charges"
            description="Automatically assess a percentage-based late fee on invoices that remain unpaid past the grace period."
            position="left"
          />
        </FormControl>
        {applyLateCharges && (
          <div className="flex flex-col pl-10">
            <FormControl className="min-h-[3em] max-w-[400px]">
              <NumberField
                control={control}
                name="billingProfile.lateChargeRate"
                label="Late Charge Rate"
                placeholder="1.50"
                sideText="%"
                description="Monthly percentage applied to the overdue balance after the grace period expires."
              />
            </FormControl>
            <FormControl className="min-h-[3em] max-w-[400px]">
              <NumberField
                control={control}
                name="billingProfile.gracePeriodDays"
                label="Grace Period"
                placeholder="0"
                sideText="days"
                description="Number of days after the invoice due date before late charges begin accruing."
              />
            </FormControl>
          </div>
        )}
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.taxExempt"
            label="Tax Exempt"
            description="Mark this customer as exempt from sales tax. You must have a valid tax exemption certificate on file."
            position="left"
          />
        </FormControl>
        {taxExempt && (
          <FormControl className="min-h-[3em] max-w-[400px] pl-10">
            <InputField
              control={control}
              name="billingProfile.taxExemptNumber"
              label="Exemption Certificate Number"
              placeholder="e.g., EX-2024-00123"
              description="The customer's tax exemption certificate or resale number, required for audit compliance."
            />
          </FormControl>
        )}
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={SettingsIcon}
        title="Billing Automation"
        description="Control which steps in the billing pipeline happen automatically vs. requiring manual action"
      />
      <FormGroup cols={1}>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.autoTransfer"
            label="Auto-Transfer to Billing"
            description="Automatically move completed shipments from operations into the billing queue without manual handoff."
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.autoMarkReadyToBill"
            label="Auto-Mark Ready to Bill"
            description="Automatically flag transferred shipments as 'Ready to Bill' once all required documents and validations are satisfied."
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.autoBill"
            label="Auto-Generate Invoices"
            description="Automatically create invoices for shipments marked as ready to bill, removing the need for a billing clerk to manually trigger invoice generation."
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.autoApplyAccessorials"
            label="Auto-Apply Accessorial Charges"
            description="Automatically add applicable accessorial charges (fuel surcharge, detention, liftgate, etc.) to shipments based on service rules."
            position="left"
          />
        </FormControl>
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={GavelIcon}
        title="Billing Requirements"
        description="Enforce documentation and validation rules before shipments can be billed"
      />
      <FormGroup cols={1}>
        <FormControl className="min-h-[3em]">
          <SelectField
            control={control}
            name="billingProfile.invoiceAdjustmentSupportingDocumentPolicy"
            label="Invoice Adjustment Supporting Documents"
            description="Controls whether supporting documents are required for this customer's invoice adjustments. 'Inherit Organization Default' uses the organization invoice-adjustment policy."
            options={invoiceAdjustmentSupportingDocumentPolicyChoices}
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.enforceCustomerBillingReq"
            label="Enforce Customer Billing Requirements"
            description="Require that all customer-mandated documentation, reference numbers, and special instructions are present before a shipment can enter the billing queue."
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.validateCustomerRates"
            label="Validate Rates Against Contracts"
            description="Cross-check all applied rates against this customer's contracted rate agreements before invoicing. Mismatches will block billing and flag for review."
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.requirePONumber"
            label="Require PO Number"
            description="Shipments cannot be billed without a customer purchase order number. Ensures the customer's AP department can match invoices to approved POs."
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.requireBOLNumber"
            label="Require BOL Number"
            description="Shipments must have a bill of lading number before they can be invoiced. Required by many customers for freight payment verification."
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.requireDeliveryNumber"
            label="Require Delivery Number"
            description="A delivery confirmation number must be recorded before the shipment can move to billing. Commonly required for retail and distribution customers."
            position="left"
          />
        </FormControl>
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={TruckIcon}
        title="Detention Billing"
        description="Charge customers for excessive loading or unloading wait times"
      />
      <FormGroup cols={1}>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.detentionBillingEnabled"
            label="Enable Detention Billing"
            description="When enabled, detention charges are automatically calculated when driver wait time at pickup or delivery exceeds the free time allowance."
            position="left"
          />
        </FormControl>
        {detentionBillingEnabled && (
          <div className="flex flex-col pl-10">
            <FormControl className="min-h-[3em] max-w-[400px]">
              <NumberField
                control={control}
                name="billingProfile.detentionFreeMinutes"
                label="Free Time"
                placeholder="120"
                sideText="min"
                description="Minutes allowed at each stop before detention charges begin. Industry standard is typically 120 minutes (2 hours)."
              />
            </FormControl>
            <FormControl className="min-h-[3em] max-w-[400px]">
              <NumberField
                control={control}
                name="billingProfile.detentionRatePerHour"
                label="Hourly Rate"
                placeholder="75.00"
                sideText="$/hr"
                description="Dollar amount charged per hour of detention after free time is exhausted."
              />
            </FormControl>
          </div>
        )}
        <div className="flex flex-col pl-10">
          <FormControl className="min-h-[3em] max-w-[500px]">
            <SwitchField
              control={control}
              name="billingProfile.countLateOnlyOnAppointmentStops"
              label="Count Late Only on Appointment Stops"
              description="When enabled, late-performance evaluation only applies to stops explicitly marked as appointment stops. Open stops remain operationally scheduled but do not count as late exceptions."
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em] max-w-[500px]">
            <SwitchField
              control={control}
              name="billingProfile.countDetentionOnlyOnAppointmentStops"
              label="Count Detention Only on Appointment Stops"
              description="When enabled, detention calculations only apply to stops explicitly marked as appointment stops. Open stops will not accrue detention by default."
              position="left"
            />
          </FormControl>
        </div>
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={UserCheckIcon}
        title="Default Biller"
        description="Assign a default biller for this customer. New billing queue items will be auto-assigned to this user."
      />
      <FormGroup cols={1}>
        <FormControl>
          <UserAutocompleteField
            control={control}
            name="billingProfile.defaultBillerId"
            label="Default Biller"
            description="When shipments for this customer are transferred to the billing queue, they will be automatically assigned to this biller."
          />
        </FormControl>
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={StickyNoteIcon}
        title="Billing Notes"
        description="Internal notes visible to billing staff when processing this customer's invoices"
      />
      <FormGroup cols={1}>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="billingProfile.billingNotes"
            label="Notes"
            placeholder="e.g., Customer requires invoices sent to AP@acme.com with PO reference in subject line..."
            description="Free-form notes for your billing team. These are not printed on invoices — use the email profile tab for customer-facing communication settings."
          />
        </FormControl>
      </FormGroup>
    </div>
  );
}
