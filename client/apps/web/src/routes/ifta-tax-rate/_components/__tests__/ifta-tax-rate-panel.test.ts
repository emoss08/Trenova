import type { IftaTaxRateRow } from "@/lib/graphql/ifta-tax-rate";
import { iftaTaxRateFormSchema } from "@trenova/shared/types/ifta-tax-rate";
import { describe, expect, it } from "vitest";
import { buildIftaTaxRateDefaults, toIftaTaxRateInput } from "../ifta-tax-rate-panel";

const row: IftaTaxRateRow = {
  id: "itr_1",
  jurisdictionId: "ij_in",
  year: 2026,
  quarter: 3,
  fuelType: "Diesel",
  ratePerGallon: "0.5700",
  surchargeRatePerGallon: "0.1100",
  sourceNote: "IFTA Inc. matrix 2026Q3",
  sourceUrl: "https://www.iftach.org/taxmatrix4/",
  version: 1,
  createdAt: 1,
  updatedAt: 2,
  jurisdiction: {
    id: "ij_in",
    countryCode: "US",
    code: "IN",
    name: "Indiana",
    hasSurcharge: true,
    isIftaMember: true,
  },
};

describe("IFTA tax rate form mapping", () => {
  it("starts a new rate in the period the page is on, for diesel, with no surcharge", () => {
    expect(buildIftaTaxRateDefaults(null, { year: 2026, quarter: 2 })).toEqual({
      jurisdictionId: "",
      year: 2026,
      quarter: "2",
      fuelType: "Diesel",
      ratePerGallon: "",
      surchargeRatePerGallon: null,
      sourceNote: null,
      sourceUrl: null,
    });
  });

  it("round-trips a server row, turning the quarter string back into an integer", () => {
    const defaults = buildIftaTaxRateDefaults(row, { year: 2025, quarter: 1 });
    expect(defaults.quarter).toBe("3");
    const parsed = iftaTaxRateFormSchema.safeParse(defaults);
    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);

    expect(toIftaTaxRateInput(defaults)).toEqual({
      jurisdictionId: "ij_in",
      year: 2026,
      quarter: 3,
      fuelType: "Diesel",
      ratePerGallon: "0.5700",
      surchargeRatePerGallon: "0.1100",
      sourceNote: "IFTA Inc. matrix 2026Q3",
      sourceUrl: "https://www.iftach.org/taxmatrix4/",
    });
  });

  it("sends a zero surcharge when the form leaves it blank", () => {
    const input = toIftaTaxRateInput({
      ...buildIftaTaxRateDefaults(row, { year: 2026, quarter: 3 }),
      surchargeRatePerGallon: null,
      sourceNote: "",
      sourceUrl: "  ",
    });
    expect(input.surchargeRatePerGallon).toBe("0");
    expect(input.sourceNote).toBeNull();
    expect(input.sourceUrl).toBeNull();
  });

  it("rejects what the matrix cannot hold, naming the field", () => {
    const base = buildIftaTaxRateDefaults(row, { year: 2026, quarter: 3 });
    const failures: Array<[Partial<typeof base>, string]> = [
      [{ jurisdictionId: "" }, "jurisdictionId"],
      [{ quarter: "5" as never }, "quarter"],
      [{ year: 1999 }, "year"],
      [{ year: 2026.5 }, "year"],
      [{ ratePerGallon: "" }, "ratePerGallon"],
      [{ ratePerGallon: "-0.1" }, "ratePerGallon"],
      [{ ratePerGallon: "0.12345" }, "ratePerGallon"],
      [{ surchargeRatePerGallon: "-0.01" }, "surchargeRatePerGallon"],
      [{ surchargeRatePerGallon: "abc" }, "surchargeRatePerGallon"],
      [{ sourceUrl: "not a url" }, "sourceUrl"],
    ];
    for (const [override, path] of failures) {
      const result = iftaTaxRateFormSchema.safeParse({ ...base, ...override });
      expect(result.success, JSON.stringify(override)).toBe(false);
      expect(
        result.error?.issues.map((issue) => issue.path.join(".")),
        JSON.stringify(override),
      ).toContain(path);
    }
  });

  it("allows a zero rate, which some jurisdictions publish", () => {
    const base = buildIftaTaxRateDefaults(row, { year: 2026, quarter: 3 });
    expect(iftaTaxRateFormSchema.safeParse({ ...base, ratePerGallon: "0" }).success).toBe(true);
  });
});
