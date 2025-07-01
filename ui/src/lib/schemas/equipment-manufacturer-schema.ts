import { Status } from "@/types/common";
import * as z from "zod/v4";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const equipmentManufacturerSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  status: z.enum(Status),
  name: z.string().min(1, { error: "Name is required" }),
  description: optionalStringSchema,
});

export type EquipmentManufacturerSchema = z.infer<
  typeof equipmentManufacturerSchema
>;
