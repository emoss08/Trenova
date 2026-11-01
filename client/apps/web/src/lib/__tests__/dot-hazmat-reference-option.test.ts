import { describe, expect, it } from "vitest";
import { dotHazmatReferenceFromOption } from "../dot-hazmat-reference-option";

describe("dotHazmatReferenceFromOption", () => {
  it("reads every field the hazmat form copies from a DOT_HAZMAT_REFERENCE option", () => {
    expect(
      dotHazmatReferenceFromOption({
        id: "dhr_1",
        label: "Gasoline",
        description: "UN1203",
        meta: {
          unNumber: "1203",
          properShippingName: "Gasoline",
          hazardClass: "3",
          subsidiaryHazard: "",
          packingGroup: "II",
          specialProvisions: "144, 177",
          ergGuide: "128",
          symbols: "RQ",
        },
      }),
    ).toEqual({
      id: "dhr_1",
      unNumber: "1203",
      properShippingName: "Gasoline",
      hazardClass: "3",
      subsidiaryHazard: "",
      packingGroup: "II",
      specialProvisions: "144, 177",
      ergGuide: "128",
      symbols: "RQ",
    });
  });

  it("falls back to the option's label when the shipping name is absent", () => {
    const reference = dotHazmatReferenceFromOption({
      id: "dhr_2",
      label: "Diesel fuel",
      description: null,
      meta: null,
    });

    expect(reference.properShippingName).toBe("Diesel fuel");
    expect(reference.unNumber).toBe("");
    expect(reference.symbols).toBe("");
  });
});
