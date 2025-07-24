/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Status } from "@/types/common";
import { EquipmentClass } from "@/types/equipment-type";
import * as z from "zod/v4";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const equipmentTypeSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  status: z.enum(Status, {
    error: "Status is required",
  }),
  code: z.string().min(1, {
    error: "Code is required",
  }),
  description: optionalStringSchema,
  class: z.enum(EquipmentClass, {
    error: "Class is required",
  }),
  color: optionalStringSchema,
});

export type EquipmentTypeSchema = z.infer<typeof equipmentTypeSchema>;
