import { z } from "zod";

export const shipmentDuplicateSchema = z.object({
  overrideDates: z.boolean(),
  includeCommodities: z.boolean(),
  includeAdditionalCharges: z.boolean(),
  shipmentID: z.string().min(1, "Shipment ID is required"),
});

export type ShipmentDuplicateSchema = z.infer<typeof shipmentDuplicateSchema>;
