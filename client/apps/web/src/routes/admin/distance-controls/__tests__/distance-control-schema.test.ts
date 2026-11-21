import { describe, expect, it } from "vitest";
import {
  distanceControlFormSchema,
  distanceControlSchema,
  type DistanceControl,
} from "@/types/distance-control";
import { savedDistanceControl } from "./distance-control-fixture";

const record = (changes: Partial<DistanceControl>): DistanceControl => ({
  ...savedDistanceControl,
  ...changes,
});

function refusals(values: unknown) {
  const result = distanceControlFormSchema.safeParse(values);
  return result.success
    ? []
    : result.error.issues.map((issue) => [issue.path.join("."), issue.message]);
}

// From the server: DistanceControl.Validate in places/domain/distancecontrol.
describe("distance control contract", () => {
  it("reads a record whose profile assignments are empty, as one made before any profile", () => {
    const empty = record({ loadedMoveDistanceProfileId: "", fuelDistanceProfileId: "" });

    expect(distanceControlSchema.parse(empty)).toEqual(empty);
  });

  it("reads both units the server accepts and refuses any other", () => {
    expect(distanceControlSchema.parse(record({ ratingDistanceUnit: "Kilometers" }))).toEqual(
      record({ ratingDistanceUnit: "Kilometers" }),
    );
    expect(
      distanceControlSchema.safeParse({ ...savedDistanceControl, ratingDistanceUnit: "Feet" })
        .success,
    ).toBe(false);
  });

  it("requires the jurisdiction switch the server always sends", () => {
    const { captureJurisdictionMiles: _omitted, ...withoutSwitch } = savedDistanceControl;

    expect(distanceControlSchema.safeParse(withoutSwitch).success).toBe(false);
  });

  it("saves the defaults as they are", () => {
    expect(refusals(savedDistanceControl)).toEqual([]);
  });

  it("refuses every empty profile assignment in the server's words", () => {
    expect(
      refusals(
        record({
          loadedMoveDistanceProfileId: "",
          emptyMoveDistanceProfileId: "",
          payDistanceProfileId: "",
          billingDistanceProfileId: "",
          fuelDistanceProfileId: "",
          etaOutOfRouteDistanceProfileId: " ",
        }),
      ),
    ).toEqual([
      ["loadedMoveDistanceProfileId", "Loaded move distance profile is required"],
      ["emptyMoveDistanceProfileId", "Empty move distance profile is required"],
      ["payDistanceProfileId", "Pay distance profile is required"],
      ["billingDistanceProfileId", "Billing distance profile is required"],
      ["fuelDistanceProfileId", "Fuel distance profile is required"],
      ["etaOutOfRouteDistanceProfileId", "ETA/out-of-route distance profile is required"],
    ]);
  });

  it("does not care which switches are on", () => {
    expect(
      refusals(
        record({
          storeMileage: false,
          autoCreateStoredMileage: false,
          postalCodeFallbackToCity: true,
          captureJurisdictionMiles: true,
        }),
      ),
    ).toEqual([]);
  });
});
