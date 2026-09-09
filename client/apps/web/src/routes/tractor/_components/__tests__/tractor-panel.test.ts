import { tractorSchema } from "@/types/tractor";
import { describe, expect, it } from "vitest";
import { buildTractorDefaults } from "../tractor-panel";

describe("tractor form defaults", () => {
  it("starts a new tractor as an IFTA-qualified diesel unit", () => {
    const defaults = buildTractorDefaults();
    expect(defaults.fuelType).toBe("Diesel");
    expect(defaults.iftaQualified).toBe(true);
  });

  it("fills the IFTA fields when a record predates them", () => {
    const parsed = tractorSchema.safeParse({
      status: "Available",
      code: "118",
      equipmentTypeId: "et_1",
      equipmentManufacturerId: "em_1",
      primaryWorkerId: "wrk_1",
      secondaryWorkerId: null,
      fleetCodeId: null,
      stateId: null,
      externalId: "",
    });
    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);
    expect(parsed.data?.fuelType).toBe("Diesel");
    expect(parsed.data?.iftaQualified).toBe(true);
  });

  it("carries a non-diesel, non-qualified unit through unchanged", () => {
    const parsed = tractorSchema.safeParse({
      ...buildTractorDefaults(),
      code: "YARD-1",
      equipmentTypeId: "et_1",
      equipmentManufacturerId: "em_1",
      primaryWorkerId: "wrk_1",
      fuelType: "Gasoline",
      iftaQualified: false,
    });
    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);
    expect(parsed.data?.fuelType).toBe("Gasoline");
    expect(parsed.data?.iftaQualified).toBe(false);
  });

  it("refuses a fuel the IFTA matrix does not know", () => {
    const parsed = tractorSchema.safeParse({
      ...buildTractorDefaults(),
      code: "118",
      equipmentTypeId: "et_1",
      equipmentManufacturerId: "em_1",
      primaryWorkerId: "wrk_1",
      fuelType: "Kerosene",
    });
    expect(parsed.success).toBe(false);
    expect(parsed.error?.issues[0]?.path.join(".")).toBe("fuelType");
  });
});
