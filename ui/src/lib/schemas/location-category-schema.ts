import { FacilityType, LocationCategoryType } from "@/types/location-category";
import { type InferType, boolean, mixed, object, string } from "yup";

export const locationCategorySchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  name: string().required("Name is required"),
  description: string().optional(),
  type: mixed<LocationCategoryType>()
    .required("Type is required")
    .oneOf(Object.values(LocationCategoryType)),
  facilityType: mixed<FacilityType>().optional().nullable(),
  hasSecureParking: boolean()
    .required("Has Secure Parking is required")
    .default(false),
  requiresAppointment: boolean()
    .required("Requires Appointment is required")
    .default(false),
  allowsOvernight: boolean()
    .required("Allows Overnight is required")
    .default(false),
  hasRestroom: boolean().required("Has Restroom is required").default(false),
  color: string().optional(),
});

export type LocationCategorySchema = InferType<typeof locationCategorySchema>;
