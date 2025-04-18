import { boolean, InferType, object, string } from "yup";

export const shipmentDuplicateSchema = object({
  overrideDates: boolean(),
  includeCommodities: boolean(),
  includeAdditionalCharges: boolean(),
  shipmentID: string().required("Shipment ID is required"),
});

export type ShipmentDuplicateSchema = InferType<typeof shipmentDuplicateSchema>;
