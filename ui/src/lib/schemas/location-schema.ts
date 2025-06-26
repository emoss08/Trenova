import { Status } from "@/types/common";
import * as z from "zod/v4";
import { locationCategorySchema } from "./location-category-schema";
import { usStateSchema } from "./us-state-schema";

export const locationSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  status: z.nativeEnum(Status),
  name: z.string().min(1, "Name is required"),
  code: z.string().min(1, "Code is required"),
  description: z.string().optional(),
  addressLine1: z.string().min(1, "Address line 1 is required"),
  addressLine2: z.string().optional(),
  city: z.string().min(1, "City is required"),
  postalCode: z.string().min(1, "Postal code is required"),
  longitude: z.number().optional().nullable(),
  latitude: z.number().optional().nullable(),
  placeId: z.string().optional(),
  isGeocoded: z.boolean().default(false),
  locationCategoryId: z.string().min(1, "Location category is required"),
  stateId: z.string().min(1, "State is required"),
  state: usStateSchema.optional(),
  locationCategory: locationCategorySchema.optional(),
});

export type LocationSchema = z.infer<typeof locationSchema>;
