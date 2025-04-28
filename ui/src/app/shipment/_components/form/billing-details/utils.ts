import { AdditionalChargeSchema } from "@/lib/schemas/additional-charge-schema";
import { ShipmentCommoditySchema } from "@/lib/schemas/shipment-commodity-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { toNumber } from "@/lib/utils";
import { AccessorialChargeMethod } from "@/types/billing";
import { RatingMethod } from "@/types/shipment";

/**
 * Calculate the amount for a single additional charge
 * @param charge The additional charge to calculate
 * @param baseLinehaul The base linehaul amount used for percentage calculations
 * @returns The calculated amount for the additional charge
 */
export function calculateAdditionalChargeAmount(
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
export function calculatePerLinearFoot(commodities: ShipmentCommoditySchema[]) {
  if (!commodities || !commodities.length) return 0;

  return commodities.reduce((total, commodity) => {
    if (commodity.commodity?.linearFeetPerUnit && commodity.pieces) {
      return total + commodity.commodity.linearFeetPerUnit * commodity.pieces;
    }
    return total;
  }, 0);
}

/** Per stop rate calculation */
export function calculatePerStopRate(shipment: ShipmentSchema) {
  if (!shipment.moves || !shipment.moves.length) return 0;

  const totalStops = shipment.moves.reduce(
    (total, move) => total + (move.stops?.length ?? 0),
    0,
  );
  return totalStops * toNumber(shipment.ratingUnit);
}

/** Per linear foot rate calculation */
export function calculatePerLinearFootRate(shipment: ShipmentSchema) {
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
export function calculateBaseCharge(shipment: ShipmentSchema): number {
  // * If the shipment is new, the rating method will be undefined
  if (!shipment.ratingMethod) return 0;

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
export function calculateTotalAdditionalCharges(
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
export function calculateTotalChargeAmount(shipment: ShipmentSchema): number {
  if (!shipment) return 0;

  const baseCharge = calculateBaseCharge(shipment);
  const additionalCharges = calculateTotalAdditionalCharges(
    shipment,
    baseCharge,
  );

  // Return total charges
  return baseCharge + additionalCharges;
}
