import { describe, expect, it } from "vitest";
import {
  fuelCardProviderChoices,
  fuelCardStatusChoices,
  fuelPurchaseImportStatusChoices,
  fuelPurchaseSourceChoices,
  fuelQuantityUnitChoices,
  iftaFuelTypeChoices,
  iftaMileageSourceChoices,
  iftaQuarterChoices,
  iftaReturnStatusChoices,
} from "@/lib/choices";
import {
  FUEL_CARD_PROVIDER_LABELS,
  IFTA_FUEL_TYPE_LABELS,
  fuelCardProviderSchema,
  fuelCardStatusSchema,
  fuelPurchaseImportStatusSchema,
  fuelPurchaseSourceSchema,
  fuelQuantityUnitSchema,
  iftaFuelTypeSchema,
  iftaMileageSourceSchema,
  iftaQuarterSchema,
  iftaReturnStatusSchema,
} from "@trenova/shared/types/fuel-ifta-enums";

// Choices feed selects and column filters. Every enum value must be
// selectable, in the enum's own order, under the label the rest of the app
// uses for it.

function values(choices: ReadonlyArray<{ value: string }>) {
  return choices.map((choice) => choice.value);
}

describe("fuel and IFTA choices", () => {
  it("offer every value of their enum in order", () => {
    expect(values(fuelCardProviderChoices)).toEqual(fuelCardProviderSchema.options);
    expect(values(fuelCardStatusChoices)).toEqual(fuelCardStatusSchema.options);
    expect(values(fuelPurchaseSourceChoices)).toEqual(fuelPurchaseSourceSchema.options);
    expect(values(iftaFuelTypeChoices)).toEqual(iftaFuelTypeSchema.options);
    expect(values(fuelQuantityUnitChoices)).toEqual(fuelQuantityUnitSchema.options);
    expect(values(iftaReturnStatusChoices)).toEqual(iftaReturnStatusSchema.options);
    expect(values(iftaQuarterChoices)).toEqual(iftaQuarterSchema.options);
    expect(values(iftaMileageSourceChoices)).toEqual(iftaMileageSourceSchema.options);
    expect(values(fuelPurchaseImportStatusChoices)).toEqual(fuelPurchaseImportStatusSchema.options);
  });

  it("take their labels from the shared label maps", () => {
    for (const choice of iftaFuelTypeChoices) {
      expect(choice.label).toBe(IFTA_FUEL_TYPE_LABELS[choice.value]);
    }
    for (const choice of fuelCardProviderChoices) {
      expect(choice.label).toBe(FUEL_CARD_PROVIDER_LABELS[choice.value]);
    }
    expect(fuelPurchaseSourceChoices.find((c) => c.value === "CardImport")?.label).toBe(
      "Card import",
    );
    expect(iftaMileageSourceChoices.find((c) => c.value === "RouteCalculation")?.label).toBe(
      "Route calculation",
    );
    expect(iftaQuarterChoices.map((c) => c.label)).toEqual([
      "Q1 (Jan–Mar)",
      "Q2 (Apr–Jun)",
      "Q3 (Jul–Sep)",
      "Q4 (Oct–Dec)",
    ]);
  });

  it("colour the statuses that appear in filters", () => {
    expect(fuelCardStatusChoices).toEqual([
      { label: "Active", value: "Active", color: "#15803d" },
      { label: "Suspended", value: "Suspended", color: "#b45309" },
      { label: "Cancelled", value: "Cancelled", color: "#dc2626" },
    ]);
    expect(iftaReturnStatusChoices.map((c) => c.color)).toEqual(["#6b7280", "#1d4ed8", "#15803d"]);
    for (const choice of fuelPurchaseImportStatusChoices) {
      expect(choice.color).toMatch(/^#[0-9a-f]{6}$/);
    }
  });
});
