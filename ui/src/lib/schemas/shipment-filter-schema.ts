import { ShipmentStatus } from "@/types/shipment";
import { z } from "zod";

export const shipmentFilterSchema = z.object({
  search: z.string().optional(),
  status: z
    .nativeEnum(ShipmentStatus, {
      message: "Status is required",
    })
    .optional(),
});

export type ShipmentFilterSchema = z.infer<typeof shipmentFilterSchema>;
