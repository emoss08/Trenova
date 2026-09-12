import { useT } from "@trenova/shared/i18n/use-t";
import {
  DocumentTypeMultiSelectField,
  FuelSurchargeProgramAutocompleteField,
  GLAccountAutocompleteField,
  UserAutocompleteField,
} from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Separator } from "@trenova/shared/components/ui/separator";
import {
  billingCycleChoices,
  creditStatusChoices,
  currencyChoices,
  customerFuelSurchargeModeChoices,
  customerPaymentTermChoices,
  invoiceAdjustmentSupportingDocumentPolicyChoices,
  invoiceDeliveryChoices,
  invoiceDetailChoices,
  invoiceNumberFormatChoices,
  invoiceSectionKeyChoices,
  invoiceSplitKeyChoices,
  timezoneChoices,
} from "@/lib/choices";
import { BillingSchedulePreview } from "./billing-schedule-preview";
import type { Customer } from "@trenova/shared/types/customer";
import {
  BanknoteIcon,
  ClockIcon,
  CreditCardIcon,
  FileTextIcon,
  FuelIcon,
  GavelIcon,
  MailCheckIcon,
  SettingsIcon,
  StickyNoteIcon,
  UserCheckIcon,
  TruckIcon,
} from "lucide-react";
import { useMemo } from "react";
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
      <div className="bg-primary/10 text-primary flex size-8 shrink-0 items-center justify-center rounded-lg">
        <Icon className="size-4" />
      </div>
      <div>
        <h3 className="text-sm leading-none font-semibold tracking-tight">{title}</h3>
        <p className="text-muted-foreground mt-1 text-xs">{description}</p>
      </div>
    </div>
  );
}

export function CustomerBillingProfileForm() {
  const t = useT();

  const { control } = useFormContext<Customer>();

  const creditStatus = useWatch({
    control,
    name: "billingProfile.creditStatus",
  });
  const invoiceNumberFormat = useWatch({
    control,
    name: "billingProfile.invoiceNumberFormat",
  });
  const applyLateCharges = useWatch({
    control,
    name: "billingProfile.applyLateCharges",
  });
  const taxExempt = useWatch({ control, name: "billingProfile.taxExempt" });
  const fuelSurchargeMode = useWatch({ control, name: "billingProfile.fuelSurchargeMode" });

  const invoiceDelivery = useWatch({ control, name: "billingProfile.invoiceDelivery" });
  const billingCycle = useWatch({ control, name: "billingProfile.billingCycle" });
  const billingCycleAnchorDay = useWatch({
    control,
    name: "billingProfile.billingCycleAnchorDay",
  });
  const splitBy = useWatch({ control, name: "billingProfile.splitBy" });
  const sectionBy = useWatch({ control, name: "billingProfile.sectionBy" });
  const invoiceDetail = useWatch({ control, name: "billingProfile.invoiceDetail" });
  const maxShipmentsPerInvoice = useWatch({
    control,
    name: "billingProfile.maxShipmentsPerInvoice",
  });

  const showCreditHoldReason = creditStatus === "Hold" || creditStatus === "Suspended";
  const showCustomPrefix = invoiceNumberFormat === "CustomPrefix";
  const isConsolidated = invoiceDelivery === "Consolidated";
  const isWeeklyCycle = billingCycle === "Weekly" || billingCycle === "BiWeekly";

  const schedulePreview = useMemo(
    () => ({
      invoiceDelivery: invoiceDelivery ?? "PerShipment",
      billingCycle: billingCycle ?? "Immediate",
      billingCycleAnchorDay: billingCycleAnchorDay ?? 1,
      splitBy: splitBy ?? "Customer",
      sectionBy: sectionBy ?? "Shipment",
      invoiceDetail: invoiceDetail ?? "Detailed",
      maxShipmentsPerInvoice: maxShipmentsPerInvoice ?? 0,
    }),
    [
      invoiceDelivery,
      billingCycle,
      billingCycleAnchorDay,
      splitBy,
      sectionBy,
      invoiceDetail,
      maxShipmentsPerInvoice,
    ],
  );

  return (
    <div className="space-y-6">
      <SectionHeader
        icon={ClockIcon}
        title={t("Payment Terms")}
        description={t("How long this customer has to pay, and in what currency")}
      />
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            name="billingProfile.paymentTerm"
            label={t("Payment Term")}
            description={t("The number of days this customer has to pay after an invoice is issued. Overrides the organization default when set.")}
            options={customerPaymentTermChoices}
          />
        </FormControl>
        <FormControl cols="full">
          <SelectField
            control={control}
            name="billingProfile.billingCurrency"
            label={t("Currency")}
            description={t("Currency used on all invoices for this customer. Determines how amounts are formatted and displayed on billing documents.")}
            options={currencyChoices}
          />
        </FormControl>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="billingProfile.hasBillingControlOverrides"
            label={t("Override Global Billing Settings")}
            description={t("When enabled, this customer's billing profile takes precedence over your organization's global billing control settings.")}
            outlined
          />
        </FormControl>
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={CreditCardIcon}
        title={t("Credit Management")}
        description={t("Set credit limits and automatic hold rules to manage financial exposure")}
      />
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            name="billingProfile.creditStatus"
            label={t("Credit Status")}
            description={t("Reflects this customer's current creditworthiness. 'Hold' and 'Suspended' block new shipments from being dispatched.")}
            options={creditStatusChoices}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="billingProfile.creditLimit"
            label={t("Credit Limit")}
            placeholder="0.00"
            description={t("Maximum outstanding balance allowed before shipments are blocked. Leave empty for unlimited credit.")}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="billingProfile.creditBalance"
            label={t("Outstanding Balance")}
            placeholder="0.00"
            description={t("Current unpaid invoice total. Automatically updated as invoices are generated and payments received.")}
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="billingProfile.enforceCreditLimit"
            label={t("Enforce Credit Limit")}
            description={t("When enabled, the system will prevent new shipments from being created once the outstanding balance exceeds the credit limit.")}
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="billingProfile.autoCreditHold"
            label={t("Auto Credit Hold")}
            description={t("Automatically change credit status to 'Hold' when the outstanding balance exceeds the credit limit, without requiring manual intervention.")}
          />
        </FormControl>
        {showCreditHoldReason && (
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="billingProfile.creditHoldReason"
              label={t("Hold Reason")}
              placeholder={t("e.g., Past due on Invoice #1234, awaiting payment...")}
              description={t("Document why this customer is on hold or suspended. This is visible to dispatch and billing staff when they attempt to create shipments.")}
            />
          </FormControl>
        )}
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={FileTextIcon}
        title={t("Invoice Configuration")}
        description={t("Control how invoices are formatted, numbered, and which GL accounts they post to")}
      />
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            name="billingProfile.invoiceNumberFormat"
            label={t("Invoice Number Format")}
            description={t("How invoice numbers are generated. 'Custom Prefix' prepends a customer-specific string; 'PO Based' uses the customer's PO number as the invoice identifier.")}
            options={invoiceNumberFormatChoices}
          />
        </FormControl>
        {showCustomPrefix && (
          <FormControl>
            <InputField
              control={control}
              name="billingProfile.customerInvoicePrefix"
              label={t("Invoice Prefix")}
              placeholder={t("e.g., ACME-")}
              description={t("Custom string prepended to all invoice numbers for this customer, useful when customers require a specific format for their AP system.")}
            />
          </FormControl>
        )}
        <FormControl>
          <NumberField
            control={control}
            name="billingProfile.invoiceCopies"
            label={t("Invoice Copies")}
            placeholder="1"
            description={t("Number of invoice copies to generate per billing run. Additional copies are often required for customers with multiple AP departments.")}
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="billingProfile.autoSendInvoiceOnGeneration"
            label={t("Auto-Send After PDF Generation")}
            description={t("Email the invoice to the configured recipients after the invoice PDF is generated.")}
          />
        </FormControl>
        <FormControl>
          <GLAccountAutocompleteField
            control={control}
            name="billingProfile.revenueAccountId"
            label={t("Revenue Account")}
            description={t("GL account where revenue from this customer's shipments is posted. Overrides the organization default revenue account.")}
            clearable
          />
        </FormControl>
        <FormControl>
          <GLAccountAutocompleteField
            control={control}
            name="billingProfile.arAccountId"
            label={t("Accounts Receivable")}
            description={t("GL account for tracking this customer's outstanding invoices. Overrides the organization default AR account.")}
            clearable
          />
        </FormControl>
        <FormControl cols="full">
          <DocumentTypeMultiSelectField
            control={control}
            name="billingProfile.documentTypes"
            label={t("Required Document Types")}
            description={t("Documents that must be attached before an invoice can be generated (e.g., signed BOL, proof of delivery). Missing documents will block billing.")}
          />
        </FormControl>
      </FormGroup>

      <Separator />

      <SectionHeader
        icon={MailCheckIcon}
        title={t("Billing Schedule")}
        description={t("How this customer's shipments turn into invoices, and how often")}
      />
      <FormGroup cols={1}>
        <FormControl>
          <SelectField
            control={control}
            name="billingProfile.invoiceDelivery"
            label={t("Invoice Delivery")}
            description={t("Whether this customer gets an invoice per shipment, per order, or one statement covering a billing period.")}
            options={invoiceDeliveryChoices}
          />
        </FormControl>

        {isConsolidated && (
          <div className="flex flex-col gap-4 pl-4 border-l">
            <FormGroup cols={2}>
              <FormControl>
                <SelectField
                  control={control}
                  name="billingProfile.billingCycle"
                  label={t("Bill Every")}
                  description={t("How often the period closes and invoices are produced.")}
                  options={billingCycleChoices}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="billingProfile.billingCycleAnchorDay"
                  label={isWeeklyCycle ? "Day of Week" : "Day of Month"}
                  placeholder={isWeeklyCycle ? "0-6" : "1-28"}
                  description={
                    isWeeklyCycle
                      ? "0 = Sunday through 6 = Saturday."
                      : "Capped at 28 so a monthly cycle never shifts in February."
                  }
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  name="billingProfile.billingCycleTimezone"
                  label={t("Time Zone")}
                  description={t("The zone period boundaries are evaluated in. Getting this wrong moves loads into the wrong period.")}
                  options={timezoneChoices}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="billingProfile.consolidationLookbackDays"
                  label={t("Lookback")}
                  placeholder="30"
                  sideText="days"
                  description={t("How far before the period start to sweep shipments that were approved late.")}
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  name="billingProfile.splitBy"
                  label={t("Separate Invoice For Each")}
                  description={t("Decides how many invoices a period produces. Choose nothing for a single statement.")}
                  options={invoiceSplitKeyChoices}
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  name="billingProfile.sectionBy"
                  label={t("Group Lines By")}
                  description={t("How the lines inside each invoice are organised. Never changes how many invoices there are.")}
                  options={invoiceSectionKeyChoices}
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  name="billingProfile.invoiceDetail"
                  label={t("Detail Level")}
                  description={t("Whether each section lists every charge or collapses to one line per shipment.")}
                  options={invoiceDetailChoices}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="billingProfile.maxShipmentsPerInvoice"
                  label={t("Max Shipments Per Invoice")}
                  placeholder="0"
                  description={t("Splits an oversized group across several invoices. Zero means no limit.")}
                />
              </FormControl>
              <FormControl cols="full">
                <NumberField
                  control={control}
                  name="billingProfile.minConsolidatedAmount"
                  label={t("Minimum Invoice Amount")}
                  placeholder="0.00"
                  description={t("A group worth less than this defers to the next period instead of billing.")}
                />
              </FormControl>
            </FormGroup>
          </div>
        )}

        <BillingSchedulePreview schedule={schedulePreview} />
      </FormGroup>

      <Separator />

      <SectionHeader
        icon={BanknoteIcon}
        title={t("Late Charges & Tax")}
        description={t("Configure penalty rates for overdue invoices and tax exemption status")}
      />
      <FormGroup cols={1}>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.applyLateCharges"
            label={t("Apply Late Charges")}
            description={t("Automatically assess a percentage-based late fee on invoices that remain unpaid past the grace period.")}
            position="left"
          />
        </FormControl>
        {applyLateCharges && (
          <div className="flex flex-col pl-10">
            <FormControl className="min-h-[3em] max-w-[400px]">
              <NumberField
                control={control}
                name="billingProfile.lateChargeRate"
                label={t("Late Charge Rate")}
                placeholder="1.50"
                sideText="%"
                description={t("Monthly percentage applied to the overdue balance after the grace period expires.")}
              />
            </FormControl>
            <FormControl className="min-h-[3em] max-w-[400px]">
              <NumberField
                control={control}
                name="billingProfile.gracePeriodDays"
                label={t("Grace Period")}
                placeholder="0"
                sideText="days"
                description={t("Number of days after the invoice due date before late charges begin accruing.")}
              />
            </FormControl>
          </div>
        )}
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.taxExempt"
            label={t("Tax Exempt")}
            description={t("Mark this customer as exempt from sales tax. You must have a valid tax exemption certificate on file.")}
            position="left"
          />
        </FormControl>
        {taxExempt && (
          <FormControl className="min-h-[3em] max-w-[400px] pl-10">
            <InputField
              control={control}
              name="billingProfile.taxExemptNumber"
              label={t("Exemption Certificate Number")}
              placeholder={t("e.g., EX-2024-00123")}
              description={t("The customer's tax exemption certificate or resale number, required for audit compliance.")}
            />
          </FormControl>
        )}
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={FuelIcon}
        title={t("Fuel Surcharge")}
        description={t("How fuel is billed for this customer — not everyone uses a fuel table, so pick the arrangement that matches the contract")}
      />
      <FormGroup cols={1}>
        <FormControl className="min-h-[3em] max-w-[400px]">
          <SelectField
            control={control}
            name="billingProfile.fuelSurchargeMode"
            label={t("Fuel Billing")}
            options={customerFuelSurchargeModeChoices}
            description={t("Choose how fuel costs are recovered from this customer.")}
          />
        </FormControl>
        {fuelSurchargeMode === "Program" && (
          <FormControl className="min-h-[3em] max-w-[400px] pl-10">
            <FuelSurchargeProgramAutocompleteField
              control={control}
              name="billingProfile.fuelSurchargeProgramId"
              label={t("Fuel Surcharge Program")}
              placeholder={t("Select a program")}
              rules={{ required: true }}
              description={t("Every shipment for this customer gets the correct week's indexed surcharge from this program automatically.")}
            />
          </FormControl>
        )}
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={SettingsIcon}
        title={t("Billing Automation")}
        description={t("Control which steps in the billing pipeline happen automatically vs. requiring manual action")}
      />
      <FormGroup cols={1}>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.autoTransfer"
            label={t("Auto-Transfer to Billing")}
            description={t("Automatically move completed shipments from operations into the billing queue without manual handoff.")}
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.autoMarkReadyToBill"
            label={t("Auto-Mark Ready to Bill")}
            description={t("Automatically flag transferred shipments as 'Ready to Bill' once all required documents and validations are satisfied.")}
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.autoBill"
            label={t("Auto-Generate Invoices")}
            description={t("Automatically create invoices for shipments marked as ready to bill, removing the need for a billing clerk to manually trigger invoice generation.")}
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.autoApplyAccessorials"
            label={t("Auto-Apply Accessorial Charges")}
            description={t("Automatically add applicable accessorial charges (fuel surcharge, detention, liftgate, etc.) to shipments based on service rules.")}
            position="left"
          />
        </FormControl>
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={GavelIcon}
        title={t("Billing Requirements")}
        description={t("Enforce documentation and validation rules before shipments can be billed")}
      />
      <FormGroup cols={1}>
        <FormControl className="min-h-[3em]">
          <SelectField
            control={control}
            name="billingProfile.invoiceAdjustmentSupportingDocumentPolicy"
            label={t("Invoice Adjustment Supporting Documents")}
            description={t("Controls whether supporting documents are required for this customer's invoice adjustments. 'Inherit Organization Default' uses the organization invoice-adjustment policy.")}
            options={invoiceAdjustmentSupportingDocumentPolicyChoices}
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.enforceCustomerBillingReq"
            label={t("Enforce Customer Billing Requirements")}
            description={t("Require that all customer-mandated documentation, reference numbers, and special instructions are present before a shipment can enter the billing queue.")}
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.validateCustomerRates"
            label={t("Validate Rates Against Contracts")}
            description={t("Cross-check all applied rates against this customer's contracted rate agreements before invoicing. Mismatches will block billing and flag for review.")}
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.requirePONumber"
            label={t("Require PO Number")}
            description={t("Shipments cannot be billed without a customer purchase order number. Ensures the customer's AP department can match invoices to approved POs.")}
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.requireBOLNumber"
            label={t("Require BOL Number")}
            description={t("Shipments must have a bill of lading number before they can be invoiced. Required by many customers for freight payment verification.")}
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="billingProfile.requireDeliveryNumber"
            label={t("Require Delivery Number")}
            description={t("A delivery confirmation number must be recorded before the shipment can move to billing. Commonly required for retail and distribution customers.")}
            position="left"
          />
        </FormControl>
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={TruckIcon}
        title={t("Stop Performance")}
        description={t("How appointment scheduling affects late and detention evaluation")}
      />
      <FormGroup cols={1}>
        <FormControl className="min-h-[3em] max-w-[500px]">
          <SwitchField
            control={control}
            name="billingProfile.countLateOnlyOnAppointmentStops"
            label={t("Count Late Only on Appointment Stops")}
            description={t("When enabled, late-performance evaluation only applies to stops explicitly marked as appointment stops. Open stops remain operationally scheduled but do not count as late exceptions.")}
            position="left"
          />
        </FormControl>
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={UserCheckIcon}
        title={t("Default Biller")}
        description={t("Assign a default biller for this customer. New billing queue items will be auto-assigned to this user.")}
      />
      <FormGroup cols={1}>
        <FormControl>
          <UserAutocompleteField
            control={control}
            name="billingProfile.defaultBillerId"
            label={t("Default Biller")}
            description={t("When shipments for this customer are transferred to the billing queue, they will be automatically assigned to this biller.")}
          />
        </FormControl>
      </FormGroup>
      <Separator />
      <SectionHeader
        icon={StickyNoteIcon}
        title={t("Billing Notes")}
        description={t("Internal notes visible to billing staff when processing this customer's invoices")}
      />
      <FormGroup cols={1}>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="billingProfile.billingNotes"
            label={t("Notes")}
            placeholder={t("e.g., Customer requires invoices sent to AP@acme.com with PO reference in subject line...")}
            description={t("Free-form notes for your billing team. These are not printed on invoices — use the email profile tab for customer-facing communication settings.")}
          />
        </FormControl>
      </FormGroup>
    </div>
  );
}
