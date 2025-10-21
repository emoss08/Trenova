import { Status } from "@/types/common";
import * as z from "zod/v4";
import { geocodeBadeSchema } from "./geocode-schema";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { locationCategorySchema } from "./location-category-schema";
import { usStateSchema } from "./us-state-schema";

export const locationSchema = z.object({
  ...geocodeBadeSchema.shape,
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  status: z.enum(Status),
  name: z.string().min(1, { error: "Name is required" }),
  code: z.string().min(1, { error: "Code is required" }),
  description: optionalStringSchema,
  addressLine1: z.string().min(1, { error: "Address line 1 is required" }),
  addressLine2: optionalStringSchema,
  city: z.string().min(1, { error: "City is required" }),
  postalCode: z.string().min(1, { error: "Postal code is required" }),
  locationCategoryId: z
    .string()
    .min(1, { error: "Location category is required" }),
  stateId: z.string().min(1, { error: "State is required" }),

  // * Relationships
  state: usStateSchema.nullish(),
  locationCategory: locationCategorySchema.nullish(),
});

export type LocationSchema = z.infer<typeof locationSchema>;
