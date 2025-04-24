import { Status } from "@/types/common";
import { z } from "zod";

export const equipmentManufacturerSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  status: z.nativeEnum(Status),
  name: z.string().min(1, "Name is required"),
  description: z.string().optional(),
});

export type EquipmentManufacturerSchema = z.infer<
  typeof equipmentManufacturerSchema
>;
