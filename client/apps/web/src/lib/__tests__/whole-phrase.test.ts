import { describe, expect, it } from "vitest";
import { containsWholePhrase, foldPhrase } from "../whole-phrase";

describe("foldPhrase", () => {
  it("folds case, curly apostrophes and runs of spaces", () => {
    expect(foldPhrase("  Acme’s   FOODS ")).toBe("acme's foods");
  });
});

describe("containsWholePhrase", () => {
  it("finds a phrase standing as whole words, in any case", () => {
    expect(containsWholePhrase("Tell me about seed-pay-002?", "SEED-PAY-002")).toBe(true);
    expect(containsWholePhrase("Is Harbor   Line Foods on hold?", "HARBOR Line Foods")).toBe(true);
    expect(containsWholePhrase("Acme's invoices", "Acme")).toBe(true);
  });

  it("does not find a phrase inside a longer word", () => {
    expect(containsWholePhrase("Acmeville yard", "Acme")).toBe(false);
    expect(containsWholePhrase("S-10011", "S-1001")).toBe(false);
  });

  it("finds a later whole-word occurrence after an earlier partial one", () => {
    expect(containsWholePhrase("Acmeville and Acme", "Acme")).toBe(true);
  });

  it("never finds an empty phrase", () => {
    expect(containsWholePhrase("anything", "  ")).toBe(false);
  });
});
