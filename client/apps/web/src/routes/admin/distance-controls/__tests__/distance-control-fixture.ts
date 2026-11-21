import type { DistanceControl } from "@/types/distance-control";

/**
 * A saved distance control as the server sends it (the Go entity's JSON), holding a new
 * organization's defaults (`distancecontrol.NewDefault`): mileage kept, reused and matched by
 * city, miles, no jurisdiction report, the practical profile on every purpose but the
 * shortest-route lookup.
 */
export const savedDistanceControl: DistanceControl = {
  id: "dc_01",
  version: 3,
  createdAt: 1_793_577_600,
  updatedAt: 1_793_577_600,
  organizationId: "org_01",
  businessUnitId: "bu_01",
  storeMileage: true,
  ratingDistanceUnit: "Miles",
  postalCodeFallbackToCity: true,
  autoCreateStoredMileage: true,
  captureJurisdictionMiles: false,
  loadedMoveDistanceProfileId: "dp_practical",
  emptyMoveDistanceProfileId: "dp_practical",
  payDistanceProfileId: "dp_practical",
  billingDistanceProfileId: "dp_practical",
  fuelDistanceProfileId: "dp_practical",
  etaOutOfRouteDistanceProfileId: "dp_practical",
};
