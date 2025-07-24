/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
