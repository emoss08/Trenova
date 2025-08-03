/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { LocationCategoryType } from "@/types/location-category";
import * as z from "zod/v4";
import {
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const locationCategorySchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  name: z
    .string({
      error: "Name is required",
    })
    .min(1, { error: "Name is required" }),
  description: optionalStringSchema,
  type: z.enum(LocationCategoryType, {
    error: "Type is required",
  }),
  facilityType: nullableStringSchema,
  hasSecureParking: z.boolean().default(false),
  requiresAppointment: z.boolean().default(false),
  allowsOvernight: z.boolean().default(false),
  hasRestroom: z.boolean().default(false),
  color: optionalStringSchema,
});

export type LocationCategorySchema = z.infer<typeof locationCategorySchema>;
