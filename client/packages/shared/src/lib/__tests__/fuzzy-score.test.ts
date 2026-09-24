import { describe, expect, it } from "vitest";
import { fuzzyScore, rankByFuzzyScore } from "../fuzzy-score";

describe("fuzzyScore", () => {
  it("matches everything equally on an empty query", () => {
    expect(fuzzyScore("", [{ text: "Shipments" }])).toBe(1);
    expect(fuzzyScore("   ", [{ text: "Shipments" }])).toBe(1);
  });

  it("ranks exact over prefix over word prefix over substring over subsequence", () => {
    const exact = fuzzyScore("fleet", [{ text: "Fleet" }]);
    const prefix = fuzzyScore("fle", [{ text: "Fleet codes" }]);
    const wordPrefix = fuzzyScore("cod", [{ text: "Fleet codes" }]);
    const substring = fuzzyScore("eet", [{ text: "Fleet codes" }]);
    const subsequence = fuzzyScore("fcd", [{ text: "Fleet codes" }]);

    expect(exact).toBeGreaterThan(prefix);
    expect(prefix).toBeGreaterThan(wordPrefix);
    expect(wordPrefix).toBeGreaterThan(substring);
    expect(substring).toBeGreaterThan(subsequence);
    expect(subsequence).toBeGreaterThan(0);
  });

  it("requires every term to land somewhere", () => {
    expect(fuzzyScore("new cust", [{ text: "New customer" }])).toBeGreaterThan(0);
    expect(fuzzyScore("new cust", [{ text: "New shipment" }])).toBe(0);
  });

  it("lets terms match different fields", () => {
    expect(
      fuzzyScore("billing inv", [
        { text: "Invoices" },
        { text: "Billing > Invoices", weight: 0.6 },
      ]),
    ).toBeGreaterThan(0);
  });

  it("scales a match by the field's weight", () => {
    const title = fuzzyScore("ship", [{ text: "Shipments" }]);
    const breadcrumb = fuzzyScore("ship", [{ text: "Shipments", weight: 0.5 }]);

    expect(title).toBeGreaterThan(breadcrumb);
  });

  it("ignores case and accents", () => {
    expect(fuzzyScore("JOSE", [{ text: "José Álvarez" }])).toBeGreaterThan(0);
  });

  it("matches scattered letters only in fields that allow it", () => {
    expect(fuzzyScore("acme", [{ text: "Dispatch management" }])).toBeGreaterThan(0);
    expect(fuzzyScore("acme", [{ text: "Dispatch management", subsequence: false }])).toBe(0);
    expect(
      fuzzyScore("disp", [{ text: "Dispatch management", subsequence: false }]),
    ).toBeGreaterThan(0);
  });

  it("does not treat short scattered letters as a match", () => {
    expect(fuzzyScore("fc", [{ text: "Fleet codes" }])).toBe(0);
  });
});

describe("rankByFuzzyScore", () => {
  const entries = ["Shipments", "Shipment types", "Hazmat segregation rules", "Fleet codes"];

  it("keeps matches only, best first", () => {
    expect(rankByFuzzyScore("ship", entries, (entry) => [{ text: entry }])).toEqual([
      "Shipments",
      "Shipment types",
    ]);
  });

  it("returns everything in its original order for an empty query", () => {
    expect(rankByFuzzyScore("", entries, (entry) => [{ text: entry }])).toEqual(entries);
  });

  it("keeps the original order between equal scores", () => {
    expect(rankByFuzzyScore("s", ["Sb", "Sa"], (entry) => [{ text: entry }])).toEqual(["Sb", "Sa"]);
  });
});
