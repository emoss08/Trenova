import {
  CreateDistanceProfileDocument,
  DeleteDistanceProfileDocument,
  SetDefaultDistanceProfileDocument,
  UpdateDistanceProfileDocument,
  type DistanceProfileInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { DistanceProfile } from "@/types/distance-profile";

export function toDistanceProfileInput(data: DistanceProfile): DistanceProfileInput {
  return {
    name: data.name,
    description: data.description || null,
    status: data.status,
    isDefault: data.isDefault,
    provider: data.provider,
    dataVersion: data.dataVersion,
    region: data.region,
    routingType: data.routingType,
    distanceUnits: data.distanceUnits,
    locationGranularity: data.locationGranularity,
    profileName: data.profileName || null,
    highwayOnly: data.highwayOnly,
    tollRoads: data.tollRoads,
    bordersOpen: data.bordersOpen,
    includeTollData: data.includeTollData,
    version: data.version,
  };
}

export class DistanceProfileService {
  public async create(data: DistanceProfile) {
    const response = await requestGraphQL({
      document: CreateDistanceProfileDocument,
      operationName: "CreateDistanceProfile",
      variables: { input: toDistanceProfileInput(data) },
    });

    return response.createDistanceProfile;
  }

  public async update(id: string, data: DistanceProfile) {
    const response = await requestGraphQL({
      document: UpdateDistanceProfileDocument,
      operationName: "UpdateDistanceProfile",
      variables: { id, input: toDistanceProfileInput(data) },
    });

    return response.updateDistanceProfile;
  }

  public async delete(id: string) {
    await requestGraphQL({
      document: DeleteDistanceProfileDocument,
      operationName: "DeleteDistanceProfile",
      variables: { id },
    });
  }

  public async setDefault(id: string) {
    const response = await requestGraphQL({
      document: SetDefaultDistanceProfileDocument,
      operationName: "SetDefaultDistanceProfile",
      variables: { id },
    });

    return response.setDefaultDistanceProfile;
  }
}
