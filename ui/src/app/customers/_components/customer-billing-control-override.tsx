import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { GLAccountAutocompleteField } from "@/components/ui/autocomplete-fields";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup } from "@/components/ui/form";
import { paymentTermChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { useQuery } from "@tanstack/react-query";
import { useFormContext, useWatch } from "react-hook-form";

export function BillingControlOverrides() {
  const { data: billingControl, isLoading: billingControlLoading } = useQuery({
    ...queries.organization.getBillingControl(),
  });

  const { setValue, getValues } = useFormContext<CustomerSchema>();

  const hasOverrides = useWatch({
    name: "billingProfile.hasOverrides",
  });

  const toggleBillingControlOverrides = (show: boolean) => {
    setValue("billingProfile.hasOverrides", show, {
      shouldDirty: true,
      shouldTouch: true,
      shouldValidate: true,
    });

    if (show && billingControl && !getValues("billingProfile.paymentTerm")) {
      setValue("billingProfile.paymentTerm", billingControl.paymentTerm, {
        shouldDirty: true,
      });
      setValue("billingProfile.autoTransfer", billingControl.autoTransfer, {
        shouldDirty: true,
      });
      setValue(
        "billingProfile.autoMarkReadyToBill",
        billingControl.autoMarkReadyToBill,
        { shouldDirty: true },
      );
      setValue("billingProfile.autoBill", billingControl.autoBill, {
        shouldDirty: true,
      });
      setValue(
        "billingProfile.enforceCustomerBillingReq",
        billingControl.enforceCustomerBillingReq,
        { shouldDirty: true },
      );
      setValue(
        "billingProfile.validateCustomerRates",
        billingControl.validateCustomerRates,
        { shouldDirty: true },
      );
    }
  };

  return (
    <div className="flex flex-col gap-4 border-t pt-4">
      <div className="flex items-center justify-between">
        <h3
          id="billing-control-overrides"
          className="text-sm leading-none font-semibold tracking-tight"
        >
          Customer-Specific Billing Control Overrides
        </h3>
        {hasOverrides && (
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
      {hasOverrides ? (
        <BillingControlOverridesForm />
      ) : (
        <Button
          onClick={() => toggleBillingControlOverrides(true)}
          className="w-full"
          variant="outline"
          type="button"
          disabled={billingControlLoading}
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
          name="billingProfile.enforceCustomerBillingReq"
          label="Enforce Customer-Specific Billing Requirements"
          description="When enabled, the system verifies that all customer-mandated documentation, reference numbers, and special handling instructions are fulfilled before allowing shipment transfer to billing."
          position="left"
        />
      </FormControl>
      <FormControl cols="full">
        <SwitchField
          control={control}
          name="billingProfile.validateCustomerRates"
          label="Validate Contractual Rate Compliance"
          description="When enabled, the system compares all applied charges against authorized customer rate agreements before allowing transfer to billing."
          position="left"
        />
      </FormControl>
      <FormControl>
        <GLAccountAutocompleteField
          control={control}
          name="billingProfile.revenueAccountId"
          label="Revenue Account"
          description="The GL account to use for revenue transactions."
          placeholder="Select Revenue Account"
        />
      </FormControl>
      <FormControl>
        <GLAccountAutocompleteField
          control={control}
          name="billingProfile.arAccountId"
          label="AR Account"
          description="The GL account to use for AR transactions."
          placeholder="Select AR Account"
        />
      </FormControl>
    </FormGroup>
  );
}
