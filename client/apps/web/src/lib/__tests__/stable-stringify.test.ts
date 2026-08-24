import { describe, expect, it } from "vitest";
import { stableStringify } from "../stable-stringify";

describe("stableStringify", () => {
  it("is invariant to object key order, including nested objects", () => {
    const a = { fieldFilters: [{ field: "x", operator: "eq", value: { from: 1, to: 2 } }] };
    const b = { fieldFilters: [{ value: { to: 2, from: 1 }, operator: "eq", field: "x" }] };
    expect(stableStringify(a)).toBe(stableStringify(b));
  });

  it("preserves array order", () => {
    expect(stableStringify([1, 2])).not.toBe(stableStringify([2, 1]));
  });

  it("distinguishes null, absent, and empty values", () => {
    expect(stableStringify({ a: null })).not.toBe(stableStringify({}));
    expect(stableStringify({ a: "" })).not.toBe(stableStringify({ a: null }));
    expect(stableStringify([])).not.toBe(stableStringify({}));
  });
});
