import { describe, expect, it } from "vitest";
import {
  MAX_PINNED_FACT_LENGTH,
  MAX_PINNED_FACTS,
  pinFact,
  unpinFact,
} from "../composer/desk-facts-state";

describe("pinFact", () => {
  it("pins a typed fact at the end, tidied", () => {
    expect(pinFact(["Acme pays net 45"], "  Invoice   date is Oct 3 ")).toEqual([
      "Acme pays net 45",
      "Invoice date is Oct 3",
    ]);
  });

  it("leaves the list as it was for blank input or a fact already pinned", () => {
    const facts = ["Invoice date is Oct 3"];

    expect(pinFact(facts, "   ")).toBe(facts);
    expect(pinFact(facts, "Invoice date is Oct 3")).toBe(facts);
  });

  it("stops at the cap and cuts a long fact to the limit", () => {
    const full = Array.from({ length: MAX_PINNED_FACTS }, (_, i) => `Fact ${i}`);

    expect(pinFact(full, "One more")).toBe(full);
    expect(pinFact([], "x".repeat(MAX_PINNED_FACT_LENGTH + 20))[0]).toHaveLength(
      MAX_PINNED_FACT_LENGTH,
    );
  });
});

describe("unpinFact", () => {
  it("removes one fact and keeps the order of the rest", () => {
    expect(unpinFact(["a", "b", "c"], "b")).toEqual(["a", "c"]);
  });

  it("leaves the list as it was for a fact not pinned", () => {
    const facts = ["a"];

    expect(unpinFact(facts, "z")).toBe(facts);
  });
});
