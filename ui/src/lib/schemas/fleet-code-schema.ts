/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Status } from "@/types/common";
import * as z from "zod/v4";
import {
  decimalStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { userSchema } from "./user-schema";

export const fleetCodeSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  status: z.enum(Status),
  name: z.string().min(1, "Name is required"),
  description: optionalStringSchema,
  revenueGoal: decimalStringSchema,
  deadheadGoal: decimalStringSchema,
  color: optionalStringSchema,
  managerId: z.string().min(1, {
    error: "Manager is required",
  }),
  manager: userSchema.nullish(),
});

export type FleetCodeSchema = z.infer<typeof fleetCodeSchema>;
