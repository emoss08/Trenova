import { LazyComponent } from "@/components/error-boundary";
import { AutocompleteField } from "@/components/fields/autocomplete";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { ratingMethodChoices } from "@/lib/choices";
import { AdditionalChargeSchema } from "@/lib/schemas/additional-charge-schema";
import { CustomerSchema } from "@/lib/schemas/customer-schema";
import { ShipmentCommoditySchema } from "@/lib/schemas/shipment-commodity-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { toNumber } from "@/lib/utils";
import { AccessorialChargeMethod } from "@/types/billing";
import { RatingMethod } from "@/types/shipment";
import { lazy, useEffect } from "react";
import { useFormContext } from "react-hook-form";

const AdditionalChargeDetails = lazy(
  () => import("./additional-charge/additional-charge-details"),
);

/**
 * Calculate the amount for a single additional charge
 * @param charge The additional charge to calculate
 * @param baseLinehaul The base linehaul amount used for percentage calculations
 * @returns The calculated amount for the additional charge
 */
function calculateAdditionalChargeAmount(
  charge: AdditionalChargeSchema,
  baseLinehaul: number,
) {
  // Ensure we have valid data to work with
  if (!charge) return 0;

  // Default to 1 for unit if not specified for Flat method
  const unit =
    charge.method === AccessorialChargeMethod.Flat
      ? charge.unit || 1
      : charge.unit || 0;

  switch (charge.method) {
    case AccessorialChargeMethod.Flat:
      return toNumber(charge.amount) * toNumber(unit);
    case AccessorialChargeMethod.Distance:
      return toNumber(charge.amount) * toNumber(unit);
    case AccessorialChargeMethod.Percentage:
      return (toNumber(charge.amount) / 100) * baseLinehaul;
    default:
      console.warn(`Unsupported accessorial charge method: ${charge.method}`);
      return 0;
  }
}

/** Calculate total linear feet required */
function calculatePerLinearFoot(commodities: ShipmentCommoditySchema[]) {
  if (!commodities || !commodities.length) return 0;

  return commodities.reduce((total, commodity) => {
    if (commodity.commodity?.linearFeetPerUnit && commodity.pieces) {
      return total + commodity.commodity.linearFeetPerUnit * commodity.pieces;
    }
    return total;
  }, 0);
}

/** Per stop rate calculation */
function calculatePerStopRate(shipment: ShipmentSchema) {
  if (!shipment.moves || !shipment.moves.length) return 0;

  const totalStops = shipment.moves.reduce(
    (total, move) => total + (move.stops?.length ?? 0),
    0,
  );
  return totalStops * toNumber(shipment.ratingUnit);
}

/** Per linear foot rate calculation */
function calculatePerLinearFootRate(shipment: ShipmentSchema) {
  if (!shipment.commodities || !shipment.commodities.length) return 0;

  const totalLinearFeet = calculatePerLinearFoot(shipment.commodities);
  // Return 0 if ratingUnit is missing rather than throwing an error
  if (!shipment.ratingUnit) return 0;

  return totalLinearFeet * toNumber(shipment.ratingUnit);
}

/**
 * Calculate the base charge for a shipment based on its rating method
 * @param shipment The shipment to calculate the base charge for
 * @returns The calculated base charge
 */
function calculateBaseCharge(shipment: ShipmentSchema): number {
  if (!shipment) return 0;

  let baseCharge = 0;

  switch (shipment.ratingMethod) {
    case RatingMethod.FlatRate:
      baseCharge = toNumber(shipment.freightChargeAmount);
      break;
    case RatingMethod.PerMile:
      baseCharge =
        toNumber(shipment.ratingUnit) * toNumber(shipment.freightChargeAmount);
      break;
    case RatingMethod.PerStop:
      baseCharge = calculatePerStopRate(shipment);
      break;
    case RatingMethod.PerPound:
      baseCharge = toNumber(shipment.weight) * toNumber(shipment.ratingUnit);
      break;
    case RatingMethod.PerPallet:
      baseCharge = toNumber(shipment.pieces) * toNumber(shipment.ratingUnit);
      break;
    case RatingMethod.PerLinearFoot:
      baseCharge = calculatePerLinearFootRate(shipment);
      break;
    case RatingMethod.Other:
      baseCharge =
        toNumber(shipment.ratingUnit) * toNumber(shipment.freightChargeAmount);
      break;
    default:
      // Return 0 instead of throwing an error to make the function more robust
      console.warn(`Unsupported rating method: ${shipment.ratingMethod}`);
      return 0;
  }

  return baseCharge;
}

/**
 * Calculate the total additional charges for a shipment
 * @param shipment The shipment to calculate additional charges for
 * @param baseCharge The base charge used for percentage calculations
 * @returns The sum of all additional charges
 */
function calculateTotalAdditionalCharges(
  shipment: ShipmentSchema,
  baseCharge: number,
): number {
  if (
    !shipment ||
    !shipment.additionalCharges ||
    !shipment.additionalCharges.length
  ) {
    return 0;
  }

  return shipment.additionalCharges.reduce((total, charge) => {
    try {
      return (
        total +
        calculateAdditionalChargeAmount(
          charge as AdditionalChargeSchema,
          baseCharge,
        )
      );
    } catch (error) {
      console.error("Error calculating additional charge:", error);
      return total; // Skip problematic charges rather than breaking the whole calculation
    }
  }, 0);
}

/**
 * Calculate the total charge amount for a shipment
 * @param shipment The shipment to calculate the total charge for
 * @returns The total charge amount including base, other, and additional charges
 */
function calculateTotalChargeAmount(shipment: ShipmentSchema): number {
  if (!shipment) return 0;

  const baseCharge = calculateBaseCharge(shipment);
  const additionalCharges = calculateTotalAdditionalCharges(
    shipment,
    baseCharge,
  );

  // Return total charges
  return baseCharge + additionalCharges;
}

/** Billing details component */
export default function ShipmentBillingDetails() {
  const { watch, control, setValue, getValues } =
    useFormContext<ShipmentSchema>();

  // Handle changes to the form that affect billing calculations
  useEffect(() => {
    const subscription = watch((value, { name }) => {
      // Check if additionalCharges have been modified
      if (
        name?.startsWith("additionalCharges") ||
        name === "additionalCharges"
      ) {
        try {
          const shipment = value as ShipmentSchema;
          const baseCharge = calculateBaseCharge(shipment);

          // Calculate the total of all additional charges
          const additionalChargesTotal = calculateTotalAdditionalCharges(
            shipment,
            baseCharge,
          );

          // Update the otherChargeAmount field with the sum of additional charges
          setValue("otherChargeAmount", additionalChargesTotal, {
            shouldValidate: true,
            shouldDirty: false,
          });
        } catch (error) {
          console.error("Error updating otherChargeAmount:", error);
        }
      }

      // Update the total charge amount if any relevant fields change
      if (
        [
          "ratingMethod",
          "ratingUnit",
          "commodities",
          "weight",
          "pieces",
          "moves",
          "otherChargeAmount",
          "freightChargeAmount",
          "additionalCharges",
        ].includes(name ?? "") ||
        name?.startsWith("additionalCharges")
      ) {
        try {
          const shipment = value as ShipmentSchema;
          setValue("totalChargeAmount", calculateTotalChargeAmount(shipment), {
            shouldValidate: true,
          });
        } catch (error) {
          console.error("Error updating totalChargeAmount:", error);
        }
      }
    });

    return () => subscription.unsubscribe();
  }, [watch, setValue]);

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
    <div className="flex flex-col gap-2 border-y border-bg-sidebar-border py-4">
      <h3 className="text-sm font-medium">Billing Information</h3>
      <FormGroup cols={2} className="gap-4">
        <FormControl>
          <AutocompleteField<CustomerSchema, ShipmentSchema>
            name="customerId"
            control={control}
            link="/customers/"
            label="Customer"
            rules={{ required: true }}
            placeholder="Select Customer"
            description="Choose the customer who requested this shipment."
            getOptionValue={(option) => option.id ?? ""}
            getDisplayValue={(option) => option.code}
            renderOption={(option) => (
              <div className="flex flex-col gap-0.5 items-start size-full">
                <p className="text-sm font-medium">{option.code}</p>
                {option.name && (
                  <p className="text-xs text-muted-foreground truncate w-full">
                    {option.name}
                  </p>
                )}
              </div>
            )}
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
            rules={{ required: true }}
            name="otherChargeAmount"
            label="Other Charges"
            placeholder="Enter Additional Charges"
            description="Include any extra costs not covered by the primary rating method (e.g., tolls, fees)."
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
      <LazyComponent
        componentLoaderProps={{
          message: "Loading Additional Charges...",
          description: "Please wait while we load the additional charges.",
        }}
      >
        <AdditionalChargeDetails />
      </LazyComponent>
    </div>
  );
}
