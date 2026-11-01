import {
  DistanceControlDocument,
  UpdateDistanceControlDocument,
  type UpdateDistanceControlInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { distanceControlSchema, type DistanceControl } from "@/types/distance-control";

export function toDistanceControlInput(data: DistanceControl): UpdateDistanceControlInput {
  return {
    version: data.version ?? 0,
    storeMileage: data.storeMileage,
    storedDistanceUnits: data.storedDistanceUnits,
    postalCodeFallbackToCity: data.postalCodeFallbackToCity,
    autoCreateStoredMileage: data.autoCreateStoredMileage,
    loadedMoveDistanceProfileId: data.loadedMoveDistanceProfileId,
    emptyMoveDistanceProfileId: data.emptyMoveDistanceProfileId,
    payDistanceProfileId: data.payDistanceProfileId,
    billingDistanceProfileId: data.billingDistanceProfileId,
    fuelDistanceProfileId: data.fuelDistanceProfileId,
    etaOutOfRouteDistanceProfileId: data.etaOutOfRouteDistanceProfileId,
    distanceCalculatorShortestDistanceProfileId: data.distanceCalculatorShortestDistanceProfileId,
    distanceCalculatorPracticalDistanceProfileId: data.distanceCalculatorPracticalDistanceProfileId,
    captureJurisdictionMiles: data.captureJurisdictionMiles,
  };
}

export class DistanceControlService {
  public async get(options?: { signal?: AbortSignal }) {
    const response = await requestGraphQL({
      document: DistanceControlDocument,
      operationName: "DistanceControl",
      signal: options?.signal,
    });

    return safeParse(distanceControlSchema, response.distanceControl, "Distance Control");
  }

  public async update(data: DistanceControl) {
    const response = await requestGraphQL({
      document: UpdateDistanceControlDocument,
      operationName: "UpdateDistanceControl",
      variables: { input: toDistanceControlInput(data) },
    });

    return safeParse(distanceControlSchema, response.updateDistanceControl, "Distance Control");
  }
}
