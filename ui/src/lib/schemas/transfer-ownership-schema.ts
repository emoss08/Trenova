import * as z from "zod/v4";
import { nullableStringSchema } from "./helpers";

export const transferOwnershipSchema = z.object({
  ownerId: nullableStringSchema,
  shipmentId: z.string().min(1, { error: "Shipment ID is required" }),
});

export type TransferOwnershipSchema = z.infer<typeof transferOwnershipSchema>;
