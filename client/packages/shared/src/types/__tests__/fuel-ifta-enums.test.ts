import { describe, expect, it } from "vitest";
import {
  FUEL_CARD_PROVIDER_LABELS,
  FUEL_CARD_STATUS_LABELS,
  FUEL_PURCHASE_IMPORT_STATUS_LABELS,
  FUEL_PURCHASE_SOURCE_LABELS,
  FUEL_QUANTITY_UNIT_LABELS,
  IFTA_FUEL_TYPE_LABELS,
  IFTA_MILEAGE_SOURCE_LABELS,
  IFTA_QUARTER_LABELS,
  IFTA_RETURN_STATUS_LABELS,
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

// These unions mirror the GraphQL enums IFTAFuelType, FuelCardProvider,
// FuelCardStatus, FuelQuantityUnit, FuelPurchaseSource,
// FuelPurchaseImportStatus, IFTAMileageSource and IFTAReturnStatus. The
// values are the wire values; the label maps are what people read.

describe("fuel and IFTA enum schemas", () => {
  it("lists every IFTA fuel type the return can carry", () => {
    expect(iftaFuelTypeSchema.options).toEqual([
      "Diesel",
      "Gasoline",
      "Gasohol",
      "Propane",
      "CNG",
      "LNG",
      "Ethanol",
      "Methanol",
      "E85",
      "M85",
      "A55",
      "Biodiesel",
      "Electricity",
      "Hydrogen",
      "DEF",
      "Reefer",
      "Other",
    ]);
  });

  it("matches the wire values of the smaller enums", () => {
    expect(fuelCardProviderSchema.options).toEqual(["Comdata", "EFS", "WEX", "Other"]);
    expect(fuelCardStatusSchema.options).toEqual(["Active", "Suspended", "Cancelled"]);
    expect(fuelQuantityUnitSchema.options).toEqual(["Gallon", "Litre"]);
    expect(fuelPurchaseSourceSchema.options).toEqual(["Manual", "CardImport"]);
    expect(fuelPurchaseImportStatusSchema.options).toEqual([
      "Pending",
      "Parsed",
      "Committed",
      "Failed",
      "Discarded",
    ]);
    expect(iftaMileageSourceSchema.options).toEqual(["Manual", "RouteCalculation", "Telematics"]);
    expect(iftaReturnStatusSchema.options).toEqual(["Draft", "Finalized", "Filed"]);
    expect(iftaQuarterSchema.options).toEqual(["1", "2", "3", "4"]);
  });

  it("rejects values the server does not know", () => {
    expect(iftaFuelTypeSchema.safeParse("Kerosene").success).toBe(false);
    expect(fuelCardStatusSchema.safeParse("Canceled").success).toBe(false);
    expect(iftaQuarterSchema.safeParse("5").success).toBe(false);
  });
});

describe("fuel and IFTA labels", () => {
  it("cover every value with readable text", () => {
    const pairs: ReadonlyArray<[readonly string[], Record<string, string>]> = [
      [iftaFuelTypeSchema.options, IFTA_FUEL_TYPE_LABELS],
      [fuelCardProviderSchema.options, FUEL_CARD_PROVIDER_LABELS],
      [fuelCardStatusSchema.options, FUEL_CARD_STATUS_LABELS],
      [fuelQuantityUnitSchema.options, FUEL_QUANTITY_UNIT_LABELS],
      [fuelPurchaseSourceSchema.options, FUEL_PURCHASE_SOURCE_LABELS],
      [fuelPurchaseImportStatusSchema.options, FUEL_PURCHASE_IMPORT_STATUS_LABELS],
      [iftaMileageSourceSchema.options, IFTA_MILEAGE_SOURCE_LABELS],
      [iftaReturnStatusSchema.options, IFTA_RETURN_STATUS_LABELS],
      [iftaQuarterSchema.options, IFTA_QUARTER_LABELS],
    ];
    for (const [options, labels] of pairs) {
      expect(Object.keys(labels).sort()).toEqual([...options].sort());
      for (const option of options) {
        expect(labels[option]?.trim().length).toBeGreaterThan(0);
      }
    }
  });

  it("spell out the values that are not words", () => {
    expect(IFTA_FUEL_TYPE_LABELS.CNG).toBe("Compressed natural gas (CNG)");
    expect(IFTA_FUEL_TYPE_LABELS.LNG).toBe("Liquefied natural gas (LNG)");
    expect(IFTA_FUEL_TYPE_LABELS.DEF).toBe("Diesel exhaust fluid (DEF)");
    expect(IFTA_FUEL_TYPE_LABELS.E85).toBe("E85 (85% ethanol)");
    expect(IFTA_FUEL_TYPE_LABELS.M85).toBe("M85 (85% methanol)");
    expect(IFTA_FUEL_TYPE_LABELS.A55).toBe("A55 (55% methanol)");
    expect(IFTA_FUEL_TYPE_LABELS.Reefer).toBe("Reefer fuel");
    expect(FUEL_PURCHASE_SOURCE_LABELS.CardImport).toBe("Card import");
    expect(IFTA_MILEAGE_SOURCE_LABELS.RouteCalculation).toBe("Route calculation");
    expect(IFTA_QUARTER_LABELS["1"]).toBe("Q1 (Jan–Mar)");
    expect(IFTA_QUARTER_LABELS["4"]).toBe("Q4 (Oct–Dec)");
  });
});
