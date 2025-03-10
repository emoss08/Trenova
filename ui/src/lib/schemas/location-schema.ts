import { Status } from "@/types/common";
import { type InferType, boolean, mixed, number, object, string } from "yup";
// import { locationCategorySchema } from "./location-category-schema";
import { usStateSchema } from "./us-state-schema";

export const locationSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  status: mixed<Status>()
    .required("Status is required")
    .oneOf(Object.values(Status)),
  name: string().required("Name is required"),
  code: string().required("Code is required"),
  description: string().optional(),
  addressLine1: string().required("Address line 1 is required"),
  addressLine2: string().optional(),
  city: string().required("City is required"),
  postalCode: string().required("Postal code is required"),
  longitude: number().optional().nullable(),
  latitude: number().optional().nullable(),
  placeId: string().optional(),
  isGeocoded: boolean().required("Is geocoded is required"),
  locationCategoryId: string().required("Location category is required"),
  stateId: string().required("State is required"),
  state: usStateSchema.notRequired().optional().nullable(),
  // locationCategory: locationCategorySchema.nullable().optional(),
});

export type LocationSchema = InferType<typeof locationSchema>;
