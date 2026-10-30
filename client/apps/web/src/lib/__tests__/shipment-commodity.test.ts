import { describe, expect, it } from "vitest";
import { commodityFromOption, commodityTotals } from "../shipment-commodity";

describe("commodityFromOption", () => {
  // `SelectOption` as the commodity select options resolver returns it, its meta from
  // `commoditySelectOption` in services/tms.
  it("carries the picked commodity's name, description and handling flags", () => {
    expect(
      commodityFromOption({
        id: "com_1",
        label: "Lithium batteries",
        description: "Boxed cells",
        meta: { hazardousMaterialId: "hm_1", stackable: true, fragile: false },
      }),
    ).toEqual({
      id: "com_1",
      name: "Lithium batteries",
      description: "Boxed cells",
      hazardousMaterialId: "hm_1",
      stackable: true,
      fragile: false,
    });
  });

  it("reads a commodity with no hazardous material, and an option with no meta, as plain", () => {
    expect(
      commodityFromOption({
        id: "com_2",
        label: "Paper rolls",
        description: null,
        meta: { hazardousMaterialId: "", stackable: false, fragile: true },
      }),
    ).toEqual({
      id: "com_2",
      name: "Paper rolls",
      description: "",
      hazardousMaterialId: null,
      stackable: false,
      fragile: true,
    });
    expect(
      commodityFromOption({ id: "com_3", label: "Lumber", description: null, meta: null }),
    ).toEqual({
      id: "com_3",
      name: "Lumber",
      description: "",
      hazardousMaterialId: null,
      stackable: false,
      fragile: false,
    });
  });
});

describe("commodityTotals", () => {
  it("adds every line's pieces and weight", () => {
    expect(
      commodityTotals([
        { pieces: 2, weight: 1000 },
        { pieces: 1, weight: 500 },
      ]),
    ).toEqual({ pieces: 3, weight: 1500 });
  });

  it("counts a line still being typed as nothing", () => {
    expect(
      commodityTotals([
        { pieces: 4, weight: 250 },
        { pieces: undefined as unknown as number, weight: undefined as unknown as number },
      ]),
    ).toEqual({ pieces: 4, weight: 250 });
  });

  it("is zero for no lines", () => {
    expect(commodityTotals([])).toEqual({ pieces: 0, weight: 0 });
  });
});
