import { Status } from "@/types/common";
import * as z from "zod/v4";
import {
  decimalStringSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const commoditySchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  hazardousMaterialId: nullableStringSchema,
  status: z.enum(Status),
  name: z.string().min(1, { error: "Name is required" }),
  description: z.string().min(1, { error: "Description is required" }),
  minTemperature: nullableIntegerSchema,
  maxTemperature: nullableIntegerSchema,
  weightPerUnit: decimalStringSchema,
  linearFeetPerUnit: decimalStringSchema,
  freightClass: optionalStringSchema,
  dotClassification: optionalStringSchema,
  stackable: z.boolean(),
  fragile: z.boolean(),
});

export type CommoditySchema = z.infer<typeof commoditySchema>;
