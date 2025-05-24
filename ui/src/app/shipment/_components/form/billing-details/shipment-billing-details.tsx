"use no memo";
import { LazyComponent } from "@/components/error-boundary";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { CustomerAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { ratingMethodChoices } from "@/lib/choices";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { toNumber } from "@/lib/utils";
import { RatingMethod } from "@/types/shipment";
import { useDebouncedEffect } from "@wojtekmaj/react-hooks";
import React, { lazy } from "react";
import { useFormContext } from "react-hook-form";

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
}

function ShipmentBillingDetailsInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-2 border-y border-bg-sidebar-border py-4">
      <h3 className="text-sm font-medium">Billing Information</h3>
      {children}
    </div>
  );
}

function ShipmentBillingDetailsForm() {
  const { control, watch, setValue } = useFormContext<ShipmentSchema>();

  const ratingMethod = watch("ratingMethod");
  const freightChargeAmount = watch("freightChargeAmount");
  const otherChargeAmount = watch("otherChargeAmount");

  // * Debounce the total charge amount calculation to prevent excessive re-renders
  useDebouncedEffect(
    () => {
      if (ratingMethod && ratingMethod === RatingMethod.FlatRate) {
        setValue(
          "totalChargeAmount",
          toNumber(freightChargeAmount) + toNumber(otherChargeAmount),
        );
      }
    },
    [ratingMethod, freightChargeAmount, otherChargeAmount, setValue],
    500, // * 500ms delay
  );

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
          placeholder="Enter Rating Unit"
          description="Specify the cost per selected rating method (e.g., per mile or per pallet)."
        />
      </FormControl>
      <FormControl>
        <InputField
          readOnly
          control={control}
          name="otherChargeAmount"
          label="Other Charges"
          placeholder="Auto-calculated Additional Charges"
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
        />
      </FormControl>
      <FormControl>
        <InputField
          readOnly
          control={control}
          rules={{ required: true }}
          name="totalChargeAmount"
          label="Total Charge"
          placeholder="Auto-calculated Total"
          description="Automatically calculated total, including base and additional charges."
        />
      </FormControl>
    </FormGroup>
  );
}
