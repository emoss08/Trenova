import { describe, expect, it } from "vitest";
import { findHoveredIdentifier, formatHoverValue } from "../expr-hover";

describe("findHoveredIdentifier", () => {
  const doc = "coalesce(weight, 0) * customer.rate + fuel";

  it("returns the identifier under the cursor with its span", () => {
    expect(findHoveredIdentifier(doc, doc.indexOf("weight") + 2)).toEqual({
      name: "weight",
      from: doc.indexOf("weight"),
      to: doc.indexOf("weight") + "weight".length,
    });
  });

  it("treats a dotted path as one identifier", () => {
    const start = doc.indexOf("customer.rate");
    expect(findHoveredIdentifier(doc, start + 10)?.name).toBe("customer.rate");
  });

  it("ignores function names, operators, and whitespace", () => {
    expect(findHoveredIdentifier(doc, 2)).toBeNull();
    expect(findHoveredIdentifier(doc, doc.indexOf("*"))).toBeNull();
    expect(findHoveredIdentifier(doc, doc.indexOf(" +"))).toBeNull();
  });
});

describe("formatHoverValue", () => {
  it("shows the value and where it came from", () => {
    expect(formatHoverValue({ name: "weight", value: 12000, source: "field" })).toBe(
      "weight = 12000 (from shipment)",
    );
    expect(formatHoverValue({ name: "rate", value: 2.5, source: "override" })).toBe(
      "rate = 2.5 (engine override)",
    );
    expect(formatHoverValue({ name: "code", value: "ACME", source: "sample" })).toBe(
      'code = "ACME" (sample data)',
    );
    expect(formatHoverValue({ name: "weight", value: null, source: "field" })).toBe(
      "weight = empty (from shipment)",
    );
  });
});
