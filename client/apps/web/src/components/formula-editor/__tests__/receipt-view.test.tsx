import type { FormulaReceipt } from "@trenova/shared/types/formula-template";
import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { describeLookupMatch, ReceiptView } from "../receipt-view";

const receipt: FormulaReceipt = {
  variables: [
    { name: "totalDistance", value: 612, source: "field" },
    { name: "baseRate", value: 2.15, source: "override" },
    { name: "fuelPct", value: 18, source: "default" },
    { name: "customer.code", value: "ACME", source: "field" },
  ],
  lookups: [
    {
      scope: "expression",
      table: "miles",
      keys: [612],
      value: 2,
      match: { bandMin: 500, bandMax: null },
    },
    {
      scope: "fuel",
      table: "fsc",
      keys: ["DIESEL"],
      value: 0.35,
      match: { matchedKey: "DIESEL" },
    },
  ],
  rawAmount: 1315.8,
  versionNumber: 4,
  effectiveFrom: null,
  durationMicros: 850,
};

describe("describeLookupMatch", () => {
  it("names an exact key, a closed band, and an open-ended band", () => {
    expect(describeLookupMatch({ matchedKey: "DIESEL" })).toBe("key DIESEL");
    expect(describeLookupMatch({ bandMin: 0, bandMax: 500 })).toBe("band 0–500");
    expect(describeLookupMatch({ bandMin: 500, bandMax: null })).toBe("band 500+");
    expect(describeLookupMatch(undefined)).toBe("no match");
  });
});

describe("ReceiptView", () => {
  it("shows every variable with where it came from, and every table row consulted", () => {
    const { container, getByText } = render(<ReceiptView receipt={receipt} />);
    fireEvent.click(getByText(/Variables \(/));
    const text = container.textContent ?? "";

    expect(text).toContain("totalDistance");
    expect(text).toContain("override");
    expect(text).toContain("default");
    expect(text).toContain("customer.code");
    expect(text).toContain("miles");
    expect(text).toContain("band 500+");
    expect(text).toContain("key DIESEL");
    expect(text).toContain("fuel");
    expect(text).toContain("v4");
  });

  it("offers the resolved values back when asked", () => {
    const onUseValues = vi.fn();
    const { getByText } = render(<ReceiptView receipt={receipt} onUseValues={onUseValues} />);
    fireEvent.click(getByText("Use these values"));
    expect(onUseValues).toHaveBeenCalledWith({
      totalDistance: 612,
      baseRate: 2.15,
      fuelPct: 18,
      "customer.code": "ACME",
    });
  });
});

describe("describeLookupMatch with adjusted keys", () => {
  it("says when the key was moved into the band that priced it", () => {
    expect(describeLookupMatch({ bandMin: 1000, bandMax: 5000, adjusted: true })).toBe(
      "band 1000–5000 (key moved into band)",
    );
    expect(describeLookupMatch({ bandMin: 1000, bandMax: 5000 })).toBe("band 1000–5000");
  });
});
