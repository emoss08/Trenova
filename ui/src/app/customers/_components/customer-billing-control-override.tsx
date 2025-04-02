import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup } from "@/components/ui/form";
import { paymentTermChoices, transferCriteriaChoices } from "@/lib/choices";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { useEffect, useState } from "react";
import { useFormContext } from "react-hook-form";

export function BillingControlOverrides() {
  const [showBillingControlOverrides, setShowBillingControlOverrides] =
    useState<boolean>(false);

  const { setValue, watch } = useFormContext<CustomerSchema>();

  // * Watch for existing value to initialize the state properly
  const hasOverrides = watch("billingProfile.hasOverrides");

  // * Initialize state from form value on component mount
  useEffect(() => {
    setShowBillingControlOverrides(!!hasOverrides);
  }, [hasOverrides]);

  const toggleBillingControlOverrides = (show: boolean) => {
    setShowBillingControlOverrides(show);
    setValue("billingProfile.hasOverrides", show, {
      shouldDirty: true,
      shouldTouch: true,
      shouldValidate: true,
    });
  };

  return (
    <div className="flex flex-col gap-4 border-t pt-4">
      <div className="flex items-center justify-between">
        <h3
          id="billing-control-overrides"
          className="font-semibold leading-none tracking-tight text-sm"
        >
          Customer-Specific Billing Control Overrides
        </h3>
        {showBillingControlOverrides && (
          <Button
            onClick={() => toggleBillingControlOverrides(false)}
            variant="destructive"
            type="button"
            size="sm"
          >
            Remove Override
          </Button>
        )}
      </div>
      {showBillingControlOverrides ? (
        <BillingControlOverridesForm />
      ) : (
        <Button
          onClick={() => toggleBillingControlOverrides(true)}
          className="w-full"
          variant="outline"
          type="button"
        >
          Add Billing Control Overrides
        </Button>
      )}
    </div>
  );
}

function BillingControlOverridesForm() {
  const { control } = useFormContext<CustomerSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl cols="full">
        <SelectField
          control={control}
          name="billingProfile.paymentTerm"
          label="Default Payment Terms"
          description="Establishes the standard timeframe for customer payment that applies when no specific terms have been negotiated."
          options={paymentTermChoices}
        />
      </FormControl>
      <FormControl cols="full">
        <SelectField
          control={control}
          name="billingProfile.transferCriteria"
          label="Transfer Qualification Criteria"
          description="Establishes the primary shipment milestone that triggers eligibility for transfer to the billing system."
          options={transferCriteriaChoices}
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="billingProfile.autoTransfer"
          label="Enable Automatic Transfers"
          description="When enabled, shipments that satisfy all transfer criteria are automatically transferred to the billing system without requiring manual verification."
          position="left"
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="billingProfile.autoMarkReadyToBill"
          label="Automate Ready-to-Bill"
          description="When enabled, shipments that satisfy all transfer criteria are automatically flagged as 'Ready to Bill' without requiring manual verification."
          position="left"
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="billingProfile.autoBill"
          label="Autonomous Invoice Generation"
          description="When enabled, the system will automatically convert qualified shipments into finalized invoices without manual review when predefined criteria are met."
          position="left"
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="billingProfile.autoMarkReadyToBill"
          label="Auto Mark Ready To Bill"
          description="Whether the shipments for this customer should automatically be marked as ready to bill"
          position="left"
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="billingProfile.enforceCustomerBillingReq"
          label="Enforce Customer-Specific Billing Requirements"
          description="When enabled, the system verifies that all customer-mandated documentation, reference numbers, and special handling instructions are fulfilled before allowing shipment transfer to billing."
          position="left"
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="billingProfile.validateCustomerRates"
          label="Validate Contractual Rate Compliance"
          description="When enabled, the system compares all applied charges against authorized customer rate agreements before allowing transfer to billing."
          position="left"
        />
      </FormControl>
    </FormGroup>
  );
}
