import { describe, expect, it } from "vitest";
import { distanceControlSchema } from "@/types/distance-control";

const base = {
  id: "dc_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  organizationId: "org_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  businessUnitId: "bu_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  version: 1,
  createdAt: 1_780_000_000,
  updatedAt: 1_780_000_000,
  storeMileage: true,
  storedDistanceUnits: "Miles" as const,
  postalCodeFallbackToCity: false,
  autoCreateStoredMileage: true,
  loadedMoveDistanceProfileId: "dp_1",
  emptyMoveDistanceProfileId: "dp_2",
  payDistanceProfileId: "dp_3",
  billingDistanceProfileId: "dp_4",
  fuelDistanceProfileId: "dp_5",
  etaOutOfRouteDistanceProfileId: "dp_6",
  distanceCalculatorShortestDistanceProfileId: "dp_7",
  distanceCalculatorPracticalDistanceProfileId: "dp_8",
};

// The jurisdiction-miles toggle is off until an admin turns it on, because
// PC*Miler may bill the state report as an extra transaction per route. A
// server that predates the column must still parse, so the field defaults.

describe("distanceControlSchema captureJurisdictionMiles", () => {
  it("defaults to off when the server omits it", () => {
    const parsed = distanceControlSchema.parse(base);
    expect(parsed.captureJurisdictionMiles).toBe(false);
  });

  it("keeps an explicit setting", () => {
    expect(
      distanceControlSchema.parse({ ...base, captureJurisdictionMiles: true })
        .captureJurisdictionMiles,
    ).toBe(true);
    expect(
      distanceControlSchema.parse({ ...base, captureJurisdictionMiles: false })
        .captureJurisdictionMiles,
    ).toBe(false);
  });

  it("rejects anything that is not a boolean", () => {
    expect(
      distanceControlSchema.safeParse({ ...base, captureJurisdictionMiles: "yes" }).success,
    ).toBe(false);
    expect(
      distanceControlSchema.safeParse({ ...base, captureJurisdictionMiles: null }).success,
    ).toBe(false);
  });
});
