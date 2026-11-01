import {
  CreateLocationCategoryDocument,
  UpdateLocationCategoryDocument,
  type LocationCategoryInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { LocationCategory } from "@/types/location-category";

export function toLocationCategoryInput(values: LocationCategory): LocationCategoryInput {
  return {
    name: values.name,
    description: values.description || null,
    type: values.type,
    facilityType: values.facilityType ?? null,
    color: values.color ?? null,
    hasSecureParking: values.hasSecureParking,
    requiresAppointment: values.requiresAppointment,
    allowsOvernight: values.allowsOvernight,
    hasRestroom: values.hasRestroom,
    version: values.version,
  };
}

export async function createLocationCategory(values: LocationCategory) {
  const data = await requestGraphQL({
    document: CreateLocationCategoryDocument,
    operationName: "CreateLocationCategory",
    variables: { input: toLocationCategoryInput(values) },
  });

  return data.createLocationCategory;
}

export async function updateLocationCategory(id: string, values: LocationCategory) {
  const data = await requestGraphQL({
    document: UpdateLocationCategoryDocument,
    operationName: "UpdateLocationCategory",
    variables: { id, input: toLocationCategoryInput(values) },
  });

  return data.updateLocationCategory;
}
