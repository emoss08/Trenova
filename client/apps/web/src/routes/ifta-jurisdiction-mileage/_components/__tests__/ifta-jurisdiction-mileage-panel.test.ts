import type { IftaMileageEntryRow } from "@/lib/graphql/ifta-jurisdiction-mileage";
import { createIftaMileageEntryFormSchema } from "@trenova/shared/types/ifta-jurisdiction-mileage";
import { describe, expect, it } from "vitest";
import {
  buildIftaMileageEntryDefaults,
  isComputedEntry,
  toIftaMileageEntryInput,
} from "../ifta-jurisdiction-mileage-panel";

const TODAY = 1_760_000_000;

const row: IftaMileageEntryRow = {
  id: "ijme_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  tractorId: "trk_118",
  jurisdictionId: "ij_tx",
  traveledAt: TODAY - 86_400,
  year: 2025,
  quarter: 4,
  miles: "412.50",
  loaded: false,
  source: "Manual",
  shipmentMoveId: null,
  notes: "Deadhead from Amarillo to the yard",
  createdById: "usr_1",
  version: 1,
  createdAt: 1,
  updatedAt: 2,
  tractor: { id: "trk_118", code: "118" },
  jurisdiction: { id: "ij_tx", countryCode: "US", code: "TX", name: "Texas" },
};

const schema = createIftaMileageEntryFormSchema(TODAY);

describe("IFTA mileage entry form mapping", () => {
  it("starts a new entry today, loaded, with nothing else filled", () => {
    expect(buildIftaMileageEntryDefaults(null, TODAY)).toEqual({
      tractorId: "",
      jurisdictionId: "",
      traveledAt: TODAY,
      miles: "",
      loaded: true,
      notes: null,
    });
  });

  it("round-trips a server row through the form and back into a mutation input", () => {
    const defaults = buildIftaMileageEntryDefaults(row, TODAY);
    const parsed = schema.safeParse(defaults);
    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);

    expect(toIftaMileageEntryInput(defaults)).toEqual({
      tractorId: "trk_118",
      jurisdictionId: "ij_tx",
      traveledAt: TODAY - 86_400,
      miles: "412.50",
      loaded: false,
      source: "Manual",
      shipmentMoveId: null,
      notes: "Deadhead from Amarillo to the yard",
    });
  });

  it("sends null for cleared notes and trims the miles", () => {
    const input = toIftaMileageEntryInput({
      ...buildIftaMileageEntryDefaults(row, TODAY),
      miles: " 12.5 ",
      notes: "   ",
    });
    expect(input.miles).toBe("12.5");
    expect(input.notes).toBeNull();
  });

  it("refuses travel in the future, zero miles and more than two decimals", () => {
    const base = buildIftaMileageEntryDefaults(row, TODAY);
    const failures: Array<[Partial<typeof base>, string]> = [
      [{ traveledAt: TODAY + 1 }, "traveledAt"],
      [{ miles: "0" }, "miles"],
      [{ miles: "0.00" }, "miles"],
      [{ miles: "-12" }, "miles"],
      [{ miles: "12.345" }, "miles"],
      [{ miles: "" }, "miles"],
      [{ tractorId: "" }, "tractorId"],
      [{ jurisdictionId: "" }, "jurisdictionId"],
      [{ notes: "x".repeat(501) }, "notes"],
    ];
    for (const [override, path] of failures) {
      const result = schema.safeParse({ ...base, ...override });
      expect(result.success, JSON.stringify(override)).toBe(false);
      expect(
        result.error?.issues.map((issue) => issue.path.join(".")),
        JSON.stringify(override),
      ).toContain(path);
    }
  });

  it("accepts travel exactly today and whole miles", () => {
    const base = buildIftaMileageEntryDefaults(row, TODAY);
    expect(schema.safeParse({ ...base, traveledAt: TODAY }).success).toBe(true);
    expect(schema.safeParse({ ...base, miles: "7" }).success).toBe(true);
    expect(schema.safeParse({ ...base, notes: null }).success).toBe(true);
  });

  it("treats only rows the system wrote as computed and read-only", () => {
    expect(isComputedEntry(row)).toBe(false);
    expect(isComputedEntry({ ...row, source: "RouteCalculation" })).toBe(true);
    expect(isComputedEntry({ ...row, source: "Telematics" })).toBe(true);
  });

  it("maps a computed row into the form unchanged so it can be shown, not edited", () => {
    const computed = { ...row, source: "Telematics" as const, shipmentMoveId: "smv_1" };
    const defaults = buildIftaMileageEntryDefaults(computed, TODAY);
    expect(defaults.miles).toBe("412.50");
    expect(defaults.tractorId).toBe("trk_118");
    expect(toIftaMileageEntryInput(defaults, computed)).toEqual({
      tractorId: "trk_118",
      jurisdictionId: "ij_tx",
      traveledAt: TODAY - 86_400,
      miles: "412.50",
      loaded: false,
      source: "Telematics",
      shipmentMoveId: "smv_1",
      notes: "Deadhead from Amarillo to the yard",
    });
  });
});
