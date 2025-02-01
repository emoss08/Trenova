import { ShipmentStatus } from "@/types/shipment";
import { type InferType, mixed, object, string } from "yup";

export const shipmentFilterSchema = object({
  search: string().optional(),
  status: mixed<ShipmentStatus>()
    .optional()
    .oneOf(Object.values(ShipmentStatus)),
});

export type ShipmentFilterSchema = InferType<typeof shipmentFilterSchema>;
