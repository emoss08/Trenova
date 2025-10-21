import * as z from "zod/v4";
import { customerSchema } from "./customer-schema";
import {
  decimalStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { locationSchema } from "./location-schema";

export const distanceOverrideSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  customerId: z.string().nullish(),
  originLocationId: z.string().min(1, { error: "Origin Location is required" }),
  destinationLocationId: z
    .string()
    .min(1, { error: "Destination Location is required" }),

  distance: decimalStringSchema.refine(
    (val) => val !== null && val !== undefined && val > 0,
    { error: "Distance is required" },
  ),

  customer: customerSchema.nullish(),
  originLocation: locationSchema.nullish(),
  destinationLocation: locationSchema.nullish(),
});

export type DistanceOverrideSchema = z.infer<typeof distanceOverrideSchema>;
