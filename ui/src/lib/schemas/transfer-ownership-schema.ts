import { z } from "zod";

export const transferOwnershipSchema = z.object({
  ownerId: z.string().nullable().optional(),
  shipmentId: z.string().min(1, "Shipment ID is required"),
});

export type TransferOwnershipSchema = z.infer<typeof transferOwnershipSchema>;
