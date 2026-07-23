import { z } from "zod";
import {
  nullableStringSchema,
  optionalStringSchema,
  tenantInfoSchema,
} from "./helpers";

export const locationCategoryTypeSchema = z.enum([
  "Terminal",
  "Warehouse",
  "DistributionCenter",
  "TruckStop",
  "RestArea",
  "CustomerLocation",
  "Port",
  "RailYard",
  "MaintenanceFacility",
]);

export type LocationCategoryType = z.infer<typeof locationCategoryTypeSchema>;

export const facilityTypeSchema = z.enum([
  "CrossDock",
  "StorageWarehouse",
  "ColdStorage",
  "HazmatFacility",
  "IntermodalFacility",
]);

export type FacilityType = z.infer<typeof facilityTypeSchema>;

export const locationCategorySchema = z.object({
  ...tenantInfoSchema.shape,
  name: z.string().min(1, { message: "Name is required" }).max(100),
  description: optionalStringSchema,
  type: locationCategoryTypeSchema,
  facilityType: facilityTypeSchema.optional().nullable(),
  color: nullableStringSchema,
  hasSecureParking: z.boolean(),
  requiresAppointment: z.boolean(),
  allowsOvernight: z.boolean(),
  hasRestroom: z.boolean(),
});

export type LocationCategory = z.infer<typeof locationCategorySchema>;
