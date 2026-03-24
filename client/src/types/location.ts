import { z } from "zod";
import { nullableStringSchema, statusSchema, tenantInfoSchema } from "./helpers";
import { usStateSchema } from "./us-state";

export const locationSchema = z.object({
  ...tenantInfoSchema.shape,
  status: statusSchema,
  code: z
    .string()
    .min(1, { error: "Code is required" })
    .max(10, { error: "Code must be 10 characters or less" }),
  name: z
    .string()
    .min(1, { error: "Name is required" })
    .max(255, { error: "Name must be 255 characters or less" }),
  locationCategoryId: z.string().min(1, { error: "Location category is required" }),
  description: nullableStringSchema,
  addressLine1: z
    .string()
    .min(1, { error: "Address line 1 is required" })
    .max(150, { error: "Address line 1 must be 150 characters or less" }),
  addressLine2: nullableStringSchema,
  city: z
    .string()
    .min(1, { error: "City is required" })
    .max(100, { error: "City must be 100 characters or less" }),
  stateId: z.string().min(1, { error: "State is required" }),
  postalCode: z.string().min(1, { error: "Postal code is required" }),
  isGeocoded: z.boolean().default(false),
  longitude: z.number().nullable().optional(),
  latitude: z.number().nullable().optional(),
  placeId: nullableStringSchema,
  state: usStateSchema.nullish(),
});

export type Location = z.infer<typeof locationSchema>;

export const bulkUpdateLocationStatusRequestSchema = z.object({
  locationIds: z.array(z.string()),
  status: statusSchema,
});

export type BulkUpdateLocationStatusRequest = z.infer<typeof bulkUpdateLocationStatusRequestSchema>;

export const bulkUpdateLocationStatusResponseSchema = z.array(locationSchema);

export type BulkUpdateLocationStatusResponse = z.infer<
  typeof bulkUpdateLocationStatusResponseSchema
>;
