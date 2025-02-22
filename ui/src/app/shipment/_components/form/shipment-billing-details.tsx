import { AutocompleteField } from "@/components/fields/autocomplete";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { ratingMethodChoices } from "@/lib/choices";
import { CustomerSchema } from "@/lib/schemas/customer-schema";
import { ShipmentCommoditySchema } from "@/lib/schemas/shipment-commodity-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { toNumber } from "@/lib/utils";
import { RatingMethod } from "@/types/shipment";
import { useEffect } from "react";
import { useFormContext } from "react-hook-form";

/** Calculate total linear feet required */
function calculatePerLinearFoot(commodities: ShipmentCommoditySchema[]) {
  return commodities.reduce((total, commodity) => {
    if (commodity.commodity?.linearFeetPerUnit && commodity.pieces) {
      return total + commodity.commodity.linearFeetPerUnit * commodity.pieces;
    }
    return total;
  }, 0);
}

/** Per stop rate calculation */
function calculatePerStopRate(shipment: ShipmentSchema) {
  if (!shipment.moves) throw new Error("No moves found");
  const totalStops = shipment.moves.reduce(
    (total, move) => total + (move.stops?.length ?? 0),
    0,
  );
  return totalStops * shipment.ratingUnit;
}

/** Per linear foot rate calculation */
function calculatePerLinearFootRate(shipment: ShipmentSchema) {
  if (!shipment.commodities) throw new Error("No commodities found");
  const totalLinearFeet = calculatePerLinearFoot(shipment.commodities);
  if (!shipment.ratingUnit) throw new Error("Rate per linear foot is required");
  return totalLinearFeet * shipment.ratingUnit;
}

/** Main function to calculate total charge amount */
function calculateTotalChargeAmount(shipment: ShipmentSchema) {
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
      throw new Error(`Unsupported rating method: ${shipment.ratingMethod}`);
  }

  // Ensure otherChargeAmount is included without affecting the calculation
  return baseCharge + toNumber(shipment.otherChargeAmount);
}

/** Billing details component */
export function ShipmentBillingDetails() {
  const { watch, control, setValue } = useFormContext<ShipmentSchema>();

  useEffect(() => {
    const subscription = watch((value, { name }) => {
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
        ].includes(name ?? "")
      ) {
        try {
          setValue(
            "totalChargeAmount",
            calculateTotalChargeAmount(value as ShipmentSchema),
          );
        } catch (error) {
          console.error(error);
        }
      }
    });

    return () => subscription.unsubscribe();
  }, [watch, setValue]);

  return (
    <div className="flex flex-col gap-2 border-t border-bg-sidebar-border py-4">
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
            renderOption={(option) => option.code}
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
    </div>
  );
}
