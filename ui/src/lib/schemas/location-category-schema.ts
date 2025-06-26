import { FacilityType, LocationCategoryType } from "@/types/location-category";
import * as z from "zod/v4";

export const locationCategorySchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  name: z
    .string({
      error: "Name is required",
    })
    .min(1, "Name is required"),
  description: z.string().optional(),
  type: z.enum(LocationCategoryType, {
    error: "Type is required",
  }),
  facilityType: z.enum(FacilityType, {
    error: "Facility type is required",
  }),
  hasSecureParking: z.boolean().default(false),
  requiresAppointment: z.boolean().default(false),
  allowsOvernight: z.boolean().default(false),
  hasRestroom: z.boolean().default(false),
  color: z.string().optional(),
});

export type LocationCategorySchema = z.infer<typeof locationCategorySchema>;
