import { FacilityType, LocationCategoryType } from "@/types/location-category";
import { z } from "zod";

export const locationCategorySchema = z.object({
  id: z.string().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  name: z.string().min(1, "Name is required"),
  description: z.string().optional(),
  type: z.nativeEnum(LocationCategoryType),
  facilityType: z.nativeEnum(FacilityType),
  hasSecureParking: z.boolean().default(false),
  requiresAppointment: z.boolean().default(false),
  allowsOvernight: z.boolean().default(false),
  hasRestroom: z.boolean().default(false),
  color: z.string().optional(),
});

export type LocationCategorySchema = z.infer<typeof locationCategorySchema>;
