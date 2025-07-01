import { ShipmentStatus } from "@/types/shipment";
import * as z from "zod/v4";

export const shipmentFilterSchema = z.object({
  search: z.string().optional(),
  status: z.enum(ShipmentStatus).optional(),
});

export type ShipmentFilterSchema = z.infer<typeof shipmentFilterSchema>;
