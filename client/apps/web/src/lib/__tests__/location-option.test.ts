import { describe, expect, it } from "vitest";
import { locationOptionPlace } from "../location-option";

describe("locationOptionPlace", () => {
  it("reads the name and address a LOCATION option carries", () => {
    expect(
      locationOptionPlace({
        id: "loc_1",
        label: "Columbus Hub",
        description: "COL01",
        meta: {
          code: "COL01",
          addressLine1: "40 Broad St",
          city: "Columbus",
          stateAbbreviation: "OH",
          postalCode: "43215",
        },
      }),
    ).toEqual({
      name: "Columbus Hub",
      addressLine1: "40 Broad St",
      city: "Columbus",
      stateAbbreviation: "OH",
      postalCode: "43215",
    });
  });

  it("reads a missing or mistyped address part as empty", () => {
    expect(
      locationOptionPlace({
        id: "loc_2",
        label: "Yard",
        description: null,
        meta: { city: 12, stateAbbreviation: null },
      }),
    ).toEqual({
      name: "Yard",
      addressLine1: "",
      city: "",
      stateAbbreviation: "",
      postalCode: "",
    });
  });
});
