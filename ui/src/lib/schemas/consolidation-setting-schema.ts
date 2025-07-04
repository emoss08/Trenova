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
