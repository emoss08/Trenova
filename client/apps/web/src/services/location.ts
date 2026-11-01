import {
  BulkUpdateLocationStatusDocument,
  CreateLocationDocument,
  PatchLocationDocument,
  UpdateLocationDocument,
  type LocationInput,
  type LocationPatchInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { BulkUpdateLocationStatusRequest, Location } from "@trenova/shared/types/location";

export function toLocationInput(data: Location): LocationInput {
  return {
    status: data.status,
    code: data.code || null,
    name: data.name,
    description: data.description || null,
    locationCategoryId: data.locationCategoryId,
    stateId: data.stateId,
    addressLine1: data.addressLine1,
    addressLine2: data.addressLine2 || null,
    city: data.city,
    postalCode: data.postalCode,
    timezone: data.timezone || null,
    placeId: data.placeId || null,
    isGeocoded: data.isGeocoded,
    longitude: data.longitude ?? null,
    latitude: data.latitude ?? null,
    geofenceType: data.geofenceType,
    geofenceRadiusMeters: data.geofenceRadiusMeters ?? null,
    geofenceVertices: data.geofenceVertices.map((vertex) => ({
      latitude: vertex.latitude,
      longitude: vertex.longitude,
    })),
    version: data.version,
  };
}

export class LocationService {
  public async create(data: Location) {
    const response = await requestGraphQL({
      document: CreateLocationDocument,
      operationName: "CreateLocation",
      variables: { input: toLocationInput(data) },
    });

    return response.createLocation;
  }

  public async update(id: string, data: Location) {
    const response = await requestGraphQL({
      document: UpdateLocationDocument,
      operationName: "UpdateLocation",
      variables: { id, input: toLocationInput(data) },
    });

    return response.updateLocation;
  }

  public async bulkUpdateStatus(request: BulkUpdateLocationStatusRequest) {
    const response = await requestGraphQL({
      document: BulkUpdateLocationStatusDocument,
      operationName: "BulkUpdateLocationStatus",
      variables: { input: request },
    });

    return response.bulkUpdateLocationStatus;
  }

  public async patch(id: string, input: LocationPatchInput) {
    const response = await requestGraphQL({
      document: PatchLocationDocument,
      operationName: "PatchLocation",
      variables: { id, input },
    });

    return response.patchLocation;
  }
}
