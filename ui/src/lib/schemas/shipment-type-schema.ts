import { Status } from "@/types/common";
import { z } from "zod";

export const shipmentTypeSchema = z.object({
  id: z.string().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  status: z.nativeEnum(Status, {
    message: "Status is required",
  }),
  code: z.string().min(1, "Code is required"),
  description: z.string().optional(),
  color: z.string().optional(),
});

export type ShipmentTypeSchema = z.infer<typeof shipmentTypeSchema>;
