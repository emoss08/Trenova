import { Status } from "@/types/common";
import { EquipmentClass } from "@/types/equipment-type";
import { z } from "zod";

export const equipmentTypeSchema = z.object({
  id: z.string().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  status: z.nativeEnum(Status),
  code: z.string().min(1, "Code is required"),
  description: z.string().optional(),
  class: z.nativeEnum(EquipmentClass, {
    message: "Class is required",
  }),
  color: z.string().optional(),
});

export type EquipmentTypeSchema = z.infer<typeof equipmentTypeSchema>;
