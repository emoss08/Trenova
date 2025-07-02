import * as z from "zod/v4";
import { ShipmentStatus } from "./shipment-schema";

export const shipmentFilterSchema = z.object({
  search: z.string().optional(),
  status: ShipmentStatus.optional(),
});

export type ShipmentFilterSchema = z.infer<typeof shipmentFilterSchema>;
