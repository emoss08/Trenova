import {
  CreateDistanceOverrideDocument,
  DeleteDistanceOverrideDocument,
  UpdateDistanceOverrideDocument,
  type DistanceOverrideInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { DistanceOverride } from "@/types/distance-override";

export function toDistanceOverrideInput(data: DistanceOverride): DistanceOverrideInput {
  return {
    originLocationId: data.originLocationId,
    destinationLocationId: data.destinationLocationId,
    customerId: data.customerId ?? null,
    distance: data.distance,
    intermediateStops: data.intermediateStops.map((stop) => ({ locationId: stop.locationId })),
    version: data.version,
  };
}

export class DistanceOverrideService {
  public async create(data: DistanceOverride) {
    const response = await requestGraphQL({
      document: CreateDistanceOverrideDocument,
      operationName: "CreateDistanceOverride",
      variables: { input: toDistanceOverrideInput(data) },
    });

    return response.createDistanceOverride;
  }

  public async update(id: string, data: DistanceOverride) {
    const response = await requestGraphQL({
      document: UpdateDistanceOverrideDocument,
      operationName: "UpdateDistanceOverride",
      variables: { id, input: toDistanceOverrideInput(data) },
    });

    return response.updateDistanceOverride;
  }

  public async delete(id: string) {
    await requestGraphQL({
      document: DeleteDistanceOverrideDocument,
      operationName: "DeleteDistanceOverride",
      variables: { id },
    });
  }
}
