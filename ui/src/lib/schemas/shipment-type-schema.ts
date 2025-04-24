import { Status } from "@/types/common";
import { z } from "zod";

export const shipmentTypeSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  status: z.nativeEnum(Status),
  code: z.string().min(1, "Code is required"),
  description: z.string().optional(),
  color: z.string().optional(),
});

export type ShipmentTypeSchema = z.infer<typeof shipmentTypeSchema>;
