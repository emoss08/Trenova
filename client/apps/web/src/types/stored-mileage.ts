import { z } from "zod";
import { tenantInfoSchema } from "@trenova/shared/types/helpers";

export const storedMileageStopKeySchema = z.object({
  method: z.string(),
  key: z.string(),
  city: z.string().optional(),
  state: z.string().optional(),
  postalCode: z.string().optional(),
  placeId: z.string().optional(),
  coordinates: z.array(z.number()).optional(),
});

export const storedMileageSchema = z.object({
  ...tenantInfoSchema.shape,
  status: z.enum(["Active", "Inactive"]),
  originKey: storedMileageStopKeySchema,
  destinationKey: storedMileageStopKeySchema,
  intermediateKeys: z.array(storedMileageStopKeySchema).optional().default([]),
  routeSignature: z.string(),
  routeHash: z.string(),
  distance: z.number(),
  distanceUnits: z.string(),
  provider: z.string(),
  source: z.string(),
  routingType: z.string(),
  method: z.string(),
  locationGranularity: z.string(),
  dataVersion: z.string(),
  distanceProfileId: z.string(),
  distanceProfileName: z.string().optional(),
  hazmat: z.boolean(),
  hazmatTypes: z.array(z.string()).optional(),
  hazmatSignature: z.string(),
  providerMetadata: z.record(z.string(), z.unknown()).optional(),
  hitCount: z.number(),
  lastUsedAt: z.number().nullish(),
  lastCalculatedAt: z.number(),
});

export type StoredMileage = z.infer<typeof storedMileageSchema>;
