/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import * as z from "zod/v4";
import {
  decimalStringSchema,
  nullableIntegerSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { organizationSchema } from "./organization-schema";

export const consolidationSettingSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  maxPickupDistance: decimalStringSchema,
  maxDeliveryDistance: decimalStringSchema,
  maxRouteDetour: decimalStringSchema,
  maxTimeWindowGap: nullableIntegerSchema,
  minTimeBuffer: nullableIntegerSchema,
  maxShipmentsPerGroup: nullableIntegerSchema,

  organization: organizationSchema.nullish(),
});

export type ConsolidationSettingSchema = z.infer<
  typeof consolidationSettingSchema
>;
