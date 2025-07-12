"use no memo";
import { LazyComponent } from "@/components/error-boundary";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import {
  CustomerAutocompleteField,
  FormulaTemplateAutocompleteField,
} from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { ratingMethodChoices } from "@/lib/choices";
import { RatingMethod, ShipmentSchema } from "@/lib/schemas/shipment-schema";
import React, { lazy } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { PreviousRatesDialog } from "./previous-rates-dialog";

const AdditionalChargeDetails = lazy(
  () => import("../additional-charge/additional-charge-details"),
);

/** Billing details component */
export default function ShipmentBillingDetails() {
  return (
    <ShipmentBillingDetailsInner>
      <ShipmentBillingDetailsForm />
      <LazyComponent
        componentLoaderProps={{
          message: "Loading Additional Charges...",
          description: "Please wait while we load the additional charges.",
        }}
      >
        <AdditionalChargeDetails />
      </LazyComponent>
    </ShipmentBillingDetailsInner>
  );

  function ShipmentBillingDetailsInner({
    children,
  }: {
    children: React.ReactNode;
  }) {
    return (
      <>
        <div className="flex flex-col gap-2 border-y border-bg-sidebar-border py-4">
          <div className="flex justify-between items-center">
            <h3 className="text-sm font-medium">Billing Information</h3>
            <PreviousRatesDialog />
          </div>
          {children}
        </div>
      </>
    );
  }

  function ShipmentBillingDetailsForm() {
    const { control } = useFormContext<ShipmentSchema>();

    const ratingMethod = useWatch({ name: "ratingMethod" });
    const ratingMethodIsFormulaTemplate =
      ratingMethod === RatingMethod.enum.FormulaTemplate;

    return (
      <FormGroup cols={2} className="gap-4">
        <FormControl>
          <CustomerAutocompleteField<ShipmentSchema>
            name="customerId"
            control={control}
            label="Customer"
            rules={{ required: true }}
            placeholder="Select Customer"
            description="Choose the customer who requested this shipment."
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="ratingMethod"
            label="Rating Method"
            placeholder="Select Rating Method"
            description="Select how the shipment charges are calculated (e.g., per mile, per stop, flat rate)."
            options={ratingMethodChoices}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="ratingUnit"
            label="Rating Unit"
            type="number"
            placeholder="Enter Rating Unit"
            description="Specify the cost per selected rating method (e.g., per mile or per pallet)."
          />
        </FormControl>
        {ratingMethodIsFormulaTemplate && (
          <FormControl>
            <FormulaTemplateAutocompleteField<ShipmentSchema>
              name="formulaTemplateId"
              control={control}
              rules={{
                required: ratingMethodIsFormulaTemplate,
              }}
              label="Formula Template"
              placeholder="Select Formula Template"
              description="Select the formula template to use for calculating the shipment charges."
            />
          </FormControl>
        )}
        <FormControl>
          <InputField
            tabIndex={-1}
            readOnly
            control={control}
            name="otherChargeAmount"
            type="number"
            label="Other Charges"
            placeholder="Additional Charges"
            description="Sum of all additional charges (tolls, fees, etc.)."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="freightChargeAmount"
            label="Freight Charges"
            placeholder="Enter Freight Charges"
            description="Base charge for transporting the shipment, excluding additional fees."
            type="number"
          />
        </FormControl>
        <FormControl cols={ratingMethodIsFormulaTemplate ? "full" : undefined}>
          <InputField
            tabIndex={-1}
            readOnly
            control={control}
            rules={{ required: true }}
            name="totalChargeAmount"
            label="Total Charge"
            placeholder="Total Charge"
            description="Automatically calculated total, including base and additional charges."
            type="number"
          />
        </FormControl>
      </FormGroup>
    );
  }
}
