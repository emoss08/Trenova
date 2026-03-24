import { z } from "zod";
import {
  optionalStringSchema,
  statusSchema,
  tenantInfoSchema,
} from "./helpers";

export const shipmentTypeSchema = z.object({
  ...tenantInfoSchema.shape,
  status: statusSchema,
  code: z.string().min(1, { error: "Code is required" }),
  description: optionalStringSchema,
  color: optionalStringSchema,
});

export type ShipmentType = z.infer<typeof shipmentTypeSchema>;

export const bulkUpdateShipmentTypeStatusRequestSchema = z.object({
  shipmentTypeIds: z.array(z.string()),
  status: statusSchema,
});

export type BulkUpdateShipmentTypeStatusRequest = z.infer<
  typeof bulkUpdateShipmentTypeStatusRequestSchema
>;

export const bulkUpdateShipmentTypeStatusResponseSchema =
  z.array(shipmentTypeSchema);

export type BulkUpdateShipmentTypeStatusResponse = z.infer<
  typeof bulkUpdateShipmentTypeStatusResponseSchema
>;
