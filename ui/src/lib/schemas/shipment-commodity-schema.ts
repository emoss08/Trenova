import { type InferType, number, object, string } from "yup";

export const shipmentCommoditySchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  shipmentId: string().required("Shipment ID is required"),
  commodityId: string().required("Commodity ID is required"),
  weight: number().required("Weight is required"),
  pieces: number().required("Pieces is required"),
});

export type ShipmentCommoditySchema = InferType<typeof shipmentCommoditySchema>;
