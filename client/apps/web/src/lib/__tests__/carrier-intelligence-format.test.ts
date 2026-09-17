import { describe, expect, it } from "vitest";
import {
  ageFromDays,
  carrierIntelDepthMerge,
  formatIntelValue,
  humanizeIntelFieldPath,
} from "../carrier-intelligence";

describe("formatIntelValue", () => {
  it("renders a missing or empty value as an em dash", () => {
    expect(formatIntelValue("safety.rating", null)).toBe("—");
    expect(formatIntelValue("safety.rating", undefined)).toBe("—");
    expect(formatIntelValue("safety.rating", "")).toBe("—");
    expect(formatIntelValue("safety.rating", "   ")).toBe("—");
    expect(formatIntelValue("safety.rating", "null")).toBe("—");
  });

  it("renders epoch seconds on date fields as a date instead of a number", () => {
    const formatted = formatIntelValue("safety.ratingDate", "843609600");

    expect(formatted).not.toBe("843609600");
    expect(formatted).toContain("1996");
    expect(formatIntelValue("insurance.pendingCancelAt", 1_788_220_800)).toContain("2026");
    expect(formatIntelValue("operations.mcs150At", "0")).toBe("—");
  });

  it("formats insurance amounts as currency", () => {
    expect(formatIntelValue("insurance.bipdOnFile", "750000")).toBe("$750,000.00");
    expect(formatIntelValue("insurance.cargoRequired", "100000.5")).toBe("$100,000.50");
    expect(formatIntelValue("insurance.cancelCount", "3")).toBe("3");
  });

  it("renders booleans as Yes and No, using the labels it is given", () => {
    expect(formatIntelValue("safety.outOfServiceOrder", "true")).toBe("Yes");
    expect(formatIntelValue("operations.hazmatCarrier", false)).toBe("No");
    expect(
      formatIntelValue("operations.hazmatCarrier", "false", { yes: "Sí", no: "No", empty: "-" }),
    ).toBe("No");
    expect(formatIntelValue("x", null, { yes: "Sí", no: "No", empty: "-" })).toBe("-");
  });

  it("renders ages in days as years, or months under a year", () => {
    expect(formatIntelValue("identity.dotAgeDays", "6600")).toMatch(/^18\s?yrs?$/);
    expect(formatIntelValue("authority.common.ageDays", "200")).toMatch(/^6\s?mths?$/);
  });

  it("turns enum codes on rating and status fields into words", () => {
    expect(formatIntelValue("safety.rating", "NotRated")).toBe("Not rated");
    expect(formatIntelValue("safety.riskScore", "VeryHigh")).toBe("Very high");
    expect(formatIntelValue("authority.common.status", "Active")).toBe("Active");
  });

  it("leaves identifiers and free text alone", () => {
    expect(formatIntelValue("identity.legalName", "McDonald Freight")).toBe("McDonald Freight");
    expect(formatIntelValue("identity.docketNumber", "0012345")).toBe("0012345");
    expect(formatIntelValue("contacts.email", '"ops@example.com"')).toBe("ops@example.com");
    expect(formatIntelValue(null, "Satisfactory")).toBe("Satisfactory");
  });
});

describe("ageFromDays", () => {
  it("floors whole years from days", () => {
    expect(ageFromDays(6570)).toEqual({ unit: "year", value: 18 });
    expect(ageFromDays(365)).toEqual({ unit: "year", value: 1 });
  });

  it("uses months under a year", () => {
    expect(ageFromDays(200)).toEqual({ unit: "month", value: 6 });
    expect(ageFromDays(10)).toEqual({ unit: "month", value: 0 });
  });

  it("returns null for missing or invalid values", () => {
    expect(ageFromDays(null)).toBeNull();
    expect(ageFromDays(undefined)).toBeNull();
    expect(ageFromDays(-1)).toBeNull();
  });
});

describe("humanizeIntelFieldPath", () => {
  it("reads the leaf of a path as a sentence", () => {
    expect(humanizeIntelFieldPath("insurance.pendingCancelAt")).toBe("Pending cancel at");
    expect(humanizeIntelFieldPath("fleet.powerUnits")).toBe("Power units");
  });

  it("names a BASIC measure after its category", () => {
    expect(humanizeIntelFieldPath("basics.unsafeDriving.percentile")).toBe(
      "Unsafe driving percentile",
    );
  });

  it("falls back to an em dash for an empty path", () => {
    expect(humanizeIntelFieldPath("")).toBe("—");
  });
});

describe("carrierIntelDepthMerge", () => {
  it("describes a snapshot whose latest fetch was shallower than the data it holds", () => {
    expect(
      carrierIntelDepthMerge({
        depth: "Full",
        depthFetchedAt: 1_000,
        fetchedDepth: "FMCSA",
        fetchedAt: 2_000,
      }),
    ).toEqual({ held: "Full", heldAt: 1_000, fetched: "FMCSA", fetchedAt: 2_000 });
  });

  it("stays quiet when the latest fetch matched the held depth or data is missing", () => {
    expect(
      carrierIntelDepthMerge({
        depth: "Full",
        depthFetchedAt: 1_000,
        fetchedDepth: "Full",
        fetchedAt: 1_000,
      }),
    ).toBeNull();
    expect(
      carrierIntelDepthMerge({
        depth: "Full",
        depthFetchedAt: null,
        fetchedDepth: "Lite",
        fetchedAt: 2_000,
      }),
    ).toBeNull();
  });
});
