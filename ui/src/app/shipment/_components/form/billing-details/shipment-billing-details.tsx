import { LazyComponent } from "@/components/error-boundary";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { CustomerAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { ratingMethodChoices } from "@/lib/choices";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import React, { lazy, useEffect } from "react";
import { useFormContext } from "react-hook-form";
import {
  calculateBaseCharge,
  calculateTotalAdditionalCharges,
  calculateTotalChargeAmount,
} from "./utils";

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
  const { watch, control, setValue, getValues } =
    useFormContext<ShipmentSchema>();

  // // Handle changes to the form that affect billing calculations
  // useEffect(() => {
  //   const subscription = watch((value, { name }) => {
  //     // Check if additionalCharges have been modified
  //     if (
  //       name?.startsWith("additionalCharges") ||
  //       name === "additionalCharges"
  //     ) {
  //       try {
  //         const shipment = value as ShipmentSchema;
  //         const baseCharge = calculateBaseCharge(shipment);

  //         // Calculate the total of all additional charges
  //         const additionalChargesTotal = calculateTotalAdditionalCharges(
  //           shipment,
  //           baseCharge,
  //         );

  //         // Update the otherChargeAmount field with the sum of additional charges
  //         setValue("otherChargeAmount", additionalChargesTotal, {
  //           shouldValidate: true,
  //           shouldDirty: false,
  //         });
  //       } catch (error) {
  //         console.error("Error updating otherChargeAmount:", error);
  //       }
  //     }

  //     // Update the total charge amount if any relevant fields change
  //     if (
  //       [
  //         "ratingMethod",
  //         "ratingUnit",
  //         "commodities",
  //         "weight",
  //         "pieces",
  //         "moves",
  //         "otherChargeAmount",
  //         "freightChargeAmount",
  //         "additionalCharges",
  //       ].includes(name ?? "") ||
  //       name?.startsWith("additionalCharges")
  //     ) {
  //       try {
  //         const shipment = value as ShipmentSchema;
  //         setValue("totalChargeAmount", calculateTotalChargeAmount(shipment), {
  //           shouldValidate: true,
  //         });
  //       } catch (error) {
  //         console.error("Error updating totalChargeAmount:", error);
  //       }
  //     }
  //   });

  //   return () => subscription.unsubscribe();
  // }, [watch, setValue]);

  // Initialize calculations when the component mounts
  useEffect(() => {
    try {
      const shipment = getValues() as ShipmentSchema;
      const baseCharge = calculateBaseCharge(shipment);

      // Calculate and set the additional charges total
      const additionalChargesTotal = calculateTotalAdditionalCharges(
        shipment,
        baseCharge,
      );

      // Only set if there are actually additional charges to prevent overriding user input
      if (shipment.additionalCharges && shipment.additionalCharges.length > 0) {
        setValue("otherChargeAmount", additionalChargesTotal, {
          shouldValidate: true,
          shouldDirty: false,
        });
      }

      // Calculate and set the total charge amount
      setValue("totalChargeAmount", calculateTotalChargeAmount(shipment), {
        shouldValidate: true,
        shouldDirty: false,
      });
    } catch (error) {
      console.error("Error initializing billing calculations:", error);
    }
  }, [getValues, setValue]);

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
