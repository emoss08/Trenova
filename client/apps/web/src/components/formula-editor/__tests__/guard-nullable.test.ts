import { describe, expect, it } from "vitest";
import { guardNullableField, scopeToFormPath } from "../guard-nullable";

describe("guardNullableField", () => {
  it("wraps a bare field reference", () => {
    expect(guardNullableField("weight * rate", "weight", "coalesce(weight, 0)")).toBe(
      "coalesce(weight, 0) * rate",
    );
  });

  it("wraps every occurrence", () => {
    expect(guardNullableField("weight + weight", "weight", "coalesce(weight, 0)")).toBe(
      "coalesce(weight, 0) + coalesce(weight, 0)",
    );
  });

  it("leaves an already guarded reference alone", () => {
    const expression = "coalesce(weight, 0) * rate + weight";
    expect(guardNullableField(expression, "weight", "coalesce(weight, 0)")).toBe(
      "coalesce(weight, 0) * rate + coalesce(weight, 0)",
    );
  });

  it("does not touch longer identifiers or other paths", () => {
    expect(guardNullableField("totalWeight * weightFactor", "weight", "coalesce(weight, 0)")).toBe(
      "totalWeight * weightFactor",
    );
    expect(guardNullableField("origin.zip + zip", "zip", 'coalesce(zip, "")')).toBe(
      'origin.zip + coalesce(zip, "")',
    );
  });

  it("handles dotted fields", () => {
    expect(
      guardNullableField(
        "customer.name == 'ACME' ? 1 : 0",
        "customer.name",
        'coalesce(customer.name, "")',
      ),
    ).toBe("coalesce(customer.name, \"\") == 'ACME' ? 1 : 0");
  });

  it("returns the input unchanged when nothing matches", () => {
    expect(guardNullableField("rate * 2", "weight", "coalesce(weight, 0)")).toBe("rate * 2");
  });
});

describe("scopeToFormPath", () => {
  it("converts bracket indices to dotted form paths", () => {
    expect(scopeToFormPath("expression")).toBe("expression");
    expect(scopeToFormPath("breakdownDefinitions[2].expression")).toBe(
      "breakdownDefinitions.2.expression",
    );
  });
});
