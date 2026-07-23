import { z } from "zod";

export const locationDetailsSchema = z.object({
  name: z.string(),
  addressLine1: z.string(),
  city: z.string(),
  state: z.string(),
  stateId: z.string(),
  postalCode: z.string(),
  longitude: z.number(),
  latitude: z.number(),
  placeId: z.string(),
  types: z.array(z.string()),
});
export type LocationDetails = z.infer<typeof locationDetailsSchema>;

export const autocompleteLocationResultSchema = z.object({
  details: z.array(locationDetailsSchema),
  count: z.number(),
});
export type AutocompleteLocationResult = z.infer<typeof autocompleteLocationResultSchema>;

export const autocompleteLocationRequestSchema = z.object({
  input: z.string(),
  sessionToken: z.string().optional(),
});

export type AutocompleteLocationRequest = z.infer<typeof autocompleteLocationRequestSchema>;

