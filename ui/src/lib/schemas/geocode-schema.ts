import * as z from "zod/v4";
import { decimalStringSchema, optionalStringSchema } from "./helpers";
import { usStateSchema } from "./us-state-schema";

export const geocodeBadeSchema = z.object({
  longitude: decimalStringSchema.nullish(),
  latitude: decimalStringSchema.nullish(),
  placeId: optionalStringSchema,
  isGeocoded: z.boolean().default(false).nullish(),
});

export const locationDetailsSchema = z.object({
  name: z.string(),
  addressLine1: z.string(),
  addressLine2: z.string().optional(),
  city: z.string(),
  stateId: z.string(),
  postalCode: z.string(),
  longitude: decimalStringSchema.nullish(),
  latitude: decimalStringSchema.nullish(),
  placeId: optionalStringSchema,
  types: z.array(z.string()),

  state: usStateSchema.nullish(),
});

export const autocompleteLocationResultSchema = z.object({
  details: z.array(locationDetailsSchema),
  count: z.number(),
});

export const getApiKeyResponseSchema = z.object({
  apiKey: z.string(),
});

export type GeocodeBadeSchema = z.infer<typeof geocodeBadeSchema>;
export type LocationDetailsSchema = z.infer<typeof locationDetailsSchema>;
export type AutocompleteLocationResultSchema = z.infer<
  typeof autocompleteLocationResultSchema
>;
export type GetApiKeyResponseSchema = z.infer<typeof getApiKeyResponseSchema>;
