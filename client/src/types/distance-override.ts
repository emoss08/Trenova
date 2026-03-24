import { z } from "zod";
import { customerSchema } from "./customer";
import { nullableStringSchema, tenantInfoSchema } from "./helpers";
import { locationSchema } from "./location";

export const distanceOverrideIntermediateStopSchema = z.object({
  locationId: z.string().min(1, { error: "Intermediate stop location is required" }),
});

export const distanceOverrideSchema = z.object({
  ...tenantInfoSchema.shape,
  originLocationId: z.string().min(1, { error: "Origin location is required" }),
  destinationLocationId: z.string().min(1, { error: "Destination location is required" }),
  customerId: nullableStringSchema,
  distance: z.coerce.number().min(0, { error: "Distance must be greater than or equal to 0" }),
  intermediateStops: z.array(distanceOverrideIntermediateStopSchema).optional().default([]),
  originLocation: locationSchema.nullish(),
  destinationLocation: locationSchema.nullish(),
  customer: customerSchema.nullish(),
});

export type DistanceOverride = z.infer<typeof distanceOverrideSchema>;
