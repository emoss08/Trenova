import type { SelectOption } from "@/lib/graphql/select-options";
import {
  selectOptionMetaBoolean,
  selectOptionMetaNumber,
  selectOptionMetaString,
} from "@/lib/select-option-meta";
import { describe, expect, it } from "vitest";

function option(meta: Record<string, unknown> | null): SelectOption {
  return { id: "payc_1", label: "LH", description: null, meta };
}

describe("selectOptionMetaString", () => {
  it("returns the string a resource put on the option", () => {
    expect(selectOptionMetaString(option({ direction: "Earning" }), "direction")).toBe("Earning");
  });

  it("returns empty for a key the resource does not send", () => {
    expect(selectOptionMetaString(option({}), "direction")).toBe("");
    expect(selectOptionMetaString(option(null), "direction")).toBe("");
  });

  // Meta is untyped JSON: a value of the wrong shape has to read as absent, not
  // reach a template as "[object Object]" or "42".
  it("returns empty for a value that is not a string", () => {
    expect(selectOptionMetaString(option({ direction: 42 }), "direction")).toBe("");
    expect(selectOptionMetaString(option({ direction: { a: 1 } }), "direction")).toBe("");
    expect(selectOptionMetaString(option({ direction: null }), "direction")).toBe("");
  });
});

describe("selectOptionMetaNumber", () => {
  it("returns the number a resource put on the option", () => {
    expect(selectOptionMetaNumber(option({ validityMonths: 36 }), "validityMonths")).toBe(36);
  });

  // Zero is a real answer for a duration, so it must survive rather than be
  // flattened into the null that means "the server did not say".
  it("keeps zero distinct from absent", () => {
    expect(selectOptionMetaNumber(option({ validityMonths: 0 }), "validityMonths")).toBe(0);
    expect(selectOptionMetaNumber(option({}), "validityMonths")).toBeNull();
    expect(selectOptionMetaNumber(option({ validityMonths: null }), "validityMonths")).toBeNull();
  });

  it("returns null for a numeric string", () => {
    expect(selectOptionMetaNumber(option({ validityMonths: "36" }), "validityMonths")).toBeNull();
  });
});

describe("selectOptionMetaBoolean", () => {
  it("is true only for an actual true", () => {
    expect(selectOptionMetaBoolean(option({ taxable: true }), "taxable")).toBe(true);
    expect(selectOptionMetaBoolean(option({ taxable: false }), "taxable")).toBe(false);
    expect(selectOptionMetaBoolean(option({ taxable: "true" }), "taxable")).toBe(false);
    expect(selectOptionMetaBoolean(option({ taxable: 1 }), "taxable")).toBe(false);
    expect(selectOptionMetaBoolean(option(null), "taxable")).toBe(false);
  });
});
