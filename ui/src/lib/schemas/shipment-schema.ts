import { RatingMethod, ShipmentStatus } from "@/types/shipment";
import { array, type InferType, mixed, number, object, string } from "yup";
import { moveSchema } from "./move-schema";
import { shipmentCommoditySchema } from "./shipment-commodity-schema";

export const shipmentSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  serviceTypeId: string().required("Service Type is required"),
  shipmentTypeId: string().required("Shipment Type is required"),
  customerId: string().required("Customer is required"),
  tractorTypeId: string()
    .nullable()
    .optional()
    .transform((_, originalValue) => {
      if (
        originalValue === "" ||
        originalValue === null ||
        originalValue === undefined
      ) {
        return undefined;
      }
      return originalValue;
    }),
  trailerTypeId: string()
    .nullable()
    .optional()
    .transform((_, originalValue) => {
      if (
        originalValue === "" ||
        originalValue === null ||
        originalValue === undefined
      ) {
        return undefined;
      }
      return originalValue;
    }),
  status: mixed<ShipmentStatus>()
    .required("Status is required")
    .oneOf(Object.values(ShipmentStatus)),
  proNumber: string().required("Pro Number is required"),
  ratingUnit: number().required("Rating Unit is required"),
  ratingMethod: mixed<RatingMethod>()
    .required("Rating Method is required")
    .oneOf(Object.values(RatingMethod)),
  otherChargeAmount: number().required("Other Charge Amount is required"),
  freightChargeAmount: number().required("Freight Charge Amount is required"),
  totalChargeAmount: number().required("Total Charge Amount is required"),
  pieces: number()
    .transform((_, originalValue) => {
      if (
        originalValue === "" ||
        originalValue === null ||
        originalValue === undefined
      ) {
        return undefined;
      }
      const parsed = parseInt(originalValue, 10);
      return isNaN(parsed) ? undefined : parsed;
    })
    .integer("Pieces must be a whole number")
    .optional(),
  weight: number()
    .transform((_, originalValue) => {
      if (
        originalValue === "" ||
        originalValue === null ||
        originalValue === undefined
      ) {
        return undefined;
      }
      const parsed = parseInt(originalValue, 10);
      return isNaN(parsed) ? undefined : parsed;
    })
    .integer("Weight must be a whole number")
    .optional(),
  temperatureMin: number().required("Temperature Min is required"),
  temperatureMax: number().required("Temperature Max is required"),
  bol: string().required("BOL is required"),
  actualDeliveryDate: number()
    .transform((_, originalValue) => {
      if (
        originalValue === "" ||
        originalValue === null ||
        originalValue === undefined
      ) {
        return undefined;
      }
      const parsed = parseInt(originalValue, 10);
      return isNaN(parsed) ? undefined : parsed;
    })
    .integer("Actual Delivery Date must be a whole number")
    .optional(),
  actualShipDate: number()
    .transform((_, originalValue) => {
      if (
        originalValue === "" ||
        originalValue === null ||
        originalValue === undefined
      ) {
        return undefined;
      }
      const parsed = parseInt(originalValue, 10);
      return isNaN(parsed) ? undefined : parsed;
    })
    .integer("Actual Ship Date must be a whole number")
    .optional(),
  moves: array().of(moveSchema),
  commodities: array().of(shipmentCommoditySchema),
});
export type ShipmentSchema = InferType<typeof shipmentSchema>;
