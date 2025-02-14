import { boolean, InferType, object, string } from "yup";

export const shipmentDuplicateSchema = object({
  overrideDates: boolean().required("Override Dates is required"),
  includeCommodities: boolean().required("Include Commodities is required"),
  shipmentID: string().required("Shipment ID is required"),
});

export type ShipmentDuplicateSchema = InferType<typeof shipmentDuplicateSchema>;
